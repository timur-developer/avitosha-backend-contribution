package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	backendauth "github.com/guitaramust-sudo/Avitosha/app/backend/internal/auth"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/client/grpc/authclient"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/client/grpc/gameclient"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/config"
	avitoshav1 "github.com/guitaramust-sudo/Avitosha/app/backend/internal/gen/avitosha/v1"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/handler"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/realtime"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/repository/postgres"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/transport/grpc/authserver"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/transport/grpc/gameserver"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
)

const downstreamUnaryTimeout = 8 * time.Second

type GatewayApp struct {
	logger          *slog.Logger
	server          *http.Server
	connections     []*grpc.ClientConn
	shutdownTimeout time.Duration
}

func NewGateway(cfg config.Config, logger *slog.Logger) (*GatewayApp, error) {
	clientOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(defaultUnaryDeadline),
	}
	authConnection, err := grpc.NewClient(cfg.AuthGRPCAddr, clientOptions...)
	if err != nil {
		return nil, fmt.Errorf("connect auth service: %w", err)
	}
	gameConnection, err := grpc.NewClient(cfg.GameGRPCAddr, clientOptions...)
	if err != nil {
		_ = authConnection.Close()
		return nil, fmt.Errorf("connect game service: %w", err)
	}

	auth := authclient.New(avitoshav1.NewAuthServiceClient(authConnection))
	game := gameclient.New(avitoshav1.NewGameServiceClient(gameConnection))
	ready := grpcServicesPinger{
		healthv1.NewHealthClient(authConnection), healthv1.NewHealthClient(gameConnection),
	}
	photoUploads, err := newPhotoUploadService(cfg)
	if err != nil {
		_ = authConnection.Close()
		_ = gameConnection.Close()
		return nil, fmt.Errorf("create photo upload service: %w", err)
	}
	router := handler.NewRouter(handler.RouterDependencies{
		Logger: logger, DB: ready, AuthService: auth, AccessTokenAuthenticator: auth, AppEnv: cfg.AppEnv,
		FrontendOrigin: cfg.FrontendOrigin, RefreshTokenTTL: cfg.RefreshTokenTTL,
		SecureRefreshCookie: cfg.AppEnv == config.AppEnvProd, GameService: game, MarketplaceService: game, EventHub: game,
		PhotoUploadService: photoUploads,
	})
	return &GatewayApp{
		logger: logger, connections: []*grpc.ClientConn{authConnection, gameConnection},
		shutdownTimeout: cfg.ShutdownTimeout,
		server: &http.Server{Addr: cfg.HTTPAddr, Handler: router, ReadTimeout: cfg.HTTPReadTimeout,
			WriteTimeout: cfg.HTTPWriteTimeout, IdleTimeout: cfg.HTTPIdleTimeout},
	}, nil
}

func (a *GatewayApp) Run(ctx context.Context) error {
	runCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	defer func() {
		for _, connection := range a.connections {
			_ = connection.Close()
		}
	}()
	errChannel := make(chan error, 1)
	go func() {
		a.logger.Info("api gateway started", "addr", a.server.Addr)
		err := a.server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		errChannel <- err
	}()
	select {
	case err := <-errChannel:
		if err != nil {
			return fmt.Errorf("serve api gateway: %w", err)
		}
		return nil
	case <-runCtx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
	defer cancel()
	if err := a.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown api gateway: %w", err)
	}
	return <-errChannel
}

type GRPCServiceApp struct {
	name            string
	addr            string
	logger          *slog.Logger
	server          *grpc.Server
	db              *pgxpool.Pool
	shutdownTimeout time.Duration
}

func NewAuthGRPCService(ctx context.Context, cfg config.Config, logger *slog.Logger) (*GRPCServiceApp, error) {
	pool, err := openDatabase(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	passwordHasher, err := backendauth.NewBcryptPasswordHasher(0)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("create password hasher: %w", err)
	}
	tokenProvider, err := backendauth.NewJWTTokenProvider(backendauth.JWTTokenProviderConfig{
		SigningKey: []byte(cfg.JWTSigningKey), Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience,
	})
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("create token provider: %w", err)
	}
	txManager := postgres.NewTxManager(pool)
	auth, err := usecase.NewAuthService(usecase.AuthConfig{AccessTokenTTL: cfg.AccessTokenTTL, RefreshTokenTTL: cfg.RefreshTokenTTL}, usecase.AuthDependencies{
		PasswordHasher: passwordHasher, TokenProvider: tokenProvider,
		UserRepository: postgres.NewUserRepository(pool), SessionRepository: postgres.NewSessionRepository(pool), TxManager: txManager,
	})
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("create auth service: %w", err)
	}
	authenticator, err := usecase.NewAccessTokenAuthService(usecase.AccessTokenAuthDependencies{
		AccessTokenVerifier: tokenProvider, SessionRepository: postgres.NewSessionRepository(pool),
	})
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("create authenticator: %w", err)
	}
	server := grpc.NewServer()
	avitoshav1.RegisterAuthServiceServer(server, authserver.New(auth, authenticator))
	registerHealth(server)
	return &GRPCServiceApp{name: "auth-service", addr: cfg.GRPCAddr, logger: logger, server: server, db: pool, shutdownTimeout: cfg.ShutdownTimeout}, nil
}

func NewGameGRPCService(ctx context.Context, cfg config.Config, logger *slog.Logger) (*GRPCServiceApp, error) {
	pool, err := openDatabase(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	hub := realtime.NewHub(realtime.DefaultBufferSize)
	advice, err := newAdviceGenerator(cfg)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("create advice generator: %w", err)
	}
	game := newGameService(pool, postgres.NewTxManager(pool), hub, advice)
	marketplace := usecase.NewMarketplaceService(postgres.NewGameRepository(pool), postgres.NewTxManager(pool), nil, game)
	server := grpc.NewServer()
	avitoshav1.RegisterGameServiceServer(server, gameserver.New(game, marketplace, hub))
	registerHealth(server)
	return &GRPCServiceApp{name: "game-service", addr: cfg.GRPCAddr, logger: logger, server: server, db: pool, shutdownTimeout: cfg.ShutdownTimeout}, nil
}

func (a *GRPCServiceApp) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", a.addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", a.name, err)
	}
	runCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	defer a.db.Close()
	errChannel := make(chan error, 1)
	go func() {
		a.logger.Info(a.name+" started", "addr", a.addr)
		errChannel <- a.server.Serve(listener)
	}()
	select {
	case err := <-errChannel:
		if err != nil {
			return fmt.Errorf("serve %s: %w", a.name, err)
		}
		return nil
	case <-runCtx.Done():
	}
	stopped := make(chan struct{})
	go func() { a.server.GracefulStop(); close(stopped) }()
	select {
	case <-stopped:
	case <-time.After(a.shutdownTimeout):
		a.server.Stop()
	}
	return <-errChannel
}

type grpcServicesPinger []healthv1.HealthClient

func (clients grpcServicesPinger) Ping(ctx context.Context) error {
	for _, client := range clients {
		response, err := client.Check(ctx, &healthv1.HealthCheckRequest{})
		if err != nil || response.GetStatus() != healthv1.HealthCheckResponse_SERVING {
			if err != nil {
				return err
			}
			return fmt.Errorf("grpc service is not serving")
		}
	}
	return nil
}

func openDatabase(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := postgres.NewPool(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, startupDatabasePingTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

func registerHealth(server *grpc.Server) {
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthv1.HealthCheckResponse_SERVING)
	healthv1.RegisterHealthServer(server, healthServer)
}

func defaultUnaryDeadline(
	ctx context.Context,
	method string,
	request, response any,
	connection *grpc.ClientConn,
	invoke grpc.UnaryInvoker,
	options ...grpc.CallOption,
) error {
	if _, exists := ctx.Deadline(); exists {
		return invoke(ctx, method, request, response, connection, options...)
	}
	deadlineContext, cancel := context.WithTimeout(ctx, downstreamUnaryTimeout)
	defer cancel()
	return invoke(deadlineContext, method, request, response, connection, options...)
}

package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	backendauth "github.com/guitaramust-sudo/Avitosha/app/backend/internal/auth"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/config"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/handler"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/realtime"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/repository/postgres"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
)

const startupDatabasePingTimeout = 5 * time.Second

type App struct {
	logger          *slog.Logger
	server          *http.Server
	db              *pgxpool.Pool
	shutdownTimeout time.Duration
}

func New(ctx context.Context, cfg config.Config, logger *slog.Logger) (*App, error) {
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	defer func() {
		if err != nil {
			pool.Close()
		}
	}()

	pingCtx, cancelDatabasePing := context.WithTimeout(ctx, startupDatabasePingTimeout)
	defer cancelDatabasePing()

	if err := pool.Ping(pingCtx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	passwordHasher, err := backendauth.NewBcryptPasswordHasher(0)
	if err != nil {
		return nil, fmt.Errorf("create password hasher: %w", err)
	}

	tokenProvider, err := backendauth.NewJWTTokenProvider(backendauth.JWTTokenProviderConfig{
		SigningKey: []byte(cfg.JWTSigningKey),
		Issuer:     cfg.JWTIssuer,
		Audience:   cfg.JWTAudience,
	})
	if err != nil {
		return nil, fmt.Errorf("create token provider: %w", err)
	}
	txManager := postgres.NewTxManager(pool)
	eventHub := realtime.NewHub(realtime.DefaultBufferSize)
	game := newGameService(pool, txManager, eventHub)

	authService, err := usecase.NewAuthService(usecase.AuthConfig{
		AccessTokenTTL:  cfg.AccessTokenTTL,
		RefreshTokenTTL: cfg.RefreshTokenTTL,
	}, usecase.AuthDependencies{
		PasswordHasher:    passwordHasher,
		TokenProvider:     tokenProvider,
		UserRepository:    postgres.NewUserRepository(pool),
		SessionRepository: postgres.NewSessionRepository(pool),
		TxManager:         txManager,
		RegistrationHook:  newRegistrationHook(game, time.Now),
	})
	if err != nil {
		return nil, fmt.Errorf("create auth service: %w", err)
	}

	accessTokenAuthenticator, err := usecase.NewAccessTokenAuthService(usecase.AccessTokenAuthDependencies{
		AccessTokenVerifier: tokenProvider,
		SessionRepository:   postgres.NewSessionRepository(pool),
	})
	if err != nil {
		return nil, fmt.Errorf("create access token authenticator: %w", err)
	}

	router := handler.NewRouter(handler.RouterDependencies{
		Logger:                   logger,
		DB:                       pool,
		AuthService:              authService,
		AccessTokenAuthenticator: accessTokenAuthenticator,
		FrontendOrigin:           cfg.FrontendOrigin,
		RefreshTokenTTL:          cfg.RefreshTokenTTL,
		SecureRefreshCookie:      cfg.AppEnv == config.AppEnvProd,
		GameService:              game,
		EventHub:                 eventHub,
	})
	server := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      router,
		ReadTimeout:  cfg.HTTPReadTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout,
		IdleTimeout:  cfg.HTTPIdleTimeout,
	}

	return newApp(logger, server, pool, cfg.ShutdownTimeout), nil
}

func newGameService(
	pool *pgxpool.Pool,
	txManager usecase.TxManager,
	publisher usecase.DomainEventPublisher,
) *usecase.GameService {
	return usecase.NewGameService(usecase.GameServiceDependencies{
		Repository: postgres.NewGameRepository(pool), TxManager: txManager,
		IDGenerator: uuid.New, Publisher: publisher,
	})
}

func newApp(logger *slog.Logger, server *http.Server, db *pgxpool.Pool, shutdownTimeout time.Duration) *App {
	return &App{
		logger:          logger,
		server:          server,
		db:              db,
		shutdownTimeout: shutdownTimeout,
	}
}

func (a *App) Run(ctx context.Context) error {
	runCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	defer func() {
		if a.db != nil {
			a.db.Close()
		}
	}()

	errCh := make(chan error, 1)
	go func() {
		a.logger.Info("api server started", "addr", a.server.Addr)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("serve http: %w", err)
		}
		return nil
	case <-runCtx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
	defer cancel()

	a.logger.Info("api server stopping")
	if err := a.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	if err := <-errCh; err != nil {
		return fmt.Errorf("serve http: %w", err)
	}

	a.logger.Info("api server stopped")
	return nil
}

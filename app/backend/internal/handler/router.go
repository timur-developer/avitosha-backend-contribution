package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/realtime"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

type AuthService interface {
	Register(ctx context.Context, params usecase.RegisterParams) (usecase.AuthenticationResult, error)
	Login(ctx context.Context, params usecase.LoginParams) (usecase.AuthenticationResult, error)
	Refresh(ctx context.Context, params usecase.RefreshParams) (usecase.RefreshResult, error)
	Logout(ctx context.Context, params usecase.LogoutParams) error
	GetCurrentUser(ctx context.Context, params usecase.GetCurrentUserParams) (model.User, error)
}

type AccessTokenAuthenticator interface {
	AuthenticateAccessToken(ctx context.Context, token string) (model.AuthenticatedUser, error)
}

type RouterDependencies struct {
	Logger                   *slog.Logger
	DB                       DatabasePinger
	AuthService              AuthService
	AccessTokenAuthenticator AccessTokenAuthenticator
	FrontendOrigin           string
	RefreshTokenTTL          time.Duration
	SecureRefreshCookie      bool
	GameService              GameUseCase
	MarketplaceService       MarketplaceUseCase
	PhotoUploadService       PhotoUploadUseCase
	EventHub                 realtime.EventSubscriber
	Now                      func() time.Time
	AppEnv                   string
}

func NewRouter(deps RouterDependencies) *chi.Mux {
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}

	r := chi.NewRouter()
	swaggerUI := NewSwaggerUIHandler("/api/openapi.yaml")

	r.Use(RequestID)
	r.Use(StructuredRequestLogger(logger))
	r.Use(Recovery(logger))
	r.Use(CORS(deps.FrontendOrigin))

	r.Get("/health/live", Live)
	r.Get("/healthz", Live)
	r.Method("GET", "/health/ready", NewReadyHandler(deps.DB))
	r.Get("/swagger", func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "/swagger/", http.StatusMovedPermanently)
	})
	r.Handle("/swagger/*", http.StripPrefix("/swagger", swaggerUI))
	r.Get("/api/openapi.yaml", OpenAPISpec)

	mountAPIRoutes(r, logger, deps)

	return r
}

func mountAPIRoutes(r chi.Router, logger *slog.Logger, deps RouterDependencies) {
	authHandler := NewAuthHandler(AuthHandlerDependencies{
		Logger:              logger,
		AuthService:         deps.AuthService,
		RefreshTokenTTL:     deps.RefreshTokenTTL,
		SecureRefreshCookie: deps.SecureRefreshCookie,
	})
	r.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/logout", authHandler.Logout)
		})

		authenticated := BearerAuth(logger, deps.AccessTokenAuthenticator)
		r.With(authenticated).Get("/me", authHandler.Me)

	})

	marketplaceHandler := NewMarketplaceHandler(logger, deps.MarketplaceService, deps.Now)
	photoUploadHandler := NewPhotoUploadHandler(logger, deps.PhotoUploadService, deps.Now)
	r.Get("/api/v1/listing-categories", marketplaceHandler.ListCategories)
	r.Get("/api/v1/listings", marketplaceHandler.ListPublic)
	r.Get("/api/v1/listings/{listing_id}", marketplaceHandler.GetPublic)

	gameHandler := NewGameHandler(logger, deps.GameService, deps.Now)
	webSocketHandler := NewGameWebSocketHandler(logger, deps.EventHub, deps.FrontendOrigin)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(GameIdentity(logger, deps.AccessTokenAuthenticator, deps.AppEnv))
		r.Get("/me/listings", marketplaceHandler.ListMine)
		r.Get("/me/favorites", marketplaceHandler.ListFavorites)
		r.Post("/listings", marketplaceHandler.Create)
		r.Post("/uploads/listing-photos", photoUploadHandler.Create)
		r.Patch("/listings/{listing_id}", marketplaceHandler.Update)
		r.Post("/listings/{listing_id}/publish", marketplaceHandler.Publish)
		r.Post("/listings/{listing_id}/unpublish", marketplaceHandler.Unpublish)
		r.Put("/listings/{listing_id}/favorite", marketplaceHandler.AddFavorite)
		r.Delete("/listings/{listing_id}/favorite", marketplaceHandler.RemoveFavorite)
		r.Post("/listings/{listing_id}/views", marketplaceHandler.RegisterView)
		r.Post("/listings/{listing_id}/messages", marketplaceHandler.ContactSeller)
		r.Get("/listings/{listing_id}/messages", marketplaceHandler.ListMessages)
		r.Post("/listings/{listing_id}/purchase", marketplaceHandler.Purchase)
		r.Get("/pet", gameHandler.GetPet)
		r.Patch("/pet", gameHandler.RenamePet)
		r.Get("/tasks", gameHandler.ListTasks)
		r.Get("/tasks/{task_id}", gameHandler.GetTask)
		r.Get("/tasks/{task_id}/advice", gameHandler.GetTaskAdvice)
		r.Post("/actions", gameHandler.ProcessAction)
		r.Get("/room", gameHandler.GetRoom)
		r.Get("/story", gameHandler.GetStory)
		r.Get("/daily-summary", gameHandler.GetDailySummary)
		r.Get("/leaderboard", gameHandler.GetLeaderboard)
		r.Get("/achievements", gameHandler.GetAchievements)
		r.Get("/rewards/balance", gameHandler.GetRewardBalances)
		r.Get("/rewards/wallet", gameHandler.GetRewardWallet)
		r.Get("/ws", webSocketHandler.ServeHTTP)
	})
}

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	backendauth "github.com/guitaramust-sudo/Avitosha/app/backend/internal/auth"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/repository/postgres"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

const smokeTestDatabaseResetQuery = "TRUNCATE TABLE sessions, users RESTART IDENTITY CASCADE"

func TestAuthSmokeFlow(t *testing.T) {
	pool := newSmokeTestPool(t)
	router := newSmokeTestRouter(t, pool)

	email := "smoke-" + strings.ToLower(strings.ReplaceAll(time.Now().UTC().Format(time.RFC3339Nano), ":", "-")) + "@example.com"
	registerResponse, refreshCookie := postAuthCredentials(t, router, "/api/auth/register", email, "password123", http.StatusCreated)
	getCurrentUser(t, router, registerResponse.AccessToken, email)
	getGamePetStatus(t, router, registerResponse.AccessToken, http.StatusOK)

	refreshResponse, rotatedRefreshCookie := postWithCookie[refreshResponse](t, router, "/api/auth/refresh", refreshCookie, http.StatusOK)
	if refreshResponse.AccessToken == "" {
		t.Fatal("refresh returned empty access token")
	}
	if rotatedRefreshCookie == nil || rotatedRefreshCookie.Value == "" {
		t.Fatal("refresh did not set rotated refresh cookie")
	}
	if rotatedRefreshCookie.Value == refreshCookie.Value {
		t.Fatal("refresh token was not rotated")
	}

	getCurrentUser(t, router, refreshResponse.AccessToken, email)
	postWithCookie[struct{}](t, router, "/api/auth/logout", rotatedRefreshCookie, http.StatusNoContent)

	assertAccessTokenUnauthorized(t, router, refreshResponse.AccessToken, "/api/me")
	assertAccessTokenUnauthorized(t, router, refreshResponse.AccessToken, "/api/v1/pet")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	req.AddCookie(rotatedRefreshCookie)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("refresh after logout status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	loginResponse, loginRefreshCookie := postAuthCredentials(t, router, "/api/auth/login", email, "password123", http.StatusOK)
	getCurrentUser(t, router, loginResponse.AccessToken, email)
	postWithCookie[struct{}](t, router, "/api/auth/logout", loginRefreshCookie, http.StatusNoContent)
}

func TestGameRoutesReturnUnauthorizedForStaleAccessTokenAfterDatabaseReset(t *testing.T) {
	pool := newSmokeTestPool(t)
	router := newSmokeTestRouter(t, pool)

	email := "stale-" + strings.ToLower(strings.ReplaceAll(time.Now().UTC().Format(time.RFC3339Nano), ":", "-")) + "@example.com"
	registerResponse, _ := postAuthCredentials(t, router, "/api/auth/register", email, "password123", http.StatusCreated)
	getGamePetStatus(t, router, registerResponse.AccessToken, http.StatusOK)

	if _, err := pool.Exec(context.Background(), smokeTestDatabaseResetQuery); err != nil {
		t.Fatalf("reset smoke test database: %v", err)
	}

	assertAccessTokenUnauthorized(t, router, registerResponse.AccessToken, "/api/v1/pet")
}

func newSmokeTestRouter(t *testing.T, pool *pgxpool.Pool) http.Handler {
	t.Helper()

	passwordHasher, err := backendauth.NewBcryptPasswordHasher(bcrypt.MinCost)
	if err != nil {
		t.Fatalf("NewBcryptPasswordHasher() error = %v", err)
	}

	tokenProvider, err := backendauth.NewJWTTokenProvider(backendauth.JWTTokenProviderConfig{
		SigningKey: []byte("test-signing-key-for-auth-smoke-flow"),
		Issuer:     "avitosha-test",
		Audience:   "avitosha-web-test",
	})
	if err != nil {
		t.Fatalf("NewJWTTokenProvider() error = %v", err)
	}

	txManager := postgres.NewTxManager(pool)
	sessionRepository := postgres.NewSessionRepository(pool)
	gameService := usecase.NewGameService(usecase.GameServiceDependencies{
		Repository:  postgres.NewGameRepository(pool),
		TxManager:   txManager,
		IDGenerator: uuid.New,
	})

	authService, err := usecase.NewAuthService(usecase.AuthConfig{
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 30 * 24 * time.Hour,
	}, usecase.AuthDependencies{
		PasswordHasher:    passwordHasher,
		TokenProvider:     tokenProvider,
		UserRepository:    postgres.NewUserRepository(pool),
		SessionRepository: sessionRepository,
		TxManager:         txManager,
	})
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}

	accessTokenAuthenticator, err := usecase.NewAccessTokenAuthService(usecase.AccessTokenAuthDependencies{
		AccessTokenVerifier: tokenProvider,
		SessionRepository:   sessionRepository,
	})
	if err != nil {
		t.Fatalf("NewAccessTokenAuthService() error = %v", err)
	}

	router := NewRouter(RouterDependencies{
		Logger:                   slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:                       pool,
		AuthService:              authService,
		AccessTokenAuthenticator: accessTokenAuthenticator,
		FrontendOrigin:           "http://localhost:3000",
		RefreshTokenTTL:          30 * 24 * time.Hour,
		SecureRefreshCookie:      false,
		GameService:              gameService,
	})

	return router
}

func newSmokeTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	requireSmokeTestDatabaseName(t, databaseURL)

	pool, err := postgres.NewPool(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("NewPool() error = %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("pool.Ping() error = %v", err)
	}
	if _, err := pool.Exec(context.Background(), smokeTestDatabaseResetQuery); err != nil {
		t.Fatalf("reset smoke test database: %v", err)
	}

	return pool
}

func requireSmokeTestDatabaseName(t *testing.T, databaseURL string) {
	t.Helper()

	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}

	databaseName := strings.TrimPrefix(parsedURL.Path, "/")
	if !strings.Contains(strings.ToLower(databaseName), "test") {
		t.Fatalf("TEST_DATABASE_URL must point to a test database, got %q", databaseName)
	}
}

func postAuthCredentials(t *testing.T, router http.Handler, path, email, password string, wantStatus int) (authResponse, *http.Cookie) {
	t.Helper()

	body := map[string]string{
		"email":    email,
		"password": password,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", jsonContentType)
	router.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("%s status = %d, want %d, body: %s", path, rec.Code, wantStatus, rec.Body.String())
	}

	var response authResponse
	decodeSmokeJSONResponse(t, rec, &response)
	if response.AccessToken == "" {
		t.Fatalf("%s returned empty access token", path)
	}

	refreshCookie := findSmokeCookie(rec.Result().Cookies(), refreshTokenCookieName)
	if refreshCookie == nil || refreshCookie.Value == "" {
		t.Fatalf("%s did not set refresh cookie", path)
	}

	return response, refreshCookie
}

func getCurrentUser(t *testing.T, router http.Handler, accessToken, wantEmail string) {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("/api/me status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var response userResponse
	decodeSmokeJSONResponse(t, rec, &response)
	if response.User.Email != wantEmail {
		t.Fatalf("/api/me email = %q, want %q", response.User.Email, wantEmail)
	}
}

func getGamePetStatus(t *testing.T, router http.Handler, accessToken string, wantStatus int) {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pet", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	router.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("/api/v1/pet status = %d, want %d, body: %s", rec.Code, wantStatus, rec.Body.String())
	}
}

func assertAccessTokenUnauthorized(t *testing.T, router http.Handler, accessToken, path string) {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("%s status = %d, want %d, body: %s", path, rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func postWithCookie[T any](t *testing.T, router http.Handler, path string, cookie *http.Cookie, wantStatus int) (T, *http.Cookie) {
	t.Helper()

	var response T
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, nil)
	req.AddCookie(cookie)
	router.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("%s status = %d, want %d, body: %s", path, rec.Code, wantStatus, rec.Body.String())
	}

	if rec.Body.Len() > 0 {
		decodeSmokeJSONResponse(t, rec, &response)
	}

	return response, findSmokeCookie(rec.Result().Cookies(), refreshTokenCookieName)
}

func decodeSmokeJSONResponse(t *testing.T, rec *httptest.ResponseRecorder, target any) {
	t.Helper()

	if err := json.Unmarshal(rec.Body.Bytes(), target); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
}

func findSmokeCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}

	return nil
}

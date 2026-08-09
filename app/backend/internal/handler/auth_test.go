package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

func TestRegisterSuccess(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t, testRouterConfig{
		authService: fakeAuthService{
			registerFunc: func(_ context.Context, params usecase.RegisterParams) (usecase.AuthenticationResult, error) {
				if params.Email != "user@example.com" {
					t.Fatalf("Email = %q, want user@example.com", params.Email)
				}
				if params.Password != "password123" {
					t.Fatalf("Password = %q, want password123", params.Password)
				}
				if params.UserAgent == nil || *params.UserAgent != "test-agent" {
					t.Fatalf("UserAgent = %#v, want test-agent", params.UserAgent)
				}

				return usecase.AuthenticationResult{
					User: model.User{
						ID:    uuid.MustParse("8f0ed065-aefa-4f56-87d0-e2ef2ef43f0d"),
						Email: "user@example.com",
					},
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
				}, nil
			},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"email":"user@example.com","password":"password123"}`))
	req.Header.Set("Content-Type", jsonContentType)
	req.Header.Set("User-Agent", "test-agent")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if contentType := rec.Header().Get("Content-Type"); contentType != jsonContentType {
		t.Fatalf("Content-Type = %q, want %q", contentType, jsonContentType)
	}

	var response authResponse
	decodeJSONResponse(t, rec, &response)
	if response.AccessToken != "access-token" {
		t.Fatalf("AccessToken = %q, want access-token", response.AccessToken)
	}
	if response.User.Email != "user@example.com" {
		t.Fatalf("User.Email = %q, want user@example.com", response.User.Email)
	}
	if strings.Contains(rec.Body.String(), "refresh-token") {
		t.Fatal("response body unexpectedly contains refresh token")
	}

	cookieHeader := rec.Header().Get("Set-Cookie")
	for _, fragment := range []string{
		"refresh_token=refresh-token",
		"Path=/api/auth",
		"HttpOnly",
		"SameSite=Lax",
		"Max-Age=2592000",
	} {
		if !strings.Contains(cookieHeader, fragment) {
			t.Fatalf("Set-Cookie = %q, want fragment %q", cookieHeader, fragment)
		}
	}
	if strings.Contains(cookieHeader, "Secure") {
		t.Fatalf("Set-Cookie = %q, should not contain Secure in non-production mode", cookieHeader)
	}
}

func TestLoginSuccessSetsSecureCookieInProduction(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t, testRouterConfig{
		secureRefreshCookie: true,
		authService: fakeAuthService{
			loginFunc: func(_ context.Context, params usecase.LoginParams) (usecase.AuthenticationResult, error) {
				return usecase.AuthenticationResult{
					User: model.User{
						ID:    uuid.MustParse("8f0ed065-aefa-4f56-87d0-e2ef2ef43f0d"),
						Email: params.Email,
					},
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
				}, nil
			},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"email":"user@example.com","password":"password123"}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if cookieHeader := rec.Header().Get("Set-Cookie"); !strings.Contains(cookieHeader, "Secure") {
		t.Fatalf("Set-Cookie = %q, want Secure attribute", cookieHeader)
	}
}

func TestRegisterMalformedJSON(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t, testRouterConfig{})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"email":"user@example.com"`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusBadRequest, invalidRequestCode, "Request body must be valid JSON")
}

func TestRegisterInvalidEmail(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t, testRouterConfig{})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"email":"not-an-email","password":"password123"}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusBadRequest, invalidRequestCode, "Email must be a valid email address")
}

func TestRegisterMissingPassword(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t, testRouterConfig{})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"email":"user@example.com"}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusBadRequest, invalidRequestCode, "Password is required")
}

func TestRegisterDuplicateEmail(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t, testRouterConfig{
		authService: fakeAuthService{
			registerFunc: func(_ context.Context, _ usecase.RegisterParams) (usecase.AuthenticationResult, error) {
				return usecase.AuthenticationResult{}, usecase.ErrEmailAlreadyExists
			},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"email":"user@example.com","password":"password123"}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusConflict, emailAlreadyExistsCode, "Email already exists")
}

func TestLoginInvalidCredentials(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t, testRouterConfig{
		authService: fakeAuthService{
			loginFunc: func(_ context.Context, _ usecase.LoginParams) (usecase.AuthenticationResult, error) {
				return usecase.AuthenticationResult{}, usecase.ErrInvalidCredentials
			},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"email":"user@example.com","password":"password123"}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusUnauthorized, invalidCredentialsCode, "Invalid email or password")
}

func TestRefreshWithoutCookie(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t, testRouterConfig{})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusUnauthorized, unauthorizedCode, "Authentication is required")
}

func TestRefreshSuccessSetsNewCookie(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t, testRouterConfig{
		authService: fakeAuthService{
			refreshFunc: func(_ context.Context, params usecase.RefreshParams) (usecase.RefreshResult, error) {
				if params.RefreshToken != "old-refresh-token" {
					t.Fatalf("RefreshToken = %q, want old-refresh-token", params.RefreshToken)
				}

				return usecase.RefreshResult{
					AccessToken:  "new-access-token",
					RefreshToken: "new-refresh-token",
				}, nil
			},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: "old-refresh-token"})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response refreshResponse
	decodeJSONResponse(t, rec, &response)
	if response.AccessToken != "new-access-token" {
		t.Fatalf("AccessToken = %q, want new-access-token", response.AccessToken)
	}
	if strings.Contains(rec.Body.String(), "new-refresh-token") {
		t.Fatal("response body unexpectedly contains refresh token")
	}
	if cookieHeader := rec.Header().Get("Set-Cookie"); !strings.Contains(cookieHeader, "refresh_token=new-refresh-token") {
		t.Fatalf("Set-Cookie = %q, want new refresh token", cookieHeader)
	}
}

func TestLogoutClearsCookie(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t, testRouterConfig{
		authService: fakeAuthService{
			logoutFunc: func(_ context.Context, params usecase.LogoutParams) error {
				if params.RefreshToken != "refresh-token" {
					t.Fatalf("RefreshToken = %q, want refresh-token", params.RefreshToken)
				}
				return nil
			},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: refreshTokenCookieName, Value: "refresh-token"})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if body := rec.Body.String(); body != "" {
		t.Fatalf("body = %q, want empty body", body)
	}
	for _, fragment := range []string{
		"refresh_token=",
		"Path=/api/auth",
		"HttpOnly",
		"SameSite=Lax",
		"Max-Age=0",
	} {
		if !strings.Contains(rec.Header().Get("Set-Cookie"), fragment) {
			t.Fatalf("Set-Cookie = %q, want fragment %q", rec.Header().Get("Set-Cookie"), fragment)
		}
	}
}

func TestMeWithoutToken(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t, testRouterConfig{})
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusUnauthorized, unauthorizedCode, "Authentication is required")
}

func TestMeWithInvalidToken(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t, testRouterConfig{
		authenticator: fakeAccessTokenAuthenticator{
			authenticateFunc: func(_ context.Context, _ string) (model.AuthenticatedUser, error) {
				return model.AuthenticatedUser{}, usecase.ErrUnauthorized
			},
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusUnauthorized, unauthorizedCode, "Authentication is required")
}

func TestMeSuccess(t *testing.T) {
	t.Parallel()

	expectedUser := model.AuthenticatedUser{
		UserID:    uuid.MustParse("8f0ed065-aefa-4f56-87d0-e2ef2ef43f0d"),
		SessionID: uuid.MustParse("8f0ed065-aefa-4f56-87d0-e2ef2ef43f0e"),
	}

	router := newTestRouter(t, testRouterConfig{
		authService: fakeAuthService{
			getCurrentUserFunc: func(_ context.Context, params usecase.GetCurrentUserParams) (model.User, error) {
				if params.AuthenticatedUser != expectedUser {
					t.Fatalf("AuthenticatedUser = %#v, want %#v", params.AuthenticatedUser, expectedUser)
				}

				return model.User{
					ID:    expectedUser.UserID,
					Email: "user@example.com",
				}, nil
			},
		},
		authenticator: fakeAccessTokenAuthenticator{
			authenticateFunc: func(_ context.Context, token string) (model.AuthenticatedUser, error) {
				if token != "valid-token" {
					t.Fatalf("token = %q, want valid-token", token)
				}
				return expectedUser, nil
			},
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if contentType := rec.Header().Get("Content-Type"); contentType != jsonContentType {
		t.Fatalf("Content-Type = %q, want %q", contentType, jsonContentType)
	}

	var response userResponse
	decodeJSONResponse(t, rec, &response)
	if response.User.Email != "user@example.com" {
		t.Fatalf("User.Email = %q, want user@example.com", response.User.Email)
	}
}

func TestMeWithInactiveSession(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t, testRouterConfig{
		authenticator: fakeAccessTokenAuthenticator{
			authenticateFunc: func(_ context.Context, token string) (model.AuthenticatedUser, error) {
				if token != "stale-token" {
					t.Fatalf("token = %q, want stale-token", token)
				}
				return model.AuthenticatedUser{}, usecase.ErrUnauthorized
			},
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer stale-token")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusUnauthorized, unauthorizedCode, "Authentication is required")
}

func TestInternalErrorsDoNotLeak(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t, testRouterConfig{
		authService: fakeAuthService{
			loginFunc: func(_ context.Context, _ usecase.LoginParams) (usecase.AuthenticationResult, error) {
				return usecase.AuthenticationResult{}, errors.New("sqlstate 23505 duplicate key")
			},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"email":"user@example.com","password":"password123"}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusInternalServerError, internalErrorCode, "Internal server error")
	if strings.Contains(rec.Body.String(), "sqlstate") {
		t.Fatalf("body = %q, should not leak storage details", rec.Body.String())
	}
}

func TestCORSPreflightUsesConcreteOrigin(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t, testRouterConfig{
		frontendOrigin: "http://localhost:3000",
	})
	req := httptest.NewRequest(http.MethodOptions, "/api/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if allowOrigin := rec.Header().Get("Access-Control-Allow-Origin"); allowOrigin != "http://localhost:3000" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want http://localhost:3000", allowOrigin)
	}
	if allowOrigin := rec.Header().Get("Access-Control-Allow-Origin"); allowOrigin == "*" {
		t.Fatal("Access-Control-Allow-Origin must not be wildcard for credentialed requests")
	}
	if credentials := rec.Header().Get("Access-Control-Allow-Credentials"); credentials != "true" {
		t.Fatalf("Access-Control-Allow-Credentials = %q, want true", credentials)
	}
	if allowMethods := rec.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(allowMethods, http.MethodPatch) {
		t.Fatalf("Access-Control-Allow-Methods = %q, want %s", allowMethods, http.MethodPatch)
	}
}

func TestRecoveryMiddlewareReturnsJSON(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t, testRouterConfig{
		authService: fakeAuthService{
			registerFunc: func(_ context.Context, _ usecase.RegisterParams) (usecase.AuthenticationResult, error) {
				panic("boom")
			},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"email":"user@example.com","password":"password123"}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusInternalServerError, internalErrorCode, "Internal server error")
}

type testRouterConfig struct {
	dbErr               error
	authService         AuthService
	authenticator       AccessTokenAuthenticator
	frontendOrigin      string
	secureRefreshCookie bool
	refreshTokenTTL     time.Duration
}

type fakeAuthService struct {
	registerFunc       func(context.Context, usecase.RegisterParams) (usecase.AuthenticationResult, error)
	loginFunc          func(context.Context, usecase.LoginParams) (usecase.AuthenticationResult, error)
	refreshFunc        func(context.Context, usecase.RefreshParams) (usecase.RefreshResult, error)
	logoutFunc         func(context.Context, usecase.LogoutParams) error
	getCurrentUserFunc func(context.Context, usecase.GetCurrentUserParams) (model.User, error)
}

type fakeAccessTokenAuthenticator struct {
	authenticateFunc func(context.Context, string) (model.AuthenticatedUser, error)
}

func (f fakeAuthService) Register(ctx context.Context, params usecase.RegisterParams) (usecase.AuthenticationResult, error) {
	if f.registerFunc == nil {
		return usecase.AuthenticationResult{}, nil
	}
	return f.registerFunc(ctx, params)
}

func (f fakeAuthService) Login(ctx context.Context, params usecase.LoginParams) (usecase.AuthenticationResult, error) {
	if f.loginFunc == nil {
		return usecase.AuthenticationResult{}, nil
	}
	return f.loginFunc(ctx, params)
}

func (f fakeAuthService) Refresh(ctx context.Context, params usecase.RefreshParams) (usecase.RefreshResult, error) {
	if f.refreshFunc == nil {
		return usecase.RefreshResult{}, nil
	}
	return f.refreshFunc(ctx, params)
}

func (f fakeAuthService) Logout(ctx context.Context, params usecase.LogoutParams) error {
	if f.logoutFunc == nil {
		return nil
	}
	return f.logoutFunc(ctx, params)
}

func (f fakeAuthService) GetCurrentUser(ctx context.Context, params usecase.GetCurrentUserParams) (model.User, error) {
	if f.getCurrentUserFunc == nil {
		return model.User{}, nil
	}
	return f.getCurrentUserFunc(ctx, params)
}

func (f fakeAccessTokenAuthenticator) AuthenticateAccessToken(ctx context.Context, token string) (model.AuthenticatedUser, error) {
	if f.authenticateFunc == nil {
		return model.AuthenticatedUser{}, nil
	}
	return f.authenticateFunc(ctx, token)
}

func newTestRouter(t *testing.T, cfg testRouterConfig) http.Handler {
	t.Helper()

	frontendOrigin := cfg.frontendOrigin
	if frontendOrigin == "" {
		frontendOrigin = "http://localhost:3000"
	}

	refreshTokenTTL := cfg.refreshTokenTTL
	if refreshTokenTTL == 0 {
		refreshTokenTTL = 30 * 24 * time.Hour
	}

	return NewRouter(RouterDependencies{
		Logger:                   slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:                       fakeDatabasePinger{err: cfg.dbErr},
		AuthService:              cfg.authService,
		AccessTokenAuthenticator: cfg.authenticator,
		FrontendOrigin:           frontendOrigin,
		RefreshTokenTTL:          refreshTokenTTL,
		SecureRefreshCookie:      cfg.secureRefreshCookie,
	})
}

func decodeJSONResponse(t *testing.T, rec *httptest.ResponseRecorder, target any) {
	t.Helper()

	if err := json.Unmarshal(rec.Body.Bytes(), target); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
}

func assertErrorResponse(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode, wantMessage string) {
	t.Helper()

	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d", rec.Code, wantStatus)
	}
	if contentType := rec.Header().Get("Content-Type"); contentType != jsonContentType {
		t.Fatalf("Content-Type = %q, want %q", contentType, jsonContentType)
	}

	var response errorResponse
	decodeJSONResponse(t, rec, &response)
	if response.Error.Code != wantCode {
		t.Fatalf("error.code = %q, want %q", response.Error.Code, wantCode)
	}
	if response.Error.Message != wantMessage {
		t.Fatalf("error.message = %q, want %q", response.Error.Message, wantMessage)
	}
}

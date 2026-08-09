package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	backendauth "github.com/guitaramust-sudo/Avitosha/app/backend/internal/auth"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

const (
	refreshTokenCookieName = "refresh_token"
	refreshTokenCookiePath = "/api/auth"
	minPasswordLength      = 8
	maxPasswordLength      = 128
	maxEmailLength         = 320
	invalidRequestCode     = "invalid_request"
	invalidCredentialsCode = "invalid_credentials"
	unauthorizedCode       = "unauthorized"
	emailAlreadyExistsCode = "email_already_exists"
	sessionExpiredCode     = "session_expired"
	internalErrorCode      = "internal_error"
)

type AuthHandlerDependencies struct {
	Logger              *slog.Logger
	AuthService         AuthService
	RefreshTokenTTL     time.Duration
	SecureRefreshCookie bool
}

type AuthHandler struct {
	logger       *slog.Logger
	authService  AuthService
	cookieConfig refreshCookieConfig
}

type refreshCookieConfig struct {
	maxAgeSeconds int
	secure        bool
}

type authCredentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type requestValidationError struct {
	message string
}

func (e requestValidationError) Error() string {
	return e.message
}

func NewAuthHandler(deps AuthHandlerDependencies) AuthHandler {
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return AuthHandler{
		logger:      logger,
		authService: deps.AuthService,
		cookieConfig: refreshCookieConfig{
			maxAgeSeconds: int(deps.RefreshTokenTTL / time.Second),
			secure:        deps.SecureRefreshCookie,
		},
	}
}

func (h AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	request, err := decodeAuthCredentialsRequest(r)
	if err != nil {
		h.writeValidationError(w, err)
		return
	}

	result, err := h.authService.Register(r.Context(), usecase.RegisterParams{
		Email:     request.Email,
		Password:  request.Password,
		UserAgent: userAgentPointer(r.UserAgent()),
	})
	if err != nil {
		h.writeUsecaseError(w, r, "register", err)
		return
	}

	h.setRefreshTokenCookie(w, result.RefreshToken)
	writeJSON(w, http.StatusCreated, authResponse{
		AccessToken: result.AccessToken,
		User:        newResponseUser(result.User),
	})
}

func (h AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	request, err := decodeAuthCredentialsRequest(r)
	if err != nil {
		h.writeValidationError(w, err)
		return
	}

	result, err := h.authService.Login(r.Context(), usecase.LoginParams{
		Email:     request.Email,
		Password:  request.Password,
		UserAgent: userAgentPointer(r.UserAgent()),
	})
	if err != nil {
		h.writeUsecaseError(w, r, "login", err)
		return
	}

	h.setRefreshTokenCookie(w, result.RefreshToken)
	writeJSON(w, http.StatusOK, authResponse{
		AccessToken: result.AccessToken,
		User:        newResponseUser(result.User),
	})
}

func (h AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := readRefreshTokenCookie(r)
	if err != nil {
		writeErrorResponse(w, http.StatusUnauthorized, unauthorizedCode, "Authentication is required")
		return
	}

	result, err := h.authService.Refresh(r.Context(), usecase.RefreshParams{
		RefreshToken: refreshToken,
	})
	if err != nil {
		h.writeUsecaseError(w, r, "refresh", err)
		return
	}

	h.setRefreshTokenCookie(w, result.RefreshToken)
	writeJSON(w, http.StatusOK, refreshResponse{
		AccessToken: result.AccessToken,
	})
}

func (h AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := readRefreshTokenCookie(r)
	if err != nil {
		writeErrorResponse(w, http.StatusUnauthorized, unauthorizedCode, "Authentication is required")
		return
	}

	if err := h.authService.Logout(r.Context(), usecase.LogoutParams{
		RefreshToken: refreshToken,
	}); err != nil {
		h.writeUsecaseError(w, r, "logout", err)
		return
	}

	h.clearRefreshTokenCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	authenticatedUser, ok := backendauth.AuthenticatedUserFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, unauthorizedCode, "Authentication is required")
		return
	}

	user, err := h.authService.GetCurrentUser(r.Context(), usecase.GetCurrentUserParams{
		AuthenticatedUser: authenticatedUser,
	})
	if err != nil {
		h.writeUsecaseError(w, r, "get_current_user", err)
		return
	}

	writeJSON(w, http.StatusOK, userResponse{
		User: newResponseUser(user),
	})
}

func (h AuthHandler) setRefreshTokenCookie(w http.ResponseWriter, refreshToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    refreshToken,
		Path:     refreshTokenCookiePath,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.cookieConfig.secure,
		MaxAge:   h.cookieConfig.maxAgeSeconds,
		Expires:  time.Now().UTC().Add(time.Duration(h.cookieConfig.maxAgeSeconds) * time.Second),
	})
}

func (h AuthHandler) clearRefreshTokenCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    "",
		Path:     refreshTokenCookiePath,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.cookieConfig.secure,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0).UTC(),
	})
}

func (h AuthHandler) writeValidationError(w http.ResponseWriter, err error) {
	writeErrorResponse(w, http.StatusBadRequest, invalidRequestCode, err.Error())
}

func (h AuthHandler) writeUsecaseError(w http.ResponseWriter, r *http.Request, operation string, err error) {
	statusCode, code, message := mapUsecaseError(err)
	if statusCode == http.StatusInternalServerError {
		h.logger.Error(
			"auth request failed",
			"request_id", chimiddleware.GetReqID(r.Context()),
			"operation", operation,
			"error", err.Error(),
		)
	}

	writeErrorResponse(w, statusCode, code, message)
}

func decodeAuthCredentialsRequest(r *http.Request) (authCredentialsRequest, error) {
	var request authCredentialsRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return authCredentialsRequest{}, requestValidationError{message: "Request body must be valid JSON"}
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return authCredentialsRequest{}, requestValidationError{message: "Request body must contain a single JSON object"}
	}

	if err := validateAuthCredentialsRequest(request); err != nil {
		return authCredentialsRequest{}, err
	}

	return request, nil
}

func validateAuthCredentialsRequest(request authCredentialsRequest) error {
	email := strings.TrimSpace(request.Email)
	password := request.Password

	switch {
	case email == "":
		return requestValidationError{message: "Email is required"}
	case len(email) > maxEmailLength:
		return requestValidationError{message: "Email must not exceed 320 characters"}
	}

	parsedAddress, err := mail.ParseAddress(email)
	if err != nil || parsedAddress.Address != email {
		return requestValidationError{message: "Email must be a valid email address"}
	}

	switch {
	case strings.TrimSpace(password) == "":
		return requestValidationError{message: "Password is required"}
	case len(password) < minPasswordLength || len(password) > maxPasswordLength:
		return requestValidationError{message: "Password must be between 8 and 128 characters"}
	}

	return nil
}

func readRefreshTokenCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(refreshTokenCookieName)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(cookie.Value) == "" {
		return "", requestValidationError{message: "Authentication is required"}
	}

	return cookie.Value, nil
}

func newResponseUser(user model.User) responseUser {
	return responseUser{
		ID:    user.ID.String(),
		Email: user.Email,
	}
}

func userAgentPointer(userAgent string) *string {
	userAgent = strings.TrimSpace(userAgent)
	if userAgent == "" {
		return nil
	}

	return &userAgent
}

func mapUsecaseError(err error) (int, string, string) {
	switch {
	case errors.Is(err, usecase.ErrInvalidInput):
		return http.StatusBadRequest, invalidRequestCode, "Request is invalid"
	case errors.Is(err, usecase.ErrEmailAlreadyExists):
		return http.StatusConflict, emailAlreadyExistsCode, "Email already exists"
	case errors.Is(err, usecase.ErrInvalidCredentials):
		return http.StatusUnauthorized, invalidCredentialsCode, "Invalid email or password"
	case errors.Is(err, usecase.ErrSessionExpired):
		return http.StatusUnauthorized, sessionExpiredCode, "Session has expired or is no longer valid"
	case errors.Is(err, usecase.ErrUnauthorized):
		return http.StatusUnauthorized, unauthorizedCode, "Authentication is required"
	default:
		return http.StatusInternalServerError, internalErrorCode, "Internal server error"
	}
}

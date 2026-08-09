package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

type gameUserContextKey struct{}

func GameIdentity(logger *slog.Logger, authenticator AccessTokenAuthenticator) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if header := strings.TrimSpace(r.Header.Get("X-User-ID")); header != "" {
				userID, err := uuid.Parse(header)
				if err != nil || userID == uuid.Nil {
					writeErrorResponse(w, http.StatusBadRequest, invalidRequestCode, "X-User-ID must be a UUID")
					return
				}
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), gameUserContextKey{}, userID)))
				return
			}

			authorization := r.Header.Get("Authorization")
			if authorization == "" {
				if queryToken := strings.TrimSpace(r.URL.Query().Get("access_token")); queryToken != "" {
					authorization = "Bearer " + queryToken
				}
			}
			token, err := bearerToken(authorization)
			if err != nil {
				writeErrorResponse(w, http.StatusUnauthorized, unauthorizedCode, "Authentication is required")
				return
			}
			authenticated, err := authenticateAccessToken(r, token, authenticator)
			if err != nil {
				if !errors.Is(err, usecase.ErrUnauthorized) {
					logger.Error("game identity session check failed",
						"request_id", chimiddleware.GetReqID(r.Context()), "path", r.URL.Path,
						"category", "access_token_auth_internal_error", "error", err.Error())
					writeErrorResponse(w, http.StatusInternalServerError, internalErrorCode, "Internal server error")
					return
				}
				logger.Warn("game identity authentication failed",
					"request_id", chimiddleware.GetReqID(r.Context()), "path", r.URL.Path,
					"category", "inactive_or_invalid_access_token")
				writeErrorResponse(w, http.StatusUnauthorized, unauthorizedCode, "Authentication is required")
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(
				r.Context(), gameUserContextKey{}, authenticated.UserID,
			)))
		})
	}
}

func gameUserID(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(gameUserContextKey{}).(uuid.UUID)
	return userID, ok && userID != uuid.Nil
}

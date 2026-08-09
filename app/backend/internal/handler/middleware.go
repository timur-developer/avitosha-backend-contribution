package handler

import (
	"bufio"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	backendauth "github.com/guitaramust-sudo/Avitosha/app/backend/internal/auth"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

func RequestID(next http.Handler) http.Handler {
	return chimiddleware.RequestID(next)
}

func StructuredRequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			recorder := &responseRecorder{ResponseWriter: w}

			next.ServeHTTP(recorder, r)

			logger.Info(
				"http request completed",
				"request_id", chimiddleware.GetReqID(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"status", recorder.Status(),
				"bytes", recorder.bytesWritten,
				"duration_ms", time.Since(startedAt).Milliseconds(),
			)
		})
	}
}

func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error(
						"panic recovered",
						"request_id", chimiddleware.GetReqID(r.Context()),
						"method", r.Method,
						"path", r.URL.Path,
						"category", "panic",
					)
					writeErrorResponse(w, http.StatusInternalServerError, "internal_error", "Internal server error")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

func CORS(frontendOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin != "" {
				w.Header().Add("Vary", "Origin")
				if frontendOrigin != "" && origin == frontendOrigin {
					w.Header().Set("Access-Control-Allow-Origin", frontendOrigin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-User-ID")
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
					w.Header().Set("Access-Control-Expose-Headers", "X-Request-Id")
				}
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func BearerAuth(logger *slog.Logger, authenticator AccessTokenAuthenticator) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := bearerToken(r.Header.Get("Authorization"))
			if err != nil {
				writeErrorResponse(w, http.StatusUnauthorized, "unauthorized", "Authentication is required")
				return
			}

			authenticatedUser, err := authenticateAccessToken(r, token, authenticator)
			if err != nil {
				if !errors.Is(err, usecase.ErrUnauthorized) {
					logger.Error(
						"access token session check failed",
						"request_id", chimiddleware.GetReqID(r.Context()),
						"path", r.URL.Path,
						"category", "access_token_auth_internal_error",
						"error", err.Error(),
					)
					writeErrorResponse(w, http.StatusInternalServerError, internalErrorCode, "Internal server error")
					return
				}
				logger.Warn(
					"access token authentication failed",
					"request_id", chimiddleware.GetReqID(r.Context()),
					"path", r.URL.Path,
					"category", "inactive_or_invalid_access_token",
				)
				writeErrorResponse(w, http.StatusUnauthorized, "unauthorized", "Authentication is required")
				return
			}

			next.ServeHTTP(w, r.WithContext(backendauth.ContextWithAuthenticatedUser(r.Context(), authenticatedUser)))
		})
	}
}

type responseRecorder struct {
	http.ResponseWriter
	status       int
	bytesWritten int
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(body []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}

	written, err := r.ResponseWriter.Write(body)
	r.bytesWritten += written
	return written, err
}

func (r *responseRecorder) Status() int {
	if r.status == 0 {
		return http.StatusOK
	}

	return r.status
}

func (r *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return hijacker.Hijack()
}

func bearerToken(authorizationHeader string) (string, error) {
	parts := strings.Fields(strings.TrimSpace(authorizationHeader))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", backendauth.ErrInvalidAccessToken
	}

	return parts[1], nil
}

func authenticateAccessToken(r *http.Request, token string, authenticator AccessTokenAuthenticator) (model.AuthenticatedUser, error) {
	if authenticator == nil {
		return model.AuthenticatedUser{}, usecase.ErrUnauthorized
	}

	return authenticator.AuthenticateAccessToken(r.Context(), token)
}

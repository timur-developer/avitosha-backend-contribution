package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrSessionExpired     = errors.New("session expired")
	ErrInternal           = errors.New("internal error")
)

type RegisterParams struct {
	Email     string
	Password  string
	UserAgent *string
}

type LoginParams struct {
	Email     string
	Password  string
	UserAgent *string
}

type RefreshParams struct {
	RefreshToken string
}

type LogoutParams struct {
	RefreshToken string
}

type GetCurrentUserParams struct {
	AuthenticatedUser model.AuthenticatedUser
}

type AuthenticationResult struct {
	User         model.User
	AccessToken  string
	RefreshToken string
}

type RefreshResult struct {
	AccessToken  string
	RefreshToken string
}

type AuthConfig struct {
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type RegistrationHook interface {
	AfterRegister(ctx context.Context, user model.User) error
}

type AuthDependencies struct {
	PasswordHasher    PasswordHasher
	TokenProvider     TokenProvider
	UserRepository    UserRepository
	SessionRepository SessionRepository
	TxManager         TxManager
	RegistrationHook  RegistrationHook
	Now               func() time.Time
}

type AuthService struct {
	passwordHasher    PasswordHasher
	tokenProvider     TokenProvider
	userRepository    UserRepository
	sessionRepository SessionRepository
	txManager         TxManager
	registrationHook  RegistrationHook
	now               func() time.Time
	accessTokenTTL    time.Duration
	refreshTokenTTL   time.Duration
}

func NewAuthService(cfg AuthConfig, deps AuthDependencies) (*AuthService, error) {
	switch {
	case cfg.AccessTokenTTL <= 0:
		return nil, fmt.Errorf("access token ttl must be positive")
	case cfg.RefreshTokenTTL <= 0:
		return nil, fmt.Errorf("refresh token ttl must be positive")
	case deps.PasswordHasher == nil:
		return nil, fmt.Errorf("password hasher is required")
	case deps.TokenProvider == nil:
		return nil, fmt.Errorf("token provider is required")
	case deps.UserRepository == nil:
		return nil, fmt.Errorf("user repository is required")
	case deps.SessionRepository == nil:
		return nil, fmt.Errorf("session repository is required")
	case deps.TxManager == nil:
		return nil, fmt.Errorf("transaction manager is required")
	}

	now := deps.Now
	if now == nil {
		now = time.Now
	}

	return &AuthService{
		passwordHasher:    deps.PasswordHasher,
		tokenProvider:     deps.TokenProvider,
		userRepository:    deps.UserRepository,
		sessionRepository: deps.SessionRepository,
		txManager:         deps.TxManager,
		registrationHook:  deps.RegistrationHook,
		now:               now,
		accessTokenTTL:    cfg.AccessTokenTTL,
		refreshTokenTTL:   cfg.RefreshTokenTTL,
	}, nil
}

func (s *AuthService) Register(ctx context.Context, params RegisterParams) (AuthenticationResult, error) {
	email := normalizeEmail(params.Email)
	if err := validateCredentialsInput(email, params.Password); err != nil {
		return AuthenticationResult{}, fmt.Errorf("register: %w", err)
	}

	passwordHash, err := s.passwordHasher.Hash(params.Password)
	if err != nil {
		return AuthenticationResult{}, mapInternalError("register: hash password", err)
	}

	refreshToken, err := s.tokenProvider.CreateRefreshToken()
	if err != nil {
		return AuthenticationResult{}, mapInternalError("register: create refresh token", err)
	}

	now := s.nowUTC()
	accessTokenExpiresAt := now.Add(s.accessTokenTTL)
	refreshTokenExpiresAt := now.Add(s.refreshTokenTTL)
	refreshTokenHash := s.tokenProvider.HashRefreshToken(refreshToken)

	var result AuthenticationResult
	err = s.txManager.WithinTx(ctx, func(ctx context.Context) error {
		user, err := s.userRepository.Create(ctx, CreateUserParams{
			Email:        email,
			PasswordHash: passwordHash,
		})
		if err != nil {
			return err
		}

		session, err := s.sessionRepository.Create(ctx, CreateSessionParams{
			UserID:           user.ID,
			RefreshTokenHash: refreshTokenHash,
			ExpiresAt:        refreshTokenExpiresAt,
			LastUsedAt:       now,
			UserAgent:        params.UserAgent,
		})
		if err != nil {
			return err
		}

		if s.registrationHook != nil {
			if err := s.registrationHook.AfterRegister(ctx, user); err != nil {
				return err
			}
		}

		accessToken, err := s.tokenProvider.CreateAccessToken(user.ID, session.ID, now, accessTokenExpiresAt)
		if err != nil {
			return err
		}

		result = AuthenticationResult{
			User:         user,
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		}

		return nil
	})
	if err != nil {
		return AuthenticationResult{}, mapAuthError("register", err)
	}

	return result, nil
}

func (s *AuthService) Login(ctx context.Context, params LoginParams) (AuthenticationResult, error) {
	email := normalizeEmail(params.Email)
	if err := validateCredentialsInput(email, params.Password); err != nil {
		return AuthenticationResult{}, fmt.Errorf("login: %w", err)
	}

	authUser, err := s.userRepository.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return AuthenticationResult{}, fmt.Errorf("login: %w", ErrInvalidCredentials)
		}
		return AuthenticationResult{}, mapAuthError("login", err)
	}

	passwordMatches, err := s.passwordHasher.Matches(authUser.PasswordHash, params.Password)
	if err != nil {
		return AuthenticationResult{}, mapInternalError("login: compare password", err)
	}
	if !passwordMatches {
		return AuthenticationResult{}, fmt.Errorf("login: %w", ErrInvalidCredentials)
	}

	refreshToken, err := s.tokenProvider.CreateRefreshToken()
	if err != nil {
		return AuthenticationResult{}, mapInternalError("login: create refresh token", err)
	}

	now := s.nowUTC()
	accessTokenExpiresAt := now.Add(s.accessTokenTTL)
	refreshTokenExpiresAt := now.Add(s.refreshTokenTTL)
	refreshTokenHash := s.tokenProvider.HashRefreshToken(refreshToken)

	var result AuthenticationResult
	err = s.txManager.WithinTx(ctx, func(ctx context.Context) error {
		session, err := s.sessionRepository.Create(ctx, CreateSessionParams{
			UserID:           authUser.ID,
			RefreshTokenHash: refreshTokenHash,
			ExpiresAt:        refreshTokenExpiresAt,
			LastUsedAt:       now,
			UserAgent:        params.UserAgent,
		})
		if err != nil {
			return err
		}

		accessToken, err := s.tokenProvider.CreateAccessToken(authUser.ID, session.ID, now, accessTokenExpiresAt)
		if err != nil {
			return err
		}

		result = AuthenticationResult{
			User:         authUser.User,
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		}

		return nil
	})
	if err != nil {
		return AuthenticationResult{}, mapAuthError("login", err)
	}

	return result, nil
}

func (s *AuthService) Refresh(ctx context.Context, params RefreshParams) (RefreshResult, error) {
	if strings.TrimSpace(params.RefreshToken) == "" {
		return RefreshResult{}, fmt.Errorf("refresh: %w", ErrUnauthorized)
	}

	now := s.nowUTC()
	refreshTokenHash := s.tokenProvider.HashRefreshToken(params.RefreshToken)

	session, err := s.sessionRepository.GetActiveByRefreshTokenHash(ctx, refreshTokenHash, now)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return RefreshResult{}, fmt.Errorf("refresh: %w", ErrSessionExpired)
		}
		return RefreshResult{}, mapAuthError("refresh", err)
	}

	newRefreshToken, err := s.tokenProvider.CreateRefreshToken()
	if err != nil {
		return RefreshResult{}, mapInternalError("refresh: create refresh token", err)
	}

	newRefreshTokenHash := s.tokenProvider.HashRefreshToken(newRefreshToken)
	accessTokenExpiresAt := now.Add(s.accessTokenTTL)
	refreshTokenExpiresAt := now.Add(s.refreshTokenTTL)

	var result RefreshResult
	err = s.txManager.WithinTx(ctx, func(ctx context.Context) error {
		rotatedSession, err := s.sessionRepository.Rotate(ctx, RotateSessionParams{
			SessionID:           session.ID,
			OldRefreshTokenHash: refreshTokenHash,
			NewRefreshTokenHash: newRefreshTokenHash,
			NewExpiresAt:        refreshTokenExpiresAt,
			LastUsedAt:          now,
		})
		if err != nil {
			if errors.Is(err, ErrSessionNotFound) {
				return ErrSessionExpired
			}
			return err
		}

		accessToken, err := s.tokenProvider.CreateAccessToken(rotatedSession.UserID, rotatedSession.ID, now, accessTokenExpiresAt)
		if err != nil {
			return err
		}

		result = RefreshResult{
			AccessToken:  accessToken,
			RefreshToken: newRefreshToken,
		}

		return nil
	})
	if err != nil {
		return RefreshResult{}, mapAuthError("refresh", err)
	}

	return result, nil
}

func (s *AuthService) Logout(ctx context.Context, params LogoutParams) error {
	if strings.TrimSpace(params.RefreshToken) == "" {
		return fmt.Errorf("logout: %w", ErrUnauthorized)
	}

	now := s.nowUTC()
	refreshTokenHash := s.tokenProvider.HashRefreshToken(params.RefreshToken)

	session, err := s.sessionRepository.GetActiveByRefreshTokenHash(ctx, refreshTokenHash, now)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return fmt.Errorf("logout: %w", ErrUnauthorized)
		}
		return mapAuthError("logout", err)
	}

	if err := s.sessionRepository.Revoke(ctx, session.ID, now); err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return fmt.Errorf("logout: %w", ErrUnauthorized)
		}
		return mapAuthError("logout", err)
	}

	return nil
}

func (s *AuthService) GetCurrentUser(ctx context.Context, params GetCurrentUserParams) (model.User, error) {
	if params.AuthenticatedUser.UserID == uuid.Nil {
		return model.User{}, fmt.Errorf("get current user: %w", ErrUnauthorized)
	}

	user, err := s.userRepository.GetByID(ctx, params.AuthenticatedUser.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return model.User{}, fmt.Errorf("get current user: %w", ErrUnauthorized)
		}
		return model.User{}, mapAuthError("get current user", err)
	}

	return user, nil
}

func (s *AuthService) nowUTC() time.Time {
	return s.now().UTC()
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validateCredentialsInput(email, password string) error {
	if email == "" || strings.TrimSpace(password) == "" {
		return ErrInvalidInput
	}

	return nil
}

func mapAuthError(operation string, err error) error {
	switch {
	case errors.Is(err, ErrInvalidInput),
		errors.Is(err, ErrEmailAlreadyExists),
		errors.Is(err, ErrInvalidCredentials),
		errors.Is(err, ErrUnauthorized),
		errors.Is(err, ErrSessionExpired):
		return fmt.Errorf("%s: %w", operation, err)
	case errors.Is(err, ErrUnexpectedStorage):
		return fmt.Errorf("%s: %w", operation, ErrInternal)
	default:
		return fmt.Errorf("%s: %w", operation, ErrInternal)
	}
}

func mapInternalError(operation string, err error) error {
	return fmt.Errorf("%s: %w", operation, ErrInternal)
}

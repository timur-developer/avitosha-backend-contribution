package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
)

var ErrSessionNotFound = errors.New("session not found")

type CreateSessionParams struct {
	UserID           uuid.UUID
	RefreshTokenHash []byte
	ExpiresAt        time.Time
	LastUsedAt       time.Time
	UserAgent        *string
}

type RotateSessionParams struct {
	SessionID           uuid.UUID
	OldRefreshTokenHash []byte
	NewRefreshTokenHash []byte
	NewExpiresAt        time.Time
	LastUsedAt          time.Time
}

type AccessTokenVerifier interface {
	VerifyAccessToken(token string) (model.AuthenticatedUser, error)
}

type SessionRepository interface {
	Create(ctx context.Context, params CreateSessionParams) (model.Session, error)
	GetActiveByRefreshTokenHash(ctx context.Context, refreshTokenHash []byte, now time.Time) (model.Session, error)
	GetActiveByIDAndUserID(ctx context.Context, sessionID, userID uuid.UUID, now time.Time) (model.Session, error)
	Rotate(ctx context.Context, params RotateSessionParams) (model.Session, error)
	Revoke(ctx context.Context, sessionID uuid.UUID, revokedAt time.Time) error
}

type AccessTokenAuthDependencies struct {
	AccessTokenVerifier AccessTokenVerifier
	SessionRepository   SessionRepository
	Now                 func() time.Time
}

type AccessTokenAuthService struct {
	accessTokenVerifier AccessTokenVerifier
	sessionRepository   SessionRepository
	now                 func() time.Time
}

func NewAccessTokenAuthService(deps AccessTokenAuthDependencies) (*AccessTokenAuthService, error) {
	switch {
	case deps.AccessTokenVerifier == nil:
		return nil, fmt.Errorf("access token verifier is required")
	case deps.SessionRepository == nil:
		return nil, fmt.Errorf("session repository is required")
	}

	now := deps.Now
	if now == nil {
		now = time.Now
	}

	return &AccessTokenAuthService{
		accessTokenVerifier: deps.AccessTokenVerifier,
		sessionRepository:   deps.SessionRepository,
		now:                 now,
	}, nil
}

func (s *AccessTokenAuthService) AuthenticateAccessToken(ctx context.Context, token string) (model.AuthenticatedUser, error) {
	authenticatedUser, err := s.accessTokenVerifier.VerifyAccessToken(token)
	if err != nil {
		return model.AuthenticatedUser{}, fmt.Errorf("authenticate access token: %w", ErrUnauthorized)
	}
	if authenticatedUser.UserID == uuid.Nil || authenticatedUser.SessionID == uuid.Nil {
		return model.AuthenticatedUser{}, fmt.Errorf("authenticate access token: %w", ErrUnauthorized)
	}

	_, err = s.sessionRepository.GetActiveByIDAndUserID(
		ctx,
		authenticatedUser.SessionID,
		authenticatedUser.UserID,
		s.now().UTC(),
	)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return model.AuthenticatedUser{}, fmt.Errorf("authenticate access token: %w", ErrUnauthorized)
		}
		return model.AuthenticatedUser{}, fmt.Errorf("authenticate access token: %w", ErrInternal)
	}

	return authenticatedUser, nil
}

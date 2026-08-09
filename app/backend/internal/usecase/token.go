package usecase

import (
	"time"

	"github.com/google/uuid"
)

type TokenProvider interface {
	CreateAccessToken(userID, sessionID uuid.UUID, issuedAt, expiresAt time.Time) (string, error)
	CreateRefreshToken() (string, error)
	HashRefreshToken(token string) []byte
}

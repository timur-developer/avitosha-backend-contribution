package model

import "github.com/google/uuid"

type AuthenticatedUser struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
}

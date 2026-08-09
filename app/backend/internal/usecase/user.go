package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
)

type CreateUserParams struct {
	Email        string
	PasswordHash string
}

type UserRepository interface {
	Create(ctx context.Context, params CreateUserParams) (model.User, error)
	GetByEmail(ctx context.Context, email string) (model.AuthUser, error)
	GetByID(ctx context.Context, id uuid.UUID) (model.User, error)
}

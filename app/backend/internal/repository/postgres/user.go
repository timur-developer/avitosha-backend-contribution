package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

const uniqueViolationCode = "23505"

type UserRepository struct {
	executor QueryExecutor
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{executor: pool}
}

func (r *UserRepository) Create(ctx context.Context, params usecase.CreateUserParams) (model.User, error) {
	user, err := scanUser(executorFromContext(ctx, r.executor).QueryRow(ctx, `
INSERT INTO users (email, password_hash)
VALUES ($1, $2)
RETURNING id, email, created_at, updated_at
`, params.Email, params.PasswordHash))
	if err != nil {
		return model.User{}, mapUserError("create user", err)
	}

	return user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (model.AuthUser, error) {
	authUser, err := scanAuthUser(executorFromContext(ctx, r.executor).QueryRow(ctx, `
SELECT id, email, password_hash, created_at, updated_at
FROM users
WHERE email = $1
`, email))
	if err != nil {
		return model.AuthUser{}, mapUserError("get user by email", err)
	}

	return authUser, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (model.User, error) {
	user, err := scanUser(executorFromContext(ctx, r.executor).QueryRow(ctx, `
SELECT id, email, created_at, updated_at
FROM users
WHERE id = $1
`, id))
	if err != nil {
		return model.User{}, mapUserError("get user by id", err)
	}

	return user, nil
}

func scanUser(row pgx.Row) (model.User, error) {
	var user model.User

	if err := row.Scan(
		&user.ID,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return model.User{}, err
	}

	return user, nil
}

func scanAuthUser(row pgx.Row) (model.AuthUser, error) {
	var authUser model.AuthUser

	if err := row.Scan(
		&authUser.ID,
		&authUser.Email,
		&authUser.PasswordHash,
		&authUser.CreatedAt,
		&authUser.UpdatedAt,
	); err != nil {
		return model.AuthUser{}, err
	}

	return authUser, nil
}

func mapUserError(operation string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s: %w", operation, usecase.ErrUserNotFound)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
		return fmt.Errorf("%s: %w", operation, usecase.ErrEmailAlreadyExists)
	}

	return fmt.Errorf("%s: %w", operation, usecase.ErrUnexpectedStorage)
}

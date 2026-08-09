package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/repository/postgres"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

func TestTxManagerRollsBackNestedOperations(t *testing.T) {
	pool := newTestPool(t)
	userRepository := postgres.NewUserRepository(pool)
	sessionRepository := postgres.NewSessionRepository(pool)
	txManager := postgres.NewTxManager(pool)

	now := time.Now().UTC().Truncate(time.Microsecond)
	expectedErr := errors.New("force rollback")

	err := txManager.WithinTx(context.Background(), func(ctx context.Context) error {
		user, err := userRepository.Create(ctx, usecase.CreateUserParams{
			Email:        "tx-user@example.com",
			PasswordHash: "hashed-password",
		})
		if err != nil {
			return err
		}

		return txManager.WithinTx(ctx, func(ctx context.Context) error {
			_, err := sessionRepository.Create(ctx, usecase.CreateSessionParams{
				UserID:           user.ID,
				RefreshTokenHash: []byte("tx-session-hash"),
				ExpiresAt:        now.Add(24 * time.Hour),
				LastUsedAt:       now,
			})
			if err != nil {
				return err
			}

			return expectedErr
		})
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("WithinTx() error = %v, want expected rollback error", err)
	}

	_, err = userRepository.GetByEmail(context.Background(), "tx-user@example.com")
	if !errors.Is(err, usecase.ErrUserNotFound) {
		t.Fatalf("GetByEmail() after rollback error = %v, want ErrUserNotFound", err)
	}

	_, err = sessionRepository.GetActiveByRefreshTokenHash(context.Background(), []byte("tx-session-hash"), now.Add(time.Minute))
	if !errors.Is(err, usecase.ErrSessionNotFound) {
		t.Fatalf("GetActiveByRefreshTokenHash() after rollback error = %v, want ErrSessionNotFound", err)
	}
}

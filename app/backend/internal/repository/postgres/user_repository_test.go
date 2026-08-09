package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/repository/postgres"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

func TestUserRepositoryCreateAndGet(t *testing.T) {
	pool := newTestPool(t)
	repository := postgres.NewUserRepository(pool)

	createdUser, err := repository.Create(context.Background(), usecase.CreateUserParams{
		Email:        "user@example.com",
		PasswordHash: "hashed-password",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if createdUser.ID == uuid.Nil {
		t.Fatal("Create() returned empty user ID")
	}
	if createdUser.Email != "user@example.com" {
		t.Fatalf("Create() email = %q", createdUser.Email)
	}
	if createdUser.CreatedAt.IsZero() {
		t.Fatal("Create() returned zero CreatedAt")
	}
	if createdUser.UpdatedAt.IsZero() {
		t.Fatal("Create() returned zero UpdatedAt")
	}

	userByEmail, err := repository.GetByEmail(context.Background(), "user@example.com")
	if err != nil {
		t.Fatalf("GetByEmail() error = %v", err)
	}
	if userByEmail.ID != createdUser.ID {
		t.Fatalf("GetByEmail() user ID = %s, want %s", userByEmail.ID, createdUser.ID)
	}
	if userByEmail.PasswordHash != "hashed-password" {
		t.Fatalf("GetByEmail() password hash = %q", userByEmail.PasswordHash)
	}

	userByID, err := repository.GetByID(context.Background(), createdUser.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if userByID.Email != createdUser.Email {
		t.Fatalf("GetByID() email = %q, want %q", userByID.Email, createdUser.Email)
	}
}

func TestUserRepositoryCreateDuplicateEmail(t *testing.T) {
	pool := newTestPool(t)
	repository := postgres.NewUserRepository(pool)

	_, err := repository.Create(context.Background(), usecase.CreateUserParams{
		Email:        "duplicate@example.com",
		PasswordHash: "hash-1",
	})
	if err != nil {
		t.Fatalf("Create() first call error = %v", err)
	}

	_, err = repository.Create(context.Background(), usecase.CreateUserParams{
		Email:        "duplicate@example.com",
		PasswordHash: "hash-2",
	})
	if !errors.Is(err, usecase.ErrEmailAlreadyExists) {
		t.Fatalf("Create() duplicate error = %v, want ErrEmailAlreadyExists", err)
	}
}

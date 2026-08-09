package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/repository/postgres"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

func TestSessionRepositoryCreateAndGetActive(t *testing.T) {
	pool := newTestPool(t)
	userRepository := postgres.NewUserRepository(pool)
	sessionRepository := postgres.NewSessionRepository(pool)

	user, err := userRepository.Create(context.Background(), usecase.CreateUserParams{
		Email:        "session-user@example.com",
		PasswordHash: "hashed-password",
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	session, err := sessionRepository.Create(context.Background(), usecase.CreateSessionParams{
		UserID:           user.ID,
		RefreshTokenHash: []byte("refresh-hash"),
		ExpiresAt:        now.Add(24 * time.Hour),
		LastUsedAt:       now,
		UserAgent:        stringPointer("integration-test"),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if session.ID.String() == "" {
		t.Fatal("Create() returned empty session ID")
	}
	if session.UserID != user.ID {
		t.Fatalf("Create() user ID = %s, want %s", session.UserID, user.ID)
	}
	if string(session.RefreshTokenHash) != "refresh-hash" {
		t.Fatalf("Create() refresh token hash = %q", string(session.RefreshTokenHash))
	}
	if session.UserAgent == nil || *session.UserAgent != "integration-test" {
		t.Fatalf("Create() user agent = %v", session.UserAgent)
	}

	activeSession, err := sessionRepository.GetActiveByRefreshTokenHash(context.Background(), []byte("refresh-hash"), now)
	if err != nil {
		t.Fatalf("GetActiveByRefreshTokenHash() error = %v", err)
	}
	if activeSession.ID != session.ID {
		t.Fatalf("GetActiveByRefreshTokenHash() session ID = %s, want %s", activeSession.ID, session.ID)
	}

	activeByID, err := sessionRepository.GetActiveByIDAndUserID(context.Background(), session.ID, user.ID, now)
	if err != nil {
		t.Fatalf("GetActiveByIDAndUserID() error = %v", err)
	}
	if activeByID.ID != session.ID {
		t.Fatalf("GetActiveByIDAndUserID() session ID = %s, want %s", activeByID.ID, session.ID)
	}
}

func TestSessionRepositoryRotate(t *testing.T) {
	pool := newTestPool(t)
	userRepository := postgres.NewUserRepository(pool)
	sessionRepository := postgres.NewSessionRepository(pool)

	user, err := userRepository.Create(context.Background(), usecase.CreateUserParams{
		Email:        "rotate-user@example.com",
		PasswordHash: "hashed-password",
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	session, err := sessionRepository.Create(context.Background(), usecase.CreateSessionParams{
		UserID:           user.ID,
		RefreshTokenHash: []byte("old-hash"),
		ExpiresAt:        now.Add(24 * time.Hour),
		LastUsedAt:       now,
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	rotatedSession, err := sessionRepository.Rotate(context.Background(), usecase.RotateSessionParams{
		SessionID:           session.ID,
		OldRefreshTokenHash: []byte("old-hash"),
		NewRefreshTokenHash: []byte("new-hash"),
		NewExpiresAt:        now.Add(48 * time.Hour),
		LastUsedAt:          now.Add(5 * time.Minute),
	})
	if err != nil {
		t.Fatalf("Rotate() error = %v", err)
	}
	if string(rotatedSession.RefreshTokenHash) != "new-hash" {
		t.Fatalf("Rotate() refresh token hash = %q", string(rotatedSession.RefreshTokenHash))
	}

	_, err = sessionRepository.GetActiveByRefreshTokenHash(context.Background(), []byte("old-hash"), now.Add(5*time.Minute))
	if !errors.Is(err, usecase.ErrSessionNotFound) {
		t.Fatalf("GetActiveByRefreshTokenHash(old) error = %v, want ErrSessionNotFound", err)
	}

	_, err = sessionRepository.Rotate(context.Background(), usecase.RotateSessionParams{
		SessionID:           session.ID,
		OldRefreshTokenHash: []byte("old-hash"),
		NewRefreshTokenHash: []byte("another-hash"),
		NewExpiresAt:        now.Add(72 * time.Hour),
		LastUsedAt:          now.Add(10 * time.Minute),
	})
	if !errors.Is(err, usecase.ErrSessionNotFound) {
		t.Fatalf("Rotate() with old hash error = %v, want ErrSessionNotFound", err)
	}

	activeSession, err := sessionRepository.GetActiveByRefreshTokenHash(context.Background(), []byte("new-hash"), now.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("GetActiveByRefreshTokenHash(new) error = %v", err)
	}
	if activeSession.ID != session.ID {
		t.Fatalf("GetActiveByRefreshTokenHash(new) session ID = %s, want %s", activeSession.ID, session.ID)
	}
}

func TestSessionRepositoryRevoke(t *testing.T) {
	pool := newTestPool(t)
	userRepository := postgres.NewUserRepository(pool)
	sessionRepository := postgres.NewSessionRepository(pool)

	user, err := userRepository.Create(context.Background(), usecase.CreateUserParams{
		Email:        "revoke-user@example.com",
		PasswordHash: "hashed-password",
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	session, err := sessionRepository.Create(context.Background(), usecase.CreateSessionParams{
		UserID:           user.ID,
		RefreshTokenHash: []byte("revoke-hash"),
		ExpiresAt:        now.Add(24 * time.Hour),
		LastUsedAt:       now,
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if err := sessionRepository.Revoke(context.Background(), session.ID, now.Add(time.Minute)); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	if err := sessionRepository.Revoke(context.Background(), session.ID, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("Revoke() second call error = %v", err)
	}

	_, err = sessionRepository.GetActiveByRefreshTokenHash(context.Background(), []byte("revoke-hash"), now.Add(2*time.Minute))
	if !errors.Is(err, usecase.ErrSessionNotFound) {
		t.Fatalf("GetActiveByRefreshTokenHash() error = %v, want ErrSessionNotFound", err)
	}

	_, err = sessionRepository.GetActiveByIDAndUserID(context.Background(), session.ID, user.ID, now.Add(2*time.Minute))
	if !errors.Is(err, usecase.ErrSessionNotFound) {
		t.Fatalf("GetActiveByIDAndUserID() error = %v, want ErrSessionNotFound", err)
	}
}

func TestSessionRepositoryExpiredSessionIsNotActive(t *testing.T) {
	pool := newTestPool(t)
	userRepository := postgres.NewUserRepository(pool)
	sessionRepository := postgres.NewSessionRepository(pool)

	user, err := userRepository.Create(context.Background(), usecase.CreateUserParams{
		Email:        "expired-session@example.com",
		PasswordHash: "hashed-password",
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	_, err = sessionRepository.Create(context.Background(), usecase.CreateSessionParams{
		UserID:           user.ID,
		RefreshTokenHash: []byte("expired-hash"),
		ExpiresAt:        now.Add(-time.Minute),
		LastUsedAt:       now.Add(-2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	_, err = sessionRepository.GetActiveByRefreshTokenHash(context.Background(), []byte("expired-hash"), now)
	if !errors.Is(err, usecase.ErrSessionNotFound) {
		t.Fatalf("GetActiveByRefreshTokenHash() error = %v, want ErrSessionNotFound", err)
	}

	_, err = sessionRepository.GetActiveByIDAndUserID(context.Background(), uuid.New(), user.ID, now)
	if !errors.Is(err, usecase.ErrSessionNotFound) {
		t.Fatalf("GetActiveByIDAndUserID() missing session error = %v, want ErrSessionNotFound", err)
	}
}

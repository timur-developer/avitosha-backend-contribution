package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
)

func TestAccessTokenAuthServiceAuthenticateAccessToken(t *testing.T) {
	userID := uuid.New()
	sessionID := uuid.New()
	now := testNow()
	store := newFakeAuthStore()

	sessionRepository := &fakeSessionRepository{store: store}
	if _, err := sessionRepository.Create(context.Background(), CreateSessionParams{
		UserID:           userID,
		RefreshTokenHash: []byte("refresh-hash"),
		ExpiresAt:        now.Add(24 * time.Hour),
		LastUsedAt:       now,
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	session := store.sessionsByID[mustOnlySessionID(t, store)]
	session.ID = sessionID
	store.sessionsByID = map[uuid.UUID]model.Session{sessionID: session}

	service, err := NewAccessTokenAuthService(AccessTokenAuthDependencies{
		AccessTokenVerifier: fakeAccessTokenVerifier{
			verifyFunc: func(token string) (model.AuthenticatedUser, error) {
				if token != "valid-token" {
					t.Fatalf("VerifyAccessToken() token = %q, want valid-token", token)
				}
				return model.AuthenticatedUser{UserID: userID, SessionID: sessionID}, nil
			},
		},
		SessionRepository: sessionRepository,
		Now:               func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewAccessTokenAuthService() error = %v", err)
	}

	authenticatedUser, err := service.AuthenticateAccessToken(context.Background(), "valid-token")
	if err != nil {
		t.Fatalf("AuthenticateAccessToken() error = %v", err)
	}
	if authenticatedUser.UserID != userID || authenticatedUser.SessionID != sessionID {
		t.Fatalf("AuthenticateAccessToken() user = %#v", authenticatedUser)
	}
}

func TestAccessTokenAuthServiceAuthenticateAccessTokenRejectsInactiveSession(t *testing.T) {
	userID := uuid.New()
	sessionID := uuid.New()
	now := testNow()
	store := newFakeAuthStore()

	service, err := NewAccessTokenAuthService(AccessTokenAuthDependencies{
		AccessTokenVerifier: fakeAccessTokenVerifier{
			verifyFunc: func(string) (model.AuthenticatedUser, error) {
				return model.AuthenticatedUser{UserID: userID, SessionID: sessionID}, nil
			},
		},
		SessionRepository: &fakeSessionRepository{store: store},
		Now:               func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewAccessTokenAuthService() error = %v", err)
	}

	_, err = service.AuthenticateAccessToken(context.Background(), "stale-token")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("AuthenticateAccessToken() error = %v, want ErrUnauthorized", err)
	}
}

func TestAccessTokenAuthServiceAuthenticateAccessTokenMapsStorageFailure(t *testing.T) {
	userID := uuid.New()
	sessionID := uuid.New()

	service, err := NewAccessTokenAuthService(AccessTokenAuthDependencies{
		AccessTokenVerifier: fakeAccessTokenVerifier{
			verifyFunc: func(string) (model.AuthenticatedUser, error) {
				return model.AuthenticatedUser{UserID: userID, SessionID: sessionID}, nil
			},
		},
		SessionRepository: &fakeSessionRepository{
			store:        newFakeAuthStore(),
			getActiveErr: ErrUnexpectedStorage,
		},
	})
	if err != nil {
		t.Fatalf("NewAccessTokenAuthService() error = %v", err)
	}

	_, err = service.AuthenticateAccessToken(context.Background(), "valid-token")
	if !errors.Is(err, ErrInternal) {
		t.Fatalf("AuthenticateAccessToken() error = %v, want ErrInternal", err)
	}
}

type fakeAccessTokenVerifier struct {
	verifyFunc func(string) (model.AuthenticatedUser, error)
}

func (f fakeAccessTokenVerifier) VerifyAccessToken(token string) (model.AuthenticatedUser, error) {
	if f.verifyFunc == nil {
		return model.AuthenticatedUser{}, nil
	}
	return f.verifyFunc(token)
}

func mustOnlySessionID(t *testing.T, store *fakeAuthStore) uuid.UUID {
	t.Helper()

	for sessionID := range store.sessionsByID {
		return sessionID
	}

	t.Fatal("store.sessionsByID is empty")
	return uuid.Nil
}

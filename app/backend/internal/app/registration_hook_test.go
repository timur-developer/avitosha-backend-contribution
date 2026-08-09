package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

type fakeGameProfileBootstrapper struct {
	userID uuid.UUID
	now    time.Time
	err    error
	calls  int
}

func (f *fakeGameProfileBootstrapper) EnsureProfile(
	_ context.Context,
	userID uuid.UUID,
	now time.Time,
) (usecase.GameProfile, error) {
	f.calls++
	f.userID = userID
	f.now = now
	if f.err != nil {
		return usecase.GameProfile{}, f.err
	}
	return usecase.GameProfile{}, nil
}

func TestNewRegistrationHookReturnsNilWithoutBootstrapper(t *testing.T) {
	hook := newRegistrationHook(nil, time.Now)

	if hook != nil {
		t.Fatal("registration hook should be nil without bootstrapper")
	}
}

func TestRegistrationHookBootstrapsGameProfileAfterRegister(t *testing.T) {
	now := time.Date(2026, time.August, 6, 12, 30, 0, 0, time.FixedZone("UTC+5", 5*60*60))
	bootstrapper := &fakeGameProfileBootstrapper{}
	hook := newRegistrationHook(bootstrapper, func() time.Time { return now })
	user := model.User{ID: uuid.New()}

	if err := hook.AfterRegister(context.Background(), user); err != nil {
		t.Fatalf("AfterRegister() error = %v", err)
	}
	if bootstrapper.calls != 1 {
		t.Fatalf("EnsureProfile() calls = %d, want 1", bootstrapper.calls)
	}
	if bootstrapper.userID != user.ID {
		t.Fatalf("EnsureProfile() user ID = %s, want %s", bootstrapper.userID, user.ID)
	}
	if !bootstrapper.now.Equal(now.UTC()) {
		t.Fatalf("EnsureProfile() now = %v, want %v", bootstrapper.now, now.UTC())
	}
}

func TestRegistrationHookWrapsBootstrapError(t *testing.T) {
	bootstrapper := &fakeGameProfileBootstrapper{err: usecase.ErrUnexpectedStorage}
	hook := newRegistrationHook(bootstrapper, time.Now)

	err := hook.AfterRegister(context.Background(), model.User{ID: uuid.New()})
	if err == nil {
		t.Fatal("AfterRegister() error = nil, want wrapped error")
	}
	if !errors.Is(err, usecase.ErrUnexpectedStorage) {
		t.Fatalf("AfterRegister() error = %v, want wrapped ErrUnexpectedStorage", err)
	}
}

package app

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

type gameProfileBootstrapper interface {
	EnsureProfile(ctx context.Context, userID uuid.UUID, now time.Time) (usecase.GameProfile, error)
}

type registrationHook struct {
	bootstrapper gameProfileBootstrapper
	now          func() time.Time
}

func newRegistrationHook(
	bootstrapper gameProfileBootstrapper,
	now func() time.Time,
) usecase.RegistrationHook {
	if bootstrapper == nil {
		return nil
	}
	if now == nil {
		now = time.Now
	}
	return registrationHook{bootstrapper: bootstrapper, now: now}
}

func (hook registrationHook) AfterRegister(ctx context.Context, user model.User) error {
	if _, err := hook.bootstrapper.EnsureProfile(ctx, user.ID, hook.now().UTC()); err != nil {
		return fmt.Errorf("bootstrap game profile after register: %w", err)
	}
	return nil
}

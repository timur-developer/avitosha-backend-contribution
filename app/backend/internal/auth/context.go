package auth

import (
	"context"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
)

type authenticatedUserContextKey struct{}

func ContextWithAuthenticatedUser(ctx context.Context, user model.AuthenticatedUser) context.Context {
	return context.WithValue(ctx, authenticatedUserContextKey{}, user)
}

func AuthenticatedUserFromContext(ctx context.Context) (model.AuthenticatedUser, bool) {
	user, ok := ctx.Value(authenticatedUserContextKey{}).(model.AuthenticatedUser)
	if !ok {
		return model.AuthenticatedUser{}, false
	}
	if user.UserID == uuid.Nil || user.SessionID == uuid.Nil {
		return model.AuthenticatedUser{}, false
	}

	return user, true
}

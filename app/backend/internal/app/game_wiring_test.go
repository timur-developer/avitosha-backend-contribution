package app

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type fakeAppTxManager struct{}

func (fakeAppTxManager) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestNewGameServiceBuildsFromSharedInfrastructure(t *testing.T) {
	service := newGameService(&pgxpool.Pool{}, fakeAppTxManager{}, nil, nil)

	if service == nil {
		t.Fatal("game service is nil")
	}
}

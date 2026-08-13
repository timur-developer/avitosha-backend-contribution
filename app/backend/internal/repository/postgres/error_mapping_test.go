package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

func TestMapUserError(t *testing.T) {
	t.Run("maps not found", func(t *testing.T) {
		err := mapUserError("get user by email", pgx.ErrNoRows)
		if !errors.Is(err, usecase.ErrUserNotFound) {
			t.Fatalf("errors.Is(err, ErrUserNotFound) = false, err = %v", err)
		}
	})

	t.Run("maps unique violation", func(t *testing.T) {
		err := mapUserError("create user", &pgconn.PgError{Code: uniqueViolationCode})
		if !errors.Is(err, usecase.ErrEmailAlreadyExists) {
			t.Fatalf("errors.Is(err, ErrEmailAlreadyExists) = false, err = %v", err)
		}
	})

	t.Run("maps unexpected storage", func(t *testing.T) {
		err := mapUserError("create user", errors.New("driver exploded"))
		if !errors.Is(err, usecase.ErrUnexpectedStorage) {
			t.Fatalf("errors.Is(err, ErrUnexpectedStorage) = false, err = %v", err)
		}
	})
}

func TestMapSessionError(t *testing.T) {
	t.Run("maps not found", func(t *testing.T) {
		err := mapSessionError("get active session by refresh token hash", pgx.ErrNoRows)
		if !errors.Is(err, usecase.ErrSessionNotFound) {
			t.Fatalf("errors.Is(err, ErrSessionNotFound) = false, err = %v", err)
		}
	})

	t.Run("maps unexpected storage", func(t *testing.T) {
		err := mapSessionError("create session", errors.New("driver exploded"))
		if !errors.Is(err, usecase.ErrUnexpectedStorage) {
			t.Fatalf("errors.Is(err, ErrUnexpectedStorage) = false, err = %v", err)
		}
	})
}

func TestMapMarketplaceStorageError(t *testing.T) {
	t.Run("maps completed demo purchase", func(t *testing.T) {
		err := mapMarketplaceStorageError("create listing deal", &pgconn.PgError{
			Code:           uniqueViolationCode,
			ConstraintName: "listing_deals_listing_id_buyer_id_key",
		})
		if !errors.Is(err, usecase.ErrDemoPurchaseCompleted) {
			t.Fatalf("errors.Is(err, ErrDemoPurchaseCompleted) = false, err = %v", err)
		}
	})

	t.Run("does not map other unique violations as completed demo purchase", func(t *testing.T) {
		err := mapMarketplaceStorageError("create listing deal", &pgconn.PgError{
			Code:           uniqueViolationCode,
			ConstraintName: "listing_deals_pkey",
		})
		if errors.Is(err, usecase.ErrDemoPurchaseCompleted) {
			t.Fatalf("unexpected ErrDemoPurchaseCompleted, err = %v", err)
		}
		if !errors.Is(err, usecase.ErrUnexpectedStorage) {
			t.Fatalf("errors.Is(err, ErrUnexpectedStorage) = false, err = %v", err)
		}
	})
}

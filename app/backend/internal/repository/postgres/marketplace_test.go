package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/repository/postgres"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

func TestMarketplaceRepositoryKeepsFavoriteAndDailyViewUnique(t *testing.T) {
	pool := newTestPool(t)
	user := createMarketplaceTestUser(t, pool, "marketplace-reader@example.com")
	owner := createMarketplaceTestUser(t, pool, "marketplace-owner@example.com")
	repository := postgres.NewGameRepository(pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	listing := model.Listing{ID: uuid.New(), OwnerID: owner, CategoryCode: "FURNITURE", Title: "Lamp", Description: "Description", PriceKopecks: 100, Status: model.ListingStatusPublished, PublishedAt: &now, CreatedAt: now, UpdatedAt: now}
	if _, err := repository.CreateListing(context.Background(), listing); err != nil {
		t.Fatalf("CreateListing() error = %v", err)
	}
	firstFavorite, err := repository.AddListingFavorite(context.Background(), user, listing.ID, now)
	if err != nil || !firstFavorite {
		t.Fatalf("first favorite = %v, %v", firstFavorite, err)
	}
	secondFavorite, err := repository.AddListingFavorite(context.Background(), user, listing.ID, now)
	if err != nil || secondFavorite {
		t.Fatalf("second favorite = %v, %v", secondFavorite, err)
	}
	firstReward, err := repository.ClaimListingFavoriteReward(context.Background(), user, listing.ID, now)
	if err != nil || !firstReward {
		t.Fatalf("first favorite reward = %v, %v", firstReward, err)
	}
	if _, err = repository.RemoveListingFavorite(context.Background(), user, listing.ID); err != nil {
		t.Fatalf("remove favorite: %v", err)
	}
	if restored, err := repository.AddListingFavorite(context.Background(), user, listing.ID, now); err != nil || !restored {
		t.Fatalf("restore favorite = %v, %v", restored, err)
	}
	secondReward, err := repository.ClaimListingFavoriteReward(context.Background(), user, listing.ID, now)
	if err != nil || secondReward {
		t.Fatalf("second favorite reward = %v, %v", secondReward, err)
	}
	firstView, err := repository.RegisterListingView(context.Background(), user, listing.ID, now)
	if err != nil || !firstView {
		t.Fatalf("first view = %v, %v", firstView, err)
	}
	secondView, err := repository.RegisterListingView(context.Background(), user, listing.ID, now)
	if err != nil || secondView {
		t.Fatalf("second view = %v, %v", secondView, err)
	}
}

func createMarketplaceTestUser(t *testing.T, pool *pgxpool.Pool, email string) uuid.UUID {
	t.Helper()
	user, err := postgres.NewUserRepository(pool).Create(context.Background(), usecase.CreateUserParams{Email: email, PasswordHash: "hash"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user.ID
}

package usecase

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
)

func TestEvaluateListingQuality(t *testing.T) {
	tests := []struct {
		name     string
		listing  model.Listing
		score    int
		eligible bool
		missing  []string
	}{
		{name: "empty", listing: model.Listing{}, missing: []string{"price", "photo", "description"}},
		{name: "complete", listing: model.Listing{PriceKopecks: 100, Description: strings.Repeat("a", 150), Photos: []model.ListingPhoto{{URL: "https://example.test/photo.jpg"}}}, score: 3, eligible: true},
		{name: "short description is a recommendation", listing: model.Listing{PriceKopecks: 100, Description: "short", Photos: []model.ListingPhoto{{URL: "https://example.test/photo.jpg"}}}, score: 2, eligible: true, missing: []string{"description"}},
		{name: "photo and description are optional", listing: model.Listing{PriceKopecks: 100}, score: 1, eligible: true, missing: []string{"photo", "description"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quality := EvaluateListingQuality(tt.listing)
			if quality.Score != tt.score || quality.IsEligible != tt.eligible || len(quality.MissingFields) != len(tt.missing) {
				t.Fatalf("quality = %+v", quality)
			}
		})
	}
}

func TestCompleteListingPurchaseSupportsUserListings(t *testing.T) {
	ownerID, buyerID := uuid.New(), uuid.New()
	publishedAt := time.Date(2026, 8, 13, 9, 0, 0, 0, time.UTC)
	completedAt := publishedAt.Add(time.Hour)

	listing, shouldUpdate, err := completeListingPurchase(model.Listing{
		OwnerID: ownerID, Status: model.ListingStatusPublished,
		PublishedAt: &publishedAt,
	}, buyerID, completedAt)
	if err != nil {
		t.Fatalf("complete purchase: %v", err)
	}
	if !shouldUpdate || listing.Status != model.ListingStatusSold ||
		listing.SoldAt == nil || listing.PublishedAt != nil {
		t.Fatalf("purchased listing = %+v, shouldUpdate = %v", listing, shouldUpdate)
	}
}

func TestCompleteListingPurchaseKeepsDemoPublished(t *testing.T) {
	listing, shouldUpdate, err := completeListingPurchase(model.Listing{
		OwnerID: uuid.New(), Status: model.ListingStatusPublished, IsDemo: true,
	}, uuid.New(), time.Now())
	if err != nil || shouldUpdate || listing.Status != model.ListingStatusPublished {
		t.Fatalf("demo listing = %+v, shouldUpdate = %v, err = %v", listing, shouldUpdate, err)
	}
}

func TestCompleteListingPurchaseRejectsOwner(t *testing.T) {
	ownerID := uuid.New()
	_, _, err := completeListingPurchase(model.Listing{
		OwnerID: ownerID, Status: model.ListingStatusPublished,
	}, ownerID, time.Now())
	if err != ErrListingOwnAction {
		t.Fatalf("error = %v, want %v", err, ErrListingOwnAction)
	}
}

func TestValidateListingInput(t *testing.T) {
	if err := validateListingInput("FURNITURE", "Lamp", "", 0, nil); err != nil {
		t.Fatalf("draft without description or photo: %v", err)
	}
	if err := validateListingInput("FURNITURE", "Lamp", "", 1, []string{"https://example.test/photo.jpg"}); err != nil {
		t.Fatalf("valid input: %v", err)
	}
	if err := validateListingInput("FURNITURE", "Lamp", "", 1, []string{"/storage/avitosha-photos/listing-photos/user/photo.jpg"}); err != nil {
		t.Fatalf("valid MinIO photo URL: %v", err)
	}
	if err := validateListingInput("FURNITURE", "Lamp", "", 1, []string{"not-a-url"}); err == nil {
		t.Fatal("invalid photo URL was accepted")
	}
	if err := validateListingInput("FURNITURE", "Lamp", "", 1, []string{"/api/v1/me"}); err == nil {
		t.Fatal("non-storage relative URL was accepted")
	}
}

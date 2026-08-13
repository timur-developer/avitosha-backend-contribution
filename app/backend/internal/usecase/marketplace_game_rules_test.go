package usecase

import (
	"encoding/json"
	"testing"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
)

func TestMarketplaceActionTypesDriveExpectedCharacterScores(t *testing.T) {
	tests := []struct {
		action                             model.ActionType
		wantBuyer, wantSeller, wantQuality int
	}{
		{model.ActionTypeAdViewed, 1, 0, 0}, {model.ActionTypeAdFavorited, 1, 0, 0},
		{model.ActionTypeMessageSent, 1, 0, 0}, {model.ActionTypeAdCreated, 0, 1, 0},
		{model.ActionTypeListingSold, 0, 1, 0}, {model.ActionTypeListingImproved, 0, 0, 1},
	}
	for _, test := range tests {
		got := ActivityDelta(test.action, nil)
		if got.Buyer != test.wantBuyer || got.Seller != test.wantSeller || got.Quality != test.wantQuality {
			t.Errorf("%s: got %+v", test.action, got)
		}
	}
}

func TestMarketplaceMetadataIsServerMarker(t *testing.T) {
	if !isMarketplaceAction(json.RawMessage(`{"source":"marketplace.purchase"}`)) {
		t.Fatal("marketplace source must enable server product effects")
	}
	if isMarketplaceAction(json.RawMessage(`{"source":"client"}`)) {
		t.Fatal("client action must not receive marketplace effects")
	}
	if isTrustedMarketplaceAction(json.RawMessage(`{"source":"marketplace.purchase"}`), true) {
		t.Fatal("public action pipeline must not trust a client-supplied marketplace marker")
	}
	if !isTrustedMarketplaceAction(json.RawMessage(`{"source":"marketplace.purchase"}`), false) {
		t.Fatal("caller-owned marketplace transaction must preserve product effects")
	}
}

func TestQualityCriteriaAreAwardedOnlyOnNewCompletion(t *testing.T) {
	old := ListingQuality{MissingFields: []string{"price", "description"}}
	current := ListingQuality{MissingFields: []string{"description"}}
	got := newQualityCriteria(old, current)
	if len(got) != 1 || got[0] != "price" {
		t.Fatalf("criteria = %v", got)
	}
}

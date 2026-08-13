package handler

import (
	"errors"
	"net/http"
	"testing"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

func TestMapMarketplaceErrorCompletedDemoPurchase(t *testing.T) {
	status, code, _ := mapMarketplaceError(usecase.ErrDemoPurchaseCompleted)
	if status != http.StatusConflict || code != "demo_purchase_already_completed" {
		t.Fatalf("mapMarketplaceError() = (%d, %q), want (409, demo_purchase_already_completed)", status, code)
	}

	status, _, _ = mapMarketplaceError(errors.New("unexpected"))
	if status != http.StatusInternalServerError {
		t.Fatalf("unexpected error status = %d, want 500", status)
	}
}

package rpc

import (
	"errors"
	"testing"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

func TestDemoPurchaseCompletedRoundTripsThroughGRPC(t *testing.T) {
	decoded := DecodeGameError(GameError(usecase.ErrDemoPurchaseCompleted))
	if !errors.Is(decoded, usecase.ErrDemoPurchaseCompleted) {
		t.Fatalf("DecodeGameError(GameError()) = %v, want ErrDemoPurchaseCompleted", decoded)
	}
}

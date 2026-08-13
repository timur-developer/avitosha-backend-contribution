package gameclient_test

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/client/grpc/gameclient"
	avitoshav1 "github.com/guitaramust-sudo/Avitosha/app/backend/internal/gen/avitosha/v1"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/handler"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/realtime"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/transport/grpc/gameserver"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

func TestPublishPreservesActionResultThroughGRPCClientAndHTTPGateway(t *testing.T) {
	userID, listingID, eventID, actionID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	avitoshav1.RegisterGameServiceServer(server, gameserver.New(gameStub{}, marketplaceStub{result: usecase.MarketplaceActionResult{
		Listing:      &model.Listing{ID: listingID, OwnerID: userID, CategoryCode: "FURNITURE", Title: "Desk", Status: model.ListingStatusPublished},
		ActionResult: &usecase.ProcessActionResult{ActionID: actionID, Events: []model.DomainEvent{{ID: uuid.New(), Type: model.DomainEventListingPublished}}},
	}}, realtime.NewHub(1)))
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)

	connection, err := grpc.NewClient("passthrough:///marketplace", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("create grpc client: %v", err)
	}
	t.Cleanup(func() { _ = connection.Close() })

	marketplace := gameclient.New(avitoshav1.NewGameServiceClient(connection))
	marketplaceHandler := handler.NewMarketplaceHandler(nil, marketplace, time.Now)
	router := chi.NewRouter()
	router.With(handler.GameIdentity(nil, nil)).Post("/api/v1/listings/{listing_id}/publish", marketplaceHandler.Publish)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/listings/"+listingID.String()+"/publish", strings.NewReader(`{"eventId":"`+eventID.String()+`"}`))
	request.Header.Set("X-User-ID", userID.String())
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		ActionResult *usecase.ProcessActionResult `json:"actionResult"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode HTTP response: %v", err)
	}
	if response.ActionResult == nil || response.ActionResult.ActionID != actionID || response.ActionResult.Duplicate || len(response.ActionResult.Events) != 1 {
		t.Fatalf("actionResult lost in gateway response: %s", recorder.Body.String())
	}
}

type gameStub struct{ gameserver.GameUseCase }

type marketplaceStub struct {
	gameserver.MarketplaceUseCase
	result usecase.MarketplaceActionResult
}

func (stub marketplaceStub) PublishWithGame(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, time.Time) (usecase.MarketplaceActionResult, error) {
	return stub.result, nil
}

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/realtime"
)

func TestWebSocketDeliversUserEventsAndRemovesDisconnectedClient(t *testing.T) {
	userID := uuid.New()
	hub := realtime.NewHub(4)
	router := newGameTestRouter(RouterDependencies{EventHub: hub})
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	header := http.Header{}
	header.Set("X-User-ID", userID.String())
	connection, response, err := websocket.DefaultDialer.Dial(
		"ws"+strings.TrimPrefix(server.URL, "http")+"/api/v1/ws", header,
	)
	if response != nil {
		defer func() { _ = response.Body.Close() }()
	}
	if err != nil {
		if response != nil {
			t.Fatalf("websocket dial: %v; status = %d", err, response.StatusCode)
		}
		t.Fatalf("websocket dial: %v", err)
	}

	deadline := time.Now().Add(time.Second)
	for hub.SubscriberCount(userID) != 1 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	eventID := uuid.New()
	hub.Publish(userID, []model.DomainEvent{{
		ID: eventID, Type: model.DomainEventXPEarned, OccurredAt: time.Now().UTC(),
		Payload: json.RawMessage(`{"amount":30,"totalXp":30}`),
	}})

	if err := connection.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	var payload struct {
		Events []map[string]any `json:"events"`
	}
	if err := connection.ReadJSON(&payload); err != nil {
		t.Fatalf("read websocket events: %v", err)
	}
	if len(payload.Events) != 1 || payload.Events[0]["type"] != string(model.DomainEventXPEarned) ||
		payload.Events[0]["amount"] != float64(30) {
		t.Fatalf("payload = %+v", payload)
	}

	if err := connection.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "done")); err != nil {
		t.Fatalf("send close: %v", err)
	}
	_ = connection.Close()
	deadline = time.Now().Add(time.Second)
	for hub.SubscriberCount(userID) != 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if count := hub.SubscriberCount(userID); count != 0 {
		t.Fatalf("subscriber count after disconnect = %d", count)
	}
}

func TestWebSocketRejectsWrongOrigin(t *testing.T) {
	hub := realtime.NewHub(1)
	router := newGameTestRouter(RouterDependencies{EventHub: hub, FrontendOrigin: "https://allowed.example"})
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	header := http.Header{}
	header.Set("X-User-ID", uuid.NewString())
	header.Set("Origin", "https://wrong.example")
	connection, response, err := websocket.DefaultDialer.Dial(
		"ws"+strings.TrimPrefix(server.URL, "http")+"/api/v1/ws", header,
	)
	if response != nil {
		defer func() { _ = response.Body.Close() }()
	}
	if connection != nil {
		_ = connection.Close()
	}
	if err == nil || response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("dial error = %v, response = %+v", err, response)
	}
}

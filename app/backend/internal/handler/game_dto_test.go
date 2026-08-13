package handler

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
)

func TestDomainEventDTOFlattensPayloadAndPreservesEnvelope(t *testing.T) {
	eventID := uuid.New()
	event := newDomainEventDTO(model.DomainEvent{
		ID: eventID, Type: model.DomainEventXPEarned,
		OccurredAt: time.Date(2026, 8, 8, 12, 0, 0, 123456789, time.UTC),
		Payload:    json.RawMessage(`{"id":"payload-id","type":"PAYLOAD_TYPE","amount":30}`),
	})

	body, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if result["id"] != eventID.String() || result["type"] != string(model.DomainEventXPEarned) {
		t.Fatalf("envelope fields = %v", result)
	}
	if result["amount"] != float64(30) || result["payload"] != nil {
		t.Fatalf("flattened payload = %v", result)
	}
}

func TestDomainEventDTOFallsBackToEnvelopeForInvalidPayload(t *testing.T) {
	eventID := uuid.New()
	body, err := json.Marshal(newDomainEventDTO(model.DomainEvent{
		ID: eventID, Type: model.DomainEventTaskCompleted,
		OccurredAt: time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC),
		Payload:    json.RawMessage(`not-json`),
	}))
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if result["id"] != eventID.String() || len(result) != 3 {
		t.Fatalf("fallback envelope = %v", result)
	}
}

func TestDomainEventDTOHandlesWhitespaceOnlyObject(t *testing.T) {
	body, err := json.Marshal(newDomainEventDTO(model.DomainEvent{
		ID: uuid.New(), Type: model.DomainEventTaskCompleted,
		OccurredAt: time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC),
		Payload:    json.RawMessage(`{ }`),
	}))
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if !json.Valid(body) {
		t.Fatalf("body is invalid JSON: %s", body)
	}
}

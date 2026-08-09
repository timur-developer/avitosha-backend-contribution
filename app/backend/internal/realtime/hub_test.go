package realtime

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
)

func TestHubDeliversOnlyToTargetUserAndDisconnects(t *testing.T) {
	hub := NewHub(2)
	firstUser := uuid.New()
	secondUser := uuid.New()
	first := hub.Subscribe(firstUser)
	second := hub.Subscribe(secondUser)
	events := []model.DomainEvent{{ID: uuid.New(), Type: model.DomainEventXPEarned}}

	hub.Publish(firstUser, events)

	select {
	case got := <-first.Messages():
		if len(got) != 1 || got[0].ID != events[0].ID {
			t.Fatalf("events = %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("target user did not receive events")
	}
	select {
	case got := <-second.Messages():
		t.Fatalf("other user received events: %+v", got)
	default:
	}

	first.Close()
	if count := hub.SubscriberCount(firstUser); count != 0 {
		t.Fatalf("subscriber count = %d, want 0", count)
	}
	if _, open := <-first.Messages(); open {
		t.Fatal("subscription channel remains open")
	}
	second.Close()
}

func TestHubSlowSubscriberNeverBlocksPublisher(t *testing.T) {
	hub := NewHub(1)
	userID := uuid.New()
	slow := hub.Subscribe(userID)
	events := []model.DomainEvent{{ID: uuid.New(), Type: model.DomainEventTaskProgressUpdated}}
	hub.Publish(userID, events)

	done := make(chan struct{})
	go func() {
		hub.Publish(userID, events)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("slow subscriber blocked publisher")
	}
	if count := hub.SubscriberCount(userID); count != 0 {
		t.Fatalf("slow subscriber count = %d, want 0", count)
	}
	if first, open := <-slow.Messages(); !open || len(first) != 1 {
		t.Fatalf("buffered first message = %+v, open = %v", first, open)
	}
	if _, open := <-slow.Messages(); open {
		t.Fatal("slow subscription was not closed")
	}
}

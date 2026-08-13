package realtime

import (
	"sync"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
)

const DefaultBufferSize = 16

type Hub struct {
	mu         sync.Mutex
	bufferSize int
	byUser     map[uuid.UUID]map[*Subscription]struct{}
}

type EventSubscription interface {
	Messages() <-chan []model.DomainEvent
	Close()
}

type EventSubscriber interface {
	Subscribe(uuid.UUID) EventSubscription
}

type Subscription struct {
	hub      *Hub
	userID   uuid.UUID
	messages chan []model.DomainEvent
	closed   bool
}

func NewHub(bufferSize int) *Hub {
	if bufferSize < 1 {
		bufferSize = DefaultBufferSize
	}
	return &Hub{bufferSize: bufferSize, byUser: make(map[uuid.UUID]map[*Subscription]struct{})}
}

func (hub *Hub) Subscribe(userID uuid.UUID) EventSubscription {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	subscription := &Subscription{
		hub: hub, userID: userID, messages: make(chan []model.DomainEvent, hub.bufferSize),
	}
	if hub.byUser[userID] == nil {
		hub.byUser[userID] = make(map[*Subscription]struct{})
	}
	hub.byUser[userID][subscription] = struct{}{}
	return subscription
}

func (hub *Hub) Publish(userID uuid.UUID, events []model.DomainEvent) {
	if len(events) == 0 {
		return
	}
	batch := append([]model.DomainEvent(nil), events...)

	hub.mu.Lock()
	defer hub.mu.Unlock()
	for subscription := range hub.byUser[userID] {
		select {
		case subscription.messages <- batch:
		default:
			hub.removeLocked(subscription)
		}
	}
}

func (subscription *Subscription) Messages() <-chan []model.DomainEvent {
	return subscription.messages
}

func (subscription *Subscription) Close() {
	if subscription == nil || subscription.hub == nil {
		return
	}
	subscription.hub.mu.Lock()
	defer subscription.hub.mu.Unlock()
	subscription.hub.removeLocked(subscription)
}

func (hub *Hub) SubscriberCount(userID uuid.UUID) int {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	return len(hub.byUser[userID])
}

func (hub *Hub) removeLocked(subscription *Subscription) {
	if subscription.closed {
		return
	}
	subscription.closed = true
	delete(hub.byUser[subscription.userID], subscription)
	if len(hub.byUser[subscription.userID]) == 0 {
		delete(hub.byUser, subscription.userID)
	}
	close(subscription.messages)
}

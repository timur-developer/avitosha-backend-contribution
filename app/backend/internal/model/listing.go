package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type MarketplaceGameRequest struct {
	EventID     uuid.UUID
	UserID      uuid.UUID
	ListingID   uuid.UUID
	Operation   string
	Result      json.RawMessage
	CompletedAt *time.Time
	CreatedAt   time.Time
}

type ListingStatus string

const (
	ListingStatusDraft       ListingStatus = "DRAFT"
	ListingStatusPublished   ListingStatus = "PUBLISHED"
	ListingStatusUnpublished ListingStatus = "UNPUBLISHED"
	ListingStatusSold        ListingStatus = "SOLD"
)

type ListingCategory struct {
	Code      string
	Name      string
	SortOrder int
}

type Listing struct {
	ID           uuid.UUID
	OwnerID      uuid.UUID
	CategoryCode string
	Title        string
	Description  string
	PriceKopecks int64
	Status       ListingStatus
	IsDemo       bool
	PublishedAt  *time.Time
	SoldAt       *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Photos       []ListingPhoto
}

type ListingPhoto struct {
	ID        uuid.UUID
	ListingID uuid.UUID
	URL       string
	SortOrder int
}

type ListingFavorite struct {
	UserID    uuid.UUID
	ListingID uuid.UUID
	CreatedAt time.Time
}

type ListingMessage struct {
	ID          uuid.UUID
	ListingID   uuid.UUID
	SenderID    uuid.UUID
	RecipientID uuid.UUID
	Body        string
	CreatedAt   time.Time
}

type ListingDeal struct {
	ID           uuid.UUID
	ListingID    uuid.UUID
	BuyerID      uuid.UUID
	SellerID     uuid.UUID
	DeliveryUsed bool
	CompletedAt  time.Time
}

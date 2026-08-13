package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
)

const listingDescriptionMinimumLength = 150

type MarketplaceRepository interface {
	ListListingCategories(context.Context) ([]model.ListingCategory, error)
	ListPublicListings(context.Context, *string, string, int, int) ([]model.Listing, int, error)
	GetPublicListing(context.Context, uuid.UUID) (model.Listing, error)
	ListOwnerListings(context.Context, uuid.UUID, int, int) ([]model.Listing, int, error)
	GetListingForUpdate(context.Context, uuid.UUID) (model.Listing, error)
	CreateListing(context.Context, model.Listing) (model.Listing, error)
	UpdateListing(context.Context, model.Listing) (model.Listing, error)
	ReplaceListingPhotos(context.Context, uuid.UUID, []string, time.Time) error
	IsListingCategoryActive(context.Context, string) (bool, error)
	AddListingFavorite(context.Context, uuid.UUID, uuid.UUID, time.Time) (bool, error)
	RemoveListingFavorite(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	ListFavoriteListings(context.Context, uuid.UUID, int, int) ([]model.Listing, int, error)
	RegisterListingView(context.Context, uuid.UUID, uuid.UUID, time.Time) (bool, error)
	CreateFirstListingMessage(context.Context, model.ListingMessage) (model.ListingMessage, bool, error)
	ListListingMessages(context.Context, uuid.UUID, uuid.UUID) ([]model.ListingMessage, error)
	CreateListingDeal(context.Context, model.ListingDeal) (model.ListingDeal, error)
}

type marketplaceGameRepository interface {
	GetMarketplaceGameRequest(context.Context, uuid.UUID) (model.MarketplaceGameRequest, error)
	CreateMarketplaceGameRequest(context.Context, model.MarketplaceGameRequest) (bool, error)
	CompleteMarketplaceGameRequest(context.Context, uuid.UUID, json.RawMessage, time.Time) error
	AwardListingQualityCriteria(context.Context, uuid.UUID, []string, time.Time) ([]string, error)
	ClaimListingFavoriteReward(context.Context, uuid.UUID, uuid.UUID, time.Time) (bool, error)
}

type ListingQuality struct {
	Score          int      `json:"score"`
	IsEligible     bool     `json:"isEligible"`
	MissingFields  []string `json:"missingFields"`
	NextActionHint string   `json:"nextActionHint"`
}

type ListingPage struct {
	Items  []model.Listing
	Total  int
	Limit  int
	Offset int
}

type CreateListingCommand struct {
	OwnerID      uuid.UUID
	CategoryCode string
	Title        string
	Description  string
	PriceKopecks int64
	PhotoURLs    []string
	Now          time.Time
	EventID      uuid.UUID
}

type UpdateListingCommand struct {
	OwnerID      uuid.UUID
	ListingID    uuid.UUID
	CategoryCode string
	Title        string
	Description  string
	PriceKopecks int64
	PhotoURLs    []string
	Now          time.Time
	EventID      uuid.UUID
}

type PurchaseListingCommand struct {
	BuyerID      uuid.UUID
	ListingID    uuid.UUID
	DeliveryUsed bool
	Now          time.Time
	EventID      uuid.UUID
}

type ContactSellerResult struct {
	Message model.ListingMessage `json:"message"`
	First   bool                 `json:"first"`
}

type MarketplaceActionResult struct {
	Listing      *model.Listing        `json:"listing,omitempty"`
	Deal         *model.ListingDeal    `json:"deal,omitempty"`
	Message      *model.ListingMessage `json:"message,omitempty"`
	Favorite     *bool                 `json:"favorite,omitempty"`
	Counted      *bool                 `json:"counted,omitempty"`
	First        *bool                 `json:"first,omitempty"`
	ActionResult *ProcessActionResult  `json:"actionResult,omitempty"`
	// additionalActionResults contains server-side actions produced by one
	// marketplace request. It is intentionally not part of the API response.
	additionalActionResults []publishedActionResult
}

type publishedActionResult struct {
	userID uuid.UUID
	result ProcessActionResult
}

type GameActionProcessor interface {
	ProcessActionWithinTx(context.Context, ProcessActionCommand) (ProcessActionResult, error)
	PublishActionResult(uuid.UUID, ProcessActionResult)
}

type MarketplaceService struct {
	repository  MarketplaceRepository
	txManager   TxManager
	idGenerator IDGenerator
	game        GameActionProcessor
}

func NewMarketplaceService(repository MarketplaceRepository, txManager TxManager, idGenerator IDGenerator, game GameActionProcessor) *MarketplaceService {
	if idGenerator == nil {
		idGenerator = uuid.New
	}
	return &MarketplaceService{repository: repository, txManager: txManager, idGenerator: idGenerator, game: game}
}

func EvaluateListingQuality(listing model.Listing) ListingQuality {
	missing := make([]string, 0, 3)
	score := 0
	if listing.PriceKopecks > 0 {
		score++
	} else {
		missing = append(missing, "price")
	}
	if len(listing.Photos) > 0 {
		score++
	} else {
		missing = append(missing, "photo")
	}
	if len([]rune(strings.TrimSpace(listing.Description))) >= listingDescriptionMinimumLength {
		score++
	} else {
		missing = append(missing, "description")
	}
	hint := "Объявление готово к публикации"
	if len(missing) > 0 {
		hint = listingQualityHint(missing[0])
	}
	// Price is the only publication requirement. Photo and detailed description
	// remain useful quality recommendations, but must not block a listing.
	return ListingQuality{Score: score, IsEligible: listing.PriceKopecks > 0, MissingFields: missing, NextActionHint: hint}
}

func (s *MarketplaceService) ListCategories(ctx context.Context) ([]model.ListingCategory, error) {
	return s.repository.ListListingCategories(ctx)
}

func (s *MarketplaceService) ListPublic(ctx context.Context, category *string, query string, limit, offset int) (ListingPage, error) {
	limit, offset, err := normalizePage(limit, offset)
	if err != nil {
		return ListingPage{}, err
	}
	category = normalizedOptional(category)
	query = strings.TrimSpace(query)
	items, total, err := s.repository.ListPublicListings(ctx, category, query, limit, offset)
	if err != nil {
		return ListingPage{}, fmt.Errorf("list public listings: %w", err)
	}
	return ListingPage{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *MarketplaceService) GetPublic(ctx context.Context, listingID uuid.UUID) (model.Listing, error) {
	return s.repository.GetPublicListing(ctx, listingID)
}

func (s *MarketplaceService) ListMine(ctx context.Context, userID uuid.UUID, limit, offset int) (ListingPage, error) {
	limit, offset, err := normalizePage(limit, offset)
	if err != nil {
		return ListingPage{}, err
	}
	items, total, err := s.repository.ListOwnerListings(ctx, userID, limit, offset)
	if err != nil {
		return ListingPage{}, fmt.Errorf("list owner listings: %w", err)
	}
	return ListingPage{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *MarketplaceService) ListFavorites(ctx context.Context, userID uuid.UUID, limit, offset int) (ListingPage, error) {
	limit, offset, err := normalizePage(limit, offset)
	if err != nil {
		return ListingPage{}, err
	}
	items, total, err := s.repository.ListFavoriteListings(ctx, userID, limit, offset)
	if err != nil {
		return ListingPage{}, fmt.Errorf("list favorites: %w", err)
	}
	return ListingPage{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *MarketplaceService) Create(ctx context.Context, command CreateListingCommand) (model.Listing, error) {
	listing, err := normalizeNewListing(command, s.idGenerator())
	if err != nil {
		return model.Listing{}, err
	}
	return s.inTx(ctx, func(txCtx context.Context) (model.Listing, error) {
		active, err := s.repository.IsListingCategoryActive(txCtx, listing.CategoryCode)
		if err != nil {
			return model.Listing{}, err
		}
		if !active {
			return model.Listing{}, ErrListingCategoryNotFound
		}
		created, err := s.repository.CreateListing(txCtx, listing)
		if err != nil {
			return model.Listing{}, err
		}
		if err = s.repository.ReplaceListingPhotos(txCtx, created.ID, command.PhotoURLs, command.Now.UTC()); err != nil {
			return model.Listing{}, err
		}
		return s.getOwned(txCtx, command.OwnerID, created.ID)
	})
}

func (s *MarketplaceService) Update(ctx context.Context, command UpdateListingCommand) (model.Listing, error) {
	if err := validateListingInput(command.CategoryCode, command.Title, command.Description, command.PriceKopecks, command.PhotoURLs); err != nil {
		return model.Listing{}, err
	}
	return s.inTx(ctx, func(txCtx context.Context) (model.Listing, error) {
		listing, err := s.getOwned(txCtx, command.OwnerID, command.ListingID)
		if err != nil {
			return model.Listing{}, err
		}
		if listing.Status == model.ListingStatusSold {
			return model.Listing{}, ErrListingInvalidTransition
		}
		active, err := s.repository.IsListingCategoryActive(txCtx, strings.ToUpper(strings.TrimSpace(command.CategoryCode)))
		if err != nil {
			return model.Listing{}, err
		}
		if !active {
			return model.Listing{}, ErrListingCategoryNotFound
		}
		listing.CategoryCode = strings.ToUpper(strings.TrimSpace(command.CategoryCode))
		listing.Title = strings.TrimSpace(command.Title)
		listing.Description = strings.TrimSpace(command.Description)
		listing.PriceKopecks = command.PriceKopecks
		listing.UpdatedAt = command.Now.UTC()
		if _, err = s.repository.UpdateListing(txCtx, listing); err != nil {
			return model.Listing{}, err
		}
		if err = s.repository.ReplaceListingPhotos(txCtx, listing.ID, command.PhotoURLs, command.Now.UTC()); err != nil {
			return model.Listing{}, err
		}
		return s.getOwned(txCtx, command.OwnerID, listing.ID)
	})
}

func (s *MarketplaceService) Publish(ctx context.Context, userID, listingID uuid.UUID, now time.Time) (model.Listing, error) {
	return s.changeStatus(ctx, userID, listingID, model.ListingStatusPublished, now)
}
func (s *MarketplaceService) Unpublish(ctx context.Context, userID, listingID uuid.UUID, now time.Time) (model.Listing, error) {
	return s.changeStatus(ctx, userID, listingID, model.ListingStatusUnpublished, now)
}

func (s *MarketplaceService) AddFavorite(ctx context.Context, userID, listingID uuid.UUID, now time.Time) (bool, error) {
	return s.inTxBool(ctx, func(txCtx context.Context) (bool, error) {
		listing, err := s.repository.GetPublicListing(txCtx, listingID)
		if err != nil {
			return false, err
		}
		if listing.OwnerID == userID {
			return false, ErrListingOwnAction
		}
		return s.repository.AddListingFavorite(txCtx, userID, listingID, now.UTC())
	})
}
func (s *MarketplaceService) RemoveFavorite(ctx context.Context, userID, listingID uuid.UUID) (bool, error) {
	return s.repository.RemoveListingFavorite(ctx, userID, listingID)
}

func (s *MarketplaceService) RegisterView(ctx context.Context, userID, listingID uuid.UUID, now time.Time) (bool, error) {
	return s.inTxBool(ctx, func(txCtx context.Context) (bool, error) {
		listing, err := s.repository.GetPublicListing(txCtx, listingID)
		if err != nil {
			return false, err
		}
		if listing.OwnerID == userID {
			return false, ErrListingOwnAction
		}
		return s.repository.RegisterListingView(txCtx, userID, listingID, now.UTC())
	})
}

func (s *MarketplaceService) ContactSeller(ctx context.Context, userID, listingID uuid.UUID, body string, now time.Time) (ContactSellerResult, error) {
	body = strings.TrimSpace(body)
	if body == "" || len([]rune(body)) > 2000 {
		return ContactSellerResult{}, ErrInvalidListingInput
	}
	message, first, err := s.inTxMessage(ctx, func(txCtx context.Context) (model.ListingMessage, bool, error) {
		listing, err := s.repository.GetPublicListing(txCtx, listingID)
		if err != nil {
			return model.ListingMessage{}, false, err
		}
		if listing.OwnerID == userID {
			return model.ListingMessage{}, false, ErrListingOwnAction
		}
		return s.repository.CreateFirstListingMessage(txCtx, model.ListingMessage{ID: s.idGenerator(), ListingID: listingID, SenderID: userID, RecipientID: listing.OwnerID, Body: body, CreatedAt: now.UTC()})
	})
	if err != nil {
		return ContactSellerResult{}, err
	}
	return ContactSellerResult{Message: message, First: first}, nil
}

func (s *MarketplaceService) ListMessages(ctx context.Context, userID, listingID uuid.UUID) ([]model.ListingMessage, error) {
	if _, err := s.repository.GetListingForUpdate(ctx, listingID); err != nil {
		return nil, err
	}
	items, err := s.repository.ListListingMessages(ctx, userID, listingID)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (s *MarketplaceService) Purchase(ctx context.Context, command PurchaseListingCommand) (model.ListingDeal, error) {
	return s.inTxDeal(ctx, func(txCtx context.Context) (model.ListingDeal, error) {
		listing, err := s.repository.GetListingForUpdate(txCtx, command.ListingID)
		if err != nil {
			return model.ListingDeal{}, err
		}
		completedAt := command.Now.UTC()
		listing, shouldUpdate, err := completeListingPurchase(listing, command.BuyerID, completedAt)
		if err != nil {
			return model.ListingDeal{}, err
		}
		if shouldUpdate {
			if _, err = s.repository.UpdateListing(txCtx, listing); err != nil {
				return model.ListingDeal{}, err
			}
		}
		return s.repository.CreateListingDeal(txCtx, model.ListingDeal{ID: s.idGenerator(), ListingID: listing.ID, BuyerID: command.BuyerID, SellerID: listing.OwnerID, DeliveryUsed: command.DeliveryUsed, CompletedAt: completedAt})
	})
}

func (s *MarketplaceService) changeStatus(ctx context.Context, userID, listingID uuid.UUID, target model.ListingStatus, now time.Time) (model.Listing, error) {
	return s.inTx(ctx, func(txCtx context.Context) (model.Listing, error) {
		listing, err := s.getOwned(txCtx, userID, listingID)
		if err != nil {
			return model.Listing{}, err
		}
		quality := EvaluateListingQuality(listing)
		if target == model.ListingStatusPublished {
			if listing.Status != model.ListingStatusDraft && listing.Status != model.ListingStatusUnpublished {
				return model.Listing{}, ErrListingInvalidTransition
			}
			if !quality.IsEligible {
				return model.Listing{}, ErrListingNotEligible
			}
			publishedAt := now.UTC()
			listing.Status = target
			listing.PublishedAt = &publishedAt
		} else {
			if listing.Status != model.ListingStatusPublished {
				return model.Listing{}, ErrListingInvalidTransition
			}
			listing.Status = target
			listing.PublishedAt = nil
		}
		listing.UpdatedAt = now.UTC()
		if _, err = s.repository.UpdateListing(txCtx, listing); err != nil {
			return model.Listing{}, err
		}
		return s.getOwned(txCtx, userID, listing.ID)
	})
}

func (s *MarketplaceService) PublishWithGame(ctx context.Context, userID, listingID, eventID uuid.UUID, now time.Time) (MarketplaceActionResult, error) {
	return s.product(ctx, userID, listingID, eventID, "PUBLISH", now, func(txCtx context.Context) (MarketplaceActionResult, *ProcessActionCommand, error) {
		var result MarketplaceActionResult
		listing, err := s.getOwned(txCtx, userID, listingID)
		if err != nil {
			return MarketplaceActionResult{}, nil, err
		}
		quality := EvaluateListingQuality(listing)
		if listing.Status != model.ListingStatusDraft && listing.Status != model.ListingStatusUnpublished {
			return MarketplaceActionResult{}, nil, ErrListingInvalidTransition
		}
		if !quality.IsEligible {
			return MarketplaceActionResult{}, nil, ErrListingNotEligible
		}
		listing.Status = model.ListingStatusPublished
		listing.UpdatedAt = now.UTC()
		listing.PublishedAt = &listing.UpdatedAt
		if _, err = s.repository.UpdateListing(txCtx, listing); err != nil {
			return MarketplaceActionResult{}, nil, err
		}
		updated, err := s.getOwned(txCtx, userID, listingID)
		if err != nil {
			return MarketplaceActionResult{}, nil, err
		}
		// A fully completed quality profile at publication time also counts as
		// one listing improvement. The unique (listing_id, criterion) records
		// make this safe across re-publish and retries.
		if quality.Score == 3 {
			gameRepository, ok := s.repository.(marketplaceGameRepository)
			if !ok {
				return MarketplaceActionResult{}, nil, ErrUnexpectedStorage
			}
			criteria := []string{"price", "photo", "description"}
			awarded, err := gameRepository.AwardListingQualityCriteria(txCtx, listingID, criteria, now.UTC())
			if err != nil {
				return MarketplaceActionResult{}, nil, err
			}
			if len(awarded) > 0 {
				id, category := listingID.String(), updated.CategoryCode
				meta, _ := json.Marshal(map[string]any{"source": "marketplace.improve", "criteria": awarded})
				improve := ProcessActionCommand{UserID: userID, EventID: uuid.NewSHA1(eventID, []byte("LISTING_IMPROVED")), ActionType: model.ActionTypeListingImproved, EntityID: &id, Category: &category, Metadata: meta, OccurredAt: now, Now: now}
				improveResult, err := s.game.ProcessActionWithinTx(txCtx, improve)
				if err != nil {
					return MarketplaceActionResult{}, nil, err
				}
				// The primary action result is filled by product; retain the
				// secondary result so it is published after commit as well.
				result.additionalActionResults = append(result.additionalActionResults, publishedActionResult{userID: userID, result: improveResult})
			}
		}
		id, category := listingID.String(), updated.CategoryCode
		command := &ProcessActionCommand{UserID: userID, EventID: eventID, ActionType: model.ActionTypeAdCreated, EntityID: &id, Category: &category, Metadata: json.RawMessage(`{"source":"marketplace.publish"}`), OccurredAt: now, Now: now}
		result.Listing = &updated
		return result, command, nil
	})
}

func (s *MarketplaceService) AddFavoriteWithGame(ctx context.Context, userID, listingID, eventID uuid.UUID, now time.Time) (MarketplaceActionResult, error) {
	gameRepository, ok := s.repository.(marketplaceGameRepository)
	if !ok {
		return MarketplaceActionResult{}, ErrUnexpectedStorage
	}
	return s.product(ctx, userID, listingID, eventID, "FAVORITE", now, func(txCtx context.Context) (MarketplaceActionResult, *ProcessActionCommand, error) {
		listing, err := s.repository.GetPublicListing(txCtx, listingID)
		if err != nil {
			return MarketplaceActionResult{}, nil, err
		}
		if listing.OwnerID == userID {
			return MarketplaceActionResult{}, nil, ErrListingOwnAction
		}
		created, err := s.repository.AddListingFavorite(txCtx, userID, listingID, now.UTC())
		if err != nil {
			return MarketplaceActionResult{}, nil, err
		}
		value := true
		result := MarketplaceActionResult{Listing: &listing, Favorite: &value}
		if !created {
			return result, nil, nil
		}
		awarded, err := gameRepository.ClaimListingFavoriteReward(txCtx, userID, listingID, now.UTC())
		if err != nil {
			return MarketplaceActionResult{}, nil, err
		}
		if !awarded {
			return result, nil, nil
		}
		id, category := listingID.String(), listing.CategoryCode
		command := &ProcessActionCommand{UserID: userID, EventID: eventID, ActionType: model.ActionTypeAdFavorited, EntityID: &id, Category: &category, Metadata: json.RawMessage(`{"source":"marketplace.favorite"}`), OccurredAt: now, Now: now}
		return result, command, nil
	})
}

func (s *MarketplaceService) RegisterViewWithGame(ctx context.Context, userID, listingID, eventID uuid.UUID, now time.Time) (MarketplaceActionResult, error) {
	return s.product(ctx, userID, listingID, eventID, "VIEW", now, func(txCtx context.Context) (MarketplaceActionResult, *ProcessActionCommand, error) {
		listing, err := s.repository.GetPublicListing(txCtx, listingID)
		if err != nil {
			return MarketplaceActionResult{}, nil, err
		}
		if listing.OwnerID == userID {
			return MarketplaceActionResult{}, nil, ErrListingOwnAction
		}
		counted, err := s.repository.RegisterListingView(txCtx, userID, listingID, now.UTC())
		if err != nil {
			return MarketplaceActionResult{}, nil, err
		}
		result := MarketplaceActionResult{Listing: &listing, Counted: &counted}
		if !counted {
			return result, nil, nil
		}
		id, category := listingID.String(), listing.CategoryCode
		command := &ProcessActionCommand{UserID: userID, EventID: eventID, ActionType: model.ActionTypeAdViewed, EntityID: &id, Category: &category, Metadata: json.RawMessage(`{"source":"marketplace.view"}`), OccurredAt: now, Now: now}
		return result, command, nil
	})
}

func (s *MarketplaceService) ContactSellerWithGame(ctx context.Context, userID, listingID, eventID uuid.UUID, body string, now time.Time) (MarketplaceActionResult, error) {
	if body = strings.TrimSpace(body); body == "" || len([]rune(body)) > 2000 {
		return MarketplaceActionResult{}, ErrInvalidListingInput
	}
	return s.product(ctx, userID, listingID, eventID, "MESSAGE", now, func(txCtx context.Context) (MarketplaceActionResult, *ProcessActionCommand, error) {
		listing, err := s.repository.GetPublicListing(txCtx, listingID)
		if err != nil {
			return MarketplaceActionResult{}, nil, err
		}
		if listing.OwnerID == userID {
			return MarketplaceActionResult{}, nil, ErrListingOwnAction
		}
		message, first, err := s.repository.CreateFirstListingMessage(txCtx, model.ListingMessage{ID: s.idGenerator(), ListingID: listingID, SenderID: userID, RecipientID: listing.OwnerID, Body: body, CreatedAt: now.UTC()})
		if err != nil {
			return MarketplaceActionResult{}, nil, err
		}
		result := MarketplaceActionResult{Listing: &listing, Message: &message, First: &first}
		if !first {
			return result, nil, nil
		}
		id, category := listingID.String(), listing.CategoryCode
		command := &ProcessActionCommand{UserID: userID, EventID: eventID, ActionType: model.ActionTypeMessageSent, EntityID: &id, Category: &category, Metadata: json.RawMessage(`{"source":"marketplace.message"}`), OccurredAt: now, Now: now}
		return result, command, nil
	})
}

func (s *MarketplaceService) PurchaseWithGame(ctx context.Context, command PurchaseListingCommand) (MarketplaceActionResult, error) {
	return s.product(ctx, command.BuyerID, command.ListingID, command.EventID, "PURCHASE", command.Now, func(txCtx context.Context) (MarketplaceActionResult, *ProcessActionCommand, error) {
		listing, err := s.repository.GetListingForUpdate(txCtx, command.ListingID)
		if err != nil {
			return MarketplaceActionResult{}, nil, err
		}
		completed := command.Now.UTC()
		listing, shouldUpdate, err := completeListingPurchase(listing, command.BuyerID, completed)
		if err != nil {
			return MarketplaceActionResult{}, nil, err
		}
		if shouldUpdate {
			if _, err = s.repository.UpdateListing(txCtx, listing); err != nil {
				return MarketplaceActionResult{}, nil, err
			}
		}
		deal, err := s.repository.CreateListingDeal(txCtx, model.ListingDeal{ID: s.idGenerator(), ListingID: listing.ID, BuyerID: command.BuyerID, SellerID: listing.OwnerID, DeliveryUsed: command.DeliveryUsed, CompletedAt: completed})
		if err != nil {
			return MarketplaceActionResult{}, nil, err
		}
		id, category := listing.ID.String(), listing.CategoryCode
		// Demo listings are fixtures, not real seller activity. The buyer can
		// still receive credit for delivery below.
		var action *ProcessActionCommand
		if !listing.IsDemo {
			action = &ProcessActionCommand{UserID: listing.OwnerID, EventID: command.EventID, ActionType: model.ActionTypeListingSold, EntityID: &id, Category: &category, Metadata: json.RawMessage(`{"source":"marketplace.purchase"}`), OccurredAt: command.Now, Now: command.Now}
		}
		return MarketplaceActionResult{Listing: &listing, Deal: &deal}, action, nil
	})
}

func completeListingPurchase(
	listing model.Listing,
	buyerID uuid.UUID,
	completedAt time.Time,
) (model.Listing, bool, error) {
	if listing.OwnerID == buyerID {
		return model.Listing{}, false, ErrListingOwnAction
	}
	if listing.Status != model.ListingStatusPublished {
		return model.Listing{}, false, ErrListingInvalidTransition
	}
	if listing.IsDemo {
		return listing, false, nil
	}
	listing.Status = model.ListingStatusSold
	listing.SoldAt = &completedAt
	listing.PublishedAt = nil
	listing.UpdatedAt = completedAt
	return listing, true, nil
}

func (s *MarketplaceService) UpdateWithGame(ctx context.Context, command UpdateListingCommand) (MarketplaceActionResult, error) {
	if command.EventID == uuid.Nil {
		return MarketplaceActionResult{}, ErrInvalidAction
	}
	if err := validateListingInput(command.CategoryCode, command.Title, command.Description, command.PriceKopecks, command.PhotoURLs); err != nil {
		return MarketplaceActionResult{}, err
	}
	return s.product(ctx, command.OwnerID, command.ListingID, command.EventID, "IMPROVE", command.Now, func(txCtx context.Context) (MarketplaceActionResult, *ProcessActionCommand, error) {
		listing, err := s.getOwned(txCtx, command.OwnerID, command.ListingID)
		if err != nil {
			return MarketplaceActionResult{}, nil, err
		}
		old := EvaluateListingQuality(listing)
		if listing.Status == model.ListingStatusSold {
			return MarketplaceActionResult{}, nil, ErrListingInvalidTransition
		}
		active, err := s.repository.IsListingCategoryActive(txCtx, strings.ToUpper(strings.TrimSpace(command.CategoryCode)))
		if err != nil {
			return MarketplaceActionResult{}, nil, err
		}
		if !active {
			return MarketplaceActionResult{}, nil, ErrListingCategoryNotFound
		}
		listing.CategoryCode = strings.ToUpper(strings.TrimSpace(command.CategoryCode))
		listing.Title = strings.TrimSpace(command.Title)
		listing.Description = strings.TrimSpace(command.Description)
		listing.PriceKopecks = command.PriceKopecks
		listing.UpdatedAt = command.Now.UTC()
		if _, err = s.repository.UpdateListing(txCtx, listing); err != nil {
			return MarketplaceActionResult{}, nil, err
		}
		if err = s.repository.ReplaceListingPhotos(txCtx, listing.ID, command.PhotoURLs, command.Now.UTC()); err != nil {
			return MarketplaceActionResult{}, nil, err
		}
		updated, err := s.getOwned(txCtx, command.OwnerID, listing.ID)
		if err != nil {
			return MarketplaceActionResult{}, nil, err
		}
		result := MarketplaceActionResult{Listing: &updated}
		if updated.Status != model.ListingStatusPublished {
			return result, nil, nil
		}
		fresh := newQualityCriteria(old, EvaluateListingQuality(updated))
		gameRepository, ok := s.repository.(marketplaceGameRepository)
		if !ok {
			return MarketplaceActionResult{}, nil, ErrUnexpectedStorage
		}
		awarded, err := gameRepository.AwardListingQualityCriteria(txCtx, updated.ID, fresh, command.Now.UTC())
		if err != nil {
			return MarketplaceActionResult{}, nil, err
		}
		if len(awarded) == 0 {
			return result, nil, nil
		}
		id, category := updated.ID.String(), updated.CategoryCode
		meta, _ := json.Marshal(map[string]any{"source": "marketplace.improve", "criteria": awarded})
		action := &ProcessActionCommand{UserID: command.OwnerID, EventID: command.EventID, ActionType: model.ActionTypeListingImproved, EntityID: &id, Category: &category, Metadata: meta, OccurredAt: command.Now, Now: command.Now}
		return result, action, nil
	})
}

func (s *MarketplaceService) product(ctx context.Context, userID, listingID, eventID uuid.UUID, operation string, now time.Time, mutate func(context.Context) (MarketplaceActionResult, *ProcessActionCommand, error)) (MarketplaceActionResult, error) {
	if eventID == uuid.Nil {
		return MarketplaceActionResult{}, ErrInvalidAction
	}
	gameRepository, ok := s.repository.(marketplaceGameRepository)
	if !ok {
		return MarketplaceActionResult{}, ErrUnexpectedStorage
	}
	var result MarketplaceActionResult
	type publishedAction struct {
		userID uuid.UUID
		result ProcessActionResult
	}
	var publish []publishedAction
	err := s.txManager.WithinTx(ctx, func(txCtx context.Context) error {
		existing, err := gameRepository.GetMarketplaceGameRequest(txCtx, eventID)
		if err == nil {
			if existing.UserID != userID || existing.ListingID != listingID || existing.Operation != operation {
				return ErrEventIDConflict
			}
			if json.Unmarshal(existing.Result, &result) != nil {
				return ErrUnexpectedStorage
			}
			if result.ActionResult != nil {
				result.ActionResult.Duplicate = true
			}
			return nil
		}
		if !errors.Is(err, ErrActionNotFound) {
			return err
		}
		created, err := gameRepository.CreateMarketplaceGameRequest(txCtx, model.MarketplaceGameRequest{EventID: eventID, UserID: userID, ListingID: listingID, Operation: operation, CreatedAt: now.UTC()})
		if err != nil {
			return err
		}
		if !created {
			existing, err = gameRepository.GetMarketplaceGameRequest(txCtx, eventID)
			if err != nil {
				return err
			}
			if existing.UserID != userID || existing.ListingID != listingID || existing.Operation != operation {
				return ErrEventIDConflict
			}
			if json.Unmarshal(existing.Result, &result) != nil {
				return ErrUnexpectedStorage
			}
			if result.ActionResult != nil {
				result.ActionResult.Duplicate = true
			}
			return nil
		}
		var action *ProcessActionCommand
		result, action, err = mutate(txCtx)
		if err != nil {
			return err
		}
		if action != nil {
			gameResult, err := s.game.ProcessActionWithinTx(txCtx, *action)
			if err != nil {
				return err
			}
			if action.UserID == userID {
				result.ActionResult = &gameResult
			}
			publish = append(publish, publishedAction{userID: action.UserID, result: gameResult})
			for _, extra := range result.additionalActionResults {
				publish = append(publish, publishedAction{userID: extra.userID, result: extra.result})
			}
			if operation == "PURCHASE" && result.Deal != nil && result.Deal.DeliveryUsed {
				derived := uuid.NewSHA1(eventID, []byte("DELIVERY_USED"))
				delivery := *action
				delivery.UserID = userID
				delivery.EventID = derived
				delivery.ActionType = model.ActionTypeDeliveryUsed
				delivery.Metadata = json.RawMessage(`{"source":"marketplace.purchase.delivery"}`)
				deliveryResult, err := s.game.ProcessActionWithinTx(txCtx, delivery)
				if err != nil {
					return err
				}
				if result.ActionResult == nil {
					result.ActionResult = &deliveryResult
				} else {
					result.ActionResult.Events = append(result.ActionResult.Events, deliveryResult.Events...)
				}
				publish = append(publish, publishedAction{userID: delivery.UserID, result: deliveryResult})
			}
		} else if operation == "PURCHASE" && result.Deal != nil && result.Deal.DeliveryUsed {
			derived := uuid.NewSHA1(eventID, []byte("DELIVERY_USED"))
			listingIDString := listingID.String()
			delivery := ProcessActionCommand{UserID: userID, EventID: derived, ActionType: model.ActionTypeDeliveryUsed, EntityID: &listingIDString, Metadata: json.RawMessage(`{"source":"marketplace.purchase.delivery"}`), OccurredAt: now, Now: now}
			deliveryResult, err := s.game.ProcessActionWithinTx(txCtx, delivery)
			if err != nil {
				return err
			}
			result.ActionResult = &deliveryResult
			publish = append(publish, publishedAction{userID: userID, result: deliveryResult})
		}
		raw, err := json.Marshal(result)
		if err != nil {
			return err
		}
		return gameRepository.CompleteMarketplaceGameRequest(txCtx, eventID, raw, now.UTC())
	})
	if err != nil {
		return MarketplaceActionResult{}, err
	}
	for _, item := range publish {
		s.game.PublishActionResult(item.userID, item.result)
	}
	return result, nil
}

func newQualityCriteria(old, current ListingQuality) []string {
	oldSet := map[string]bool{}
	for _, item := range old.MissingFields {
		oldSet[item] = true
	}
	currentMissing := map[string]bool{}
	for _, item := range current.MissingFields {
		currentMissing[item] = true
	}
	result := make([]string, 0, 3)
	for _, item := range []string{"price", "photo", "description"} {
		if oldSet[item] && !currentMissing[item] {
			result = append(result, item)
		}
	}
	return result
}

func (s *MarketplaceService) getOwned(ctx context.Context, userID, listingID uuid.UUID) (model.Listing, error) {
	listing, err := s.repository.GetListingForUpdate(ctx, listingID)
	if err != nil {
		return model.Listing{}, err
	}
	if listing.OwnerID != userID {
		return model.Listing{}, ErrListingForbidden
	}
	return listing, nil
}
func (s *MarketplaceService) inTx(ctx context.Context, fn func(context.Context) (model.Listing, error)) (model.Listing, error) {
	var result model.Listing
	err := s.txManager.WithinTx(ctx, func(txCtx context.Context) error { var err error; result, err = fn(txCtx); return err })
	return result, err
}
func (s *MarketplaceService) inTxBool(ctx context.Context, fn func(context.Context) (bool, error)) (bool, error) {
	var result bool
	err := s.txManager.WithinTx(ctx, func(txCtx context.Context) error { var err error; result, err = fn(txCtx); return err })
	return result, err
}
func (s *MarketplaceService) inTxMessage(ctx context.Context, fn func(context.Context) (model.ListingMessage, bool, error)) (model.ListingMessage, bool, error) {
	var result model.ListingMessage
	var first bool
	err := s.txManager.WithinTx(ctx, func(txCtx context.Context) error { var err error; result, first, err = fn(txCtx); return err })
	return result, first, err
}
func (s *MarketplaceService) inTxDeal(ctx context.Context, fn func(context.Context) (model.ListingDeal, error)) (model.ListingDeal, error) {
	var result model.ListingDeal
	err := s.txManager.WithinTx(ctx, func(txCtx context.Context) error { var err error; result, err = fn(txCtx); return err })
	return result, err
}

func normalizeNewListing(command CreateListingCommand, id uuid.UUID) (model.Listing, error) {
	if err := validateListingInput(command.CategoryCode, command.Title, command.Description, command.PriceKopecks, command.PhotoURLs); err != nil {
		return model.Listing{}, err
	}
	now := command.Now.UTC()
	return model.Listing{ID: id, OwnerID: command.OwnerID, CategoryCode: strings.ToUpper(strings.TrimSpace(command.CategoryCode)), Title: strings.TrimSpace(command.Title), Description: strings.TrimSpace(command.Description), PriceKopecks: command.PriceKopecks, Status: model.ListingStatusDraft, CreatedAt: now, UpdatedAt: now}, nil
}
func validateListingInput(category, title, description string, price int64, photos []string) error {
	if strings.TrimSpace(category) == "" || strings.TrimSpace(title) == "" || len([]rune(strings.TrimSpace(title))) > 120 || price < 0 || len([]rune(strings.TrimSpace(description))) > 5000 || len(photos) > 10 {
		return ErrInvalidListingInput
	}
	for _, photo := range photos {
		value := strings.TrimSpace(photo)
		parsed, err := url.ParseRequestURI(value)
		isAbsoluteHTTP := err == nil && parsed.Host != "" && (parsed.Scheme == "http" || parsed.Scheme == "https")
		isLocalStorage := err == nil && parsed.Scheme == "" && parsed.Host == "" && strings.HasPrefix(parsed.Path, "/storage/avitosha-photos/")
		if !isAbsoluteHTTP && !isLocalStorage {
			return ErrInvalidListingInput
		}
	}
	return nil
}
func normalizePage(limit, offset int) (int, int, error) {
	if limit == 0 {
		limit = 20
	}
	if limit < 1 || limit > 100 || offset < 0 {
		return 0, 0, ErrInvalidListingInput
	}
	return limit, offset, nil
}
func normalizedOptional(value *string) *string {
	if value == nil {
		return nil
	}
	normalized := strings.ToUpper(strings.TrimSpace(*value))
	if normalized == "" {
		return nil
	}
	return &normalized
}
func listingQualityHint(field string) string {
	switch field {
	case "price":
		return "Укажите цену объявления"
	case "photo":
		return "Рекомендуем добавить хотя бы одну фотографию"
	default:
		return "Рекомендуем добавить подробное описание"
	}
}

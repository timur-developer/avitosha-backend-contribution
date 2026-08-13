package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

type MarketplaceUseCase interface {
	ListCategories(context.Context) ([]model.ListingCategory, error)
	ListPublic(context.Context, *string, string, int, int) (usecase.ListingPage, error)
	GetPublic(context.Context, uuid.UUID) (model.Listing, error)
	ListMine(context.Context, uuid.UUID, int, int) (usecase.ListingPage, error)
	ListFavorites(context.Context, uuid.UUID, int, int) (usecase.ListingPage, error)
	Create(context.Context, usecase.CreateListingCommand) (model.Listing, error)
	UpdateWithGame(context.Context, usecase.UpdateListingCommand) (usecase.MarketplaceActionResult, error)
	PublishWithGame(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, time.Time) (usecase.MarketplaceActionResult, error)
	Unpublish(context.Context, uuid.UUID, uuid.UUID, time.Time) (model.Listing, error)
	AddFavoriteWithGame(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, time.Time) (usecase.MarketplaceActionResult, error)
	RemoveFavorite(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	RegisterViewWithGame(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, time.Time) (usecase.MarketplaceActionResult, error)
	ContactSellerWithGame(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, time.Time) (usecase.MarketplaceActionResult, error)
	ListMessages(context.Context, uuid.UUID, uuid.UUID) ([]model.ListingMessage, error)
	PurchaseWithGame(context.Context, usecase.PurchaseListingCommand) (usecase.MarketplaceActionResult, error)
}

type MarketplaceHandler struct {
	logger  *slog.Logger
	service MarketplaceUseCase
	now     func() time.Time
}

func NewMarketplaceHandler(logger *slog.Logger, service MarketplaceUseCase, now func() time.Time) MarketplaceHandler {
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = time.Now
	}
	return MarketplaceHandler{logger: logger, service: service, now: now}
}

func (h MarketplaceHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListCategories(r.Context())
	if err != nil {
		h.writeError(w, r, "list_categories", err)
		return
	}
	result := make([]listingCategoryDTO, len(items))
	for index, item := range items {
		result[index] = listingCategoryDTO{Code: item.Code, Name: item.Name, SortOrder: item.SortOrder}
	}
	writeJSON(w, http.StatusOK, map[string]any{"categories": result})
}
func (h MarketplaceHandler) ListPublic(w http.ResponseWriter, r *http.Request) {
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	var categoryPtr *string
	if category != "" {
		categoryPtr = &category
	}
	limit, offset, err := pagination(r)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, invalidRequestCode, err.Error())
		return
	}
	page, err := h.service.ListPublic(r.Context(), categoryPtr, r.URL.Query().Get("query"), limit, offset)
	if err != nil {
		h.writeError(w, r, "list_public_listings", err)
		return
	}
	writeJSON(w, http.StatusOK, newListingPageDTO(page))
}
func (h MarketplaceHandler) GetPublic(w http.ResponseWriter, r *http.Request) {
	id, ok := listingID(w, r)
	if !ok {
		return
	}
	item, err := h.service.GetPublic(r.Context(), id)
	if err != nil {
		h.writeError(w, r, "get_public_listing", err)
		return
	}
	writeJSON(w, http.StatusOK, newListingDTO(item))
}
func (h MarketplaceHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	limit, offset, err := pagination(r)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, invalidRequestCode, err.Error())
		return
	}
	page, err := h.service.ListMine(r.Context(), user, limit, offset)
	if err != nil {
		h.writeError(w, r, "list_my_listings", err)
		return
	}
	writeJSON(w, http.StatusOK, newListingPageDTO(page))
}
func (h MarketplaceHandler) ListFavorites(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	limit, offset, err := pagination(r)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, invalidRequestCode, err.Error())
		return
	}
	page, err := h.service.ListFavorites(r.Context(), user, limit, offset)
	if err != nil {
		h.writeError(w, r, "list_favorites", err)
		return
	}
	writeJSON(w, http.StatusOK, newListingPageDTO(page))
}
func (h MarketplaceHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	request, ok := decodeListingRequest(w, r)
	if !ok {
		return
	}
	item, err := h.service.Create(r.Context(), usecase.CreateListingCommand{OwnerID: user, CategoryCode: request.CategoryCode, Title: request.Title, Description: request.Description, PriceKopecks: request.PriceKopecks, PhotoURLs: request.PhotoURLs, Now: h.now()})
	if err != nil {
		h.writeError(w, r, "create_listing", err)
		return
	}
	writeJSON(w, http.StatusCreated, newListingDTO(item))
}
func (h MarketplaceHandler) Update(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	id, ok := listingID(w, r)
	if !ok {
		return
	}
	request, ok := decodeListingRequest(w, r)
	if !ok {
		return
	}
	eventID, ok := requestEventID(w, request.EventID)
	if !ok {
		return
	}
	result, err := h.service.UpdateWithGame(r.Context(), usecase.UpdateListingCommand{OwnerID: user, ListingID: id, CategoryCode: request.CategoryCode, Title: request.Title, Description: request.Description, PriceKopecks: request.PriceKopecks, PhotoURLs: request.PhotoURLs, EventID: eventID, Now: h.now()})
	if err != nil {
		h.writeError(w, r, "update_listing", err)
		return
	}
	writeJSON(w, http.StatusOK, newMarketplaceActionDTO(result))
}
func (h MarketplaceHandler) Publish(w http.ResponseWriter, r *http.Request)   { h.status(w, r, true) }
func (h MarketplaceHandler) Unpublish(w http.ResponseWriter, r *http.Request) { h.status(w, r, false) }
func (h MarketplaceHandler) status(w http.ResponseWriter, r *http.Request, publish bool) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	id, ok := listingID(w, r)
	if !ok {
		return
	}
	if publish {
		var result usecase.MarketplaceActionResult
		var request struct {
			EventID string `json:"eventId"`
		}
		if !decodeJSON(w, r, &request) {
			return
		}
		eventID, valid := requestEventID(w, request.EventID)
		if !valid {
			return
		}
		result, err := h.service.PublishWithGame(r.Context(), user, id, eventID, h.now())
		if err != nil {
			h.writeError(w, r, "change_listing_status", err)
			return
		}
		writeJSON(w, http.StatusOK, newMarketplaceActionDTO(result))
		return
	}
	item, err := h.service.Unpublish(r.Context(), user, id, h.now())
	if err != nil {
		h.writeError(w, r, "change_listing_status", err)
		return
	}
	writeJSON(w, http.StatusOK, newListingDTO(item))
}
func (h MarketplaceHandler) AddFavorite(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	id, ok := listingID(w, r)
	if !ok {
		return
	}
	var request struct {
		EventID string `json:"eventId"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	eventID, ok := requestEventID(w, request.EventID)
	if !ok {
		return
	}
	result, err := h.service.AddFavoriteWithGame(r.Context(), user, id, eventID, h.now())
	if err != nil {
		h.writeError(w, r, "add_favorite", err)
		return
	}
	writeJSON(w, http.StatusOK, newMarketplaceActionDTO(result))
}
func (h MarketplaceHandler) RemoveFavorite(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	id, ok := listingID(w, r)
	if !ok {
		return
	}
	removed, err := h.service.RemoveFavorite(r.Context(), user, id)
	if err != nil {
		h.writeError(w, r, "remove_favorite", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"favorite": false, "removed": removed})
}
func (h MarketplaceHandler) RegisterView(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	id, ok := listingID(w, r)
	if !ok {
		return
	}
	var request struct {
		EventID string `json:"eventId"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	eventID, ok := requestEventID(w, request.EventID)
	if !ok {
		return
	}
	result, err := h.service.RegisterViewWithGame(r.Context(), user, id, eventID, h.now())
	if err != nil {
		h.writeError(w, r, "register_view", err)
		return
	}
	writeJSON(w, http.StatusOK, newMarketplaceActionDTO(result))
}
func (h MarketplaceHandler) ContactSeller(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	id, ok := listingID(w, r)
	if !ok {
		return
	}
	var request struct {
		Body    string `json:"body"`
		EventID string `json:"eventId"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	eventID, ok := requestEventID(w, request.EventID)
	if !ok {
		return
	}
	result, err := h.service.ContactSellerWithGame(r.Context(), user, id, eventID, request.Body, h.now())
	if err != nil {
		h.writeError(w, r, "contact_seller", err)
		return
	}
	writeJSON(w, http.StatusCreated, newMarketplaceActionDTO(result))
}
func (h MarketplaceHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	id, ok := listingID(w, r)
	if !ok {
		return
	}
	items, err := h.service.ListMessages(r.Context(), user, id)
	if err != nil {
		h.writeError(w, r, "list_messages", err)
		return
	}
	result := make([]listingMessageDTO, len(items))
	for i, item := range items {
		result[i] = newListingMessageDTO(item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": result})
}
func (h MarketplaceHandler) Purchase(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	id, ok := listingID(w, r)
	if !ok {
		return
	}
	var request struct {
		DeliveryUsed bool   `json:"deliveryUsed"`
		EventID      string `json:"eventId"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	eventID, ok := requestEventID(w, request.EventID)
	if !ok {
		return
	}
	result, err := h.service.PurchaseWithGame(r.Context(), usecase.PurchaseListingCommand{BuyerID: user, ListingID: id, DeliveryUsed: request.DeliveryUsed, EventID: eventID, Now: h.now()})
	if err != nil {
		h.writeError(w, r, "purchase_listing", err)
		return
	}
	writeJSON(w, http.StatusCreated, newMarketplaceActionDTO(result))
}

func (h MarketplaceHandler) requireUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	user, ok := gameUserID(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, unauthorizedCode, "Authentication is required")
	}
	return user, ok
}
func (h MarketplaceHandler) writeError(w http.ResponseWriter, r *http.Request, operation string, err error) {
	status, code, message := mapMarketplaceError(err)
	if status >= 500 {
		h.logger.Error("marketplace request failed", "operation", operation, "error", err.Error())
	}
	writeErrorResponse(w, status, code, message)
}
func mapMarketplaceError(err error) (int, string, string) {
	switch {
	case errors.Is(err, usecase.ErrInvalidListingInput):
		return 400, invalidRequestCode, "Listing request is invalid"
	case errors.Is(err, usecase.ErrListingNotFound):
		return 404, "listing_not_found", "Listing not found"
	case errors.Is(err, usecase.ErrListingCategoryNotFound):
		return 400, "listing_category_not_found", "Listing category not found"
	case errors.Is(err, usecase.ErrListingForbidden):
		return 403, "listing_forbidden", "You do not own this listing"
	case errors.Is(err, usecase.ErrListingOwnAction):
		return 409, "own_listing_action", "You cannot interact with your own listing"
	case errors.Is(err, usecase.ErrListingNotEligible):
		return 409, "listing_not_eligible", "Listing needs a price greater than zero"
	case errors.Is(err, usecase.ErrListingInvalidTransition):
		return 409, "listing_invalid_transition", "Listing status transition is not allowed"
	case errors.Is(err, usecase.ErrDemoPurchaseCompleted):
		return 409, "demo_purchase_already_completed", "This demo purchase has already been completed"
	default:
		return 500, internalErrorCode, "Internal server error"
	}
}
func listingID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "listing_id"))
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, invalidRequestCode, "listing_id must be a UUID")
		return uuid.Nil, false
	}
	return id, true
}
func pagination(r *http.Request) (int, int, error) {
	limit := 20
	offset := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return 0, 0, errors.New("limit must be an integer")
		}
		limit = parsed
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return 0, 0, errors.New("offset must be an integer")
		}
		offset = parsed
	}
	return limit, offset, nil
}
func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeErrorResponse(w, 400, invalidRequestCode, "Request body must be valid JSON")
		return false
	}
	return true
}

type listingRequest struct {
	CategoryCode string   `json:"categoryCode"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	PriceKopecks int64    `json:"priceKopecks"`
	PhotoURLs    []string `json:"photoUrls"`
	EventID      string   `json:"eventId"`
}

type marketplaceActionDTO struct {
	Listing      *listingDTO                  `json:"listing,omitempty"`
	Deal         *listingDealDTO              `json:"deal,omitempty"`
	Message      *listingMessageDTO           `json:"message,omitempty"`
	Favorite     *bool                        `json:"favorite,omitempty"`
	Counted      *bool                        `json:"counted,omitempty"`
	First        *bool                        `json:"first,omitempty"`
	ActionResult *usecase.ProcessActionResult `json:"actionResult,omitempty"`
}

func newMarketplaceActionDTO(value usecase.MarketplaceActionResult) marketplaceActionDTO {
	result := marketplaceActionDTO{Favorite: value.Favorite, Counted: value.Counted, First: value.First, ActionResult: value.ActionResult}
	if value.Listing != nil {
		item := newListingDTO(*value.Listing)
		result.Listing = &item
	}
	if value.Deal != nil {
		item := listingDealDTO{ID: value.Deal.ID, ListingID: value.Deal.ListingID, BuyerID: value.Deal.BuyerID, SellerID: value.Deal.SellerID, DeliveryUsed: value.Deal.DeliveryUsed, CompletedAt: value.Deal.CompletedAt}
		result.Deal = &item
	}
	if value.Message != nil {
		item := newListingMessageDTO(*value.Message)
		result.Message = &item
	}
	return result
}
func requestEventID(w http.ResponseWriter, value string) (uuid.UUID, bool) {
	id, err := uuid.Parse(value)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, invalidRequestCode, "eventId must be a UUID")
		return uuid.Nil, false
	}
	return id, true
}

type listingCategoryDTO struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	SortOrder int    `json:"sortOrder"`
}

func decodeListingRequest(w http.ResponseWriter, r *http.Request) (listingRequest, bool) {
	var request listingRequest
	return request, decodeJSON(w, r, &request)
}

type listingDTO struct {
	ID           uuid.UUID              `json:"id"`
	OwnerID      uuid.UUID              `json:"ownerId"`
	CategoryCode string                 `json:"categoryCode"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	PriceKopecks int64                  `json:"priceKopecks"`
	Status       model.ListingStatus    `json:"status"`
	IsDemo       bool                   `json:"isDemo"`
	PublishedAt  *time.Time             `json:"publishedAt"`
	SoldAt       *time.Time             `json:"soldAt"`
	Photos       []string               `json:"photoUrls"`
	Quality      usecase.ListingQuality `json:"quality"`
}

func newListingDTO(item model.Listing) listingDTO {
	photos := make([]string, len(item.Photos))
	for i, photo := range item.Photos {
		photos[i] = photo.URL
	}
	return listingDTO{ID: item.ID, OwnerID: item.OwnerID, CategoryCode: item.CategoryCode, Title: item.Title, Description: item.Description, PriceKopecks: item.PriceKopecks, Status: item.Status, IsDemo: item.IsDemo, PublishedAt: item.PublishedAt, SoldAt: item.SoldAt, Photos: photos, Quality: usecase.EvaluateListingQuality(item)}
}

type listingPageDTO struct {
	Items  []listingDTO `json:"items"`
	Total  int          `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}

func newListingPageDTO(page usecase.ListingPage) listingPageDTO {
	items := make([]listingDTO, len(page.Items))
	for i, item := range page.Items {
		items[i] = newListingDTO(item)
	}
	return listingPageDTO{Items: items, Total: page.Total, Limit: page.Limit, Offset: page.Offset}
}

type listingMessageDTO struct {
	ID          uuid.UUID `json:"id"`
	ListingID   uuid.UUID `json:"listingId"`
	SenderID    uuid.UUID `json:"senderId"`
	RecipientID uuid.UUID `json:"recipientId"`
	Body        string    `json:"body"`
	CreatedAt   time.Time `json:"createdAt"`
}

type listingDealDTO struct {
	ID           uuid.UUID `json:"id"`
	ListingID    uuid.UUID `json:"listingId"`
	BuyerID      uuid.UUID `json:"buyerId"`
	SellerID     uuid.UUID `json:"sellerId"`
	DeliveryUsed bool      `json:"deliveryUsed"`
	CompletedAt  time.Time `json:"completedAt"`
}

func newListingMessageDTO(item model.ListingMessage) listingMessageDTO {
	return listingMessageDTO{ID: item.ID, ListingID: item.ListingID, SenderID: item.SenderID, RecipientID: item.RecipientID, Body: item.Body, CreatedAt: item.CreatedAt}
}

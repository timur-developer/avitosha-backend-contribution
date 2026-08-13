package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

const listingSelect = `
SELECT l.id, l.owner_id, l.category_code, l.title, l.description, l.price_kopecks, l.status,
       l.is_demo, l.published_at, l.sold_at, l.created_at, l.updated_at
FROM listings l
`

func (r *GameRepository) ListListingCategories(ctx context.Context) ([]model.ListingCategory, error) {
	rows, err := executorFromContext(ctx, r.executor).Query(ctx, `
SELECT code, name, sort_order FROM listing_categories WHERE is_active ORDER BY sort_order, code`)
	if err != nil {
		return nil, mapGameStorageError("list listing categories", err)
	}
	defer rows.Close()
	items := make([]model.ListingCategory, 0)
	for rows.Next() {
		var item model.ListingCategory
		if err = rows.Scan(&item.Code, &item.Name, &item.SortOrder); err != nil {
			return nil, mapGameStorageError("scan listing category", err)
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return nil, mapGameStorageError("iterate listing categories", err)
	}
	return items, nil
}

func (r *GameRepository) ListPublicListings(ctx context.Context, category *string, query string, limit, offset int) ([]model.Listing, int, error) {
	executor := executorFromContext(ctx, r.executor)
	var total int
	err := executor.QueryRow(ctx, `
SELECT COUNT(*) FROM listings
WHERE status = 'PUBLISHED' AND ($1::TEXT IS NULL OR category_code = $1)
  AND ($2 = '' OR title ILIKE '%' || $2 || '%' OR description ILIKE '%' || $2 || '%')`, category, query).Scan(&total)
	if err != nil {
		return nil, 0, mapGameStorageError("count public listings", err)
	}
	rows, err := executor.Query(ctx, listingSelect+`
WHERE l.status = 'PUBLISHED' AND ($1::TEXT IS NULL OR l.category_code = $1)
  AND ($2 = '' OR l.title ILIKE '%' || $2 || '%' OR l.description ILIKE '%' || $2 || '%')
ORDER BY l.published_at DESC, l.id DESC LIMIT $3 OFFSET $4`, category, query, limit, offset)
	if err != nil {
		return nil, 0, mapGameStorageError("list public listings", err)
	}
	items, err := r.collectListings(ctx, rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *GameRepository) GetPublicListing(ctx context.Context, listingID uuid.UUID) (model.Listing, error) {
	listing, err := r.getListing(ctx, listingSelect+"WHERE l.id = $1 AND l.status = 'PUBLISHED'", listingID)
	if errors.Is(err, usecase.ErrListingNotFound) {
		return model.Listing{}, err
	}
	return listing, err
}

func (r *GameRepository) ListOwnerListings(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]model.Listing, int, error) {
	executor := executorFromContext(ctx, r.executor)
	var total int
	if err := executor.QueryRow(ctx, `SELECT COUNT(*) FROM listings WHERE owner_id = $1`, ownerID).Scan(&total); err != nil {
		return nil, 0, mapGameStorageError("count owner listings", err)
	}
	rows, err := executor.Query(ctx, listingSelect+`WHERE l.owner_id = $1 ORDER BY l.updated_at DESC, l.id DESC LIMIT $2 OFFSET $3`, ownerID, limit, offset)
	if err != nil {
		return nil, 0, mapGameStorageError("list owner listings", err)
	}
	items, err := r.collectListings(ctx, rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *GameRepository) ListFavoriteListings(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.Listing, int, error) {
	executor := executorFromContext(ctx, r.executor)
	var total int
	if err := executor.QueryRow(ctx, `SELECT COUNT(*) FROM listing_favorites WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, mapGameStorageError("count favorite listings", err)
	}
	rows, err := executor.Query(ctx, listingSelect+`
JOIN listing_favorites f ON f.listing_id = l.id
WHERE f.user_id = $1
ORDER BY f.created_at DESC, l.id DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, 0, mapGameStorageError("list favorite listings", err)
	}
	items, err := r.collectListings(ctx, rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *GameRepository) GetListingForUpdate(ctx context.Context, listingID uuid.UUID) (model.Listing, error) {
	return r.getListing(ctx, listingSelect+"WHERE l.id = $1 FOR UPDATE", listingID)
}

func (r *GameRepository) CreateListing(ctx context.Context, listing model.Listing) (model.Listing, error) {
	_, err := executorFromContext(ctx, r.executor).Exec(ctx, `
INSERT INTO listings (id, owner_id, category_code, title, description, price_kopecks, status, is_demo, published_at, sold_at, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, listing.ID, listing.OwnerID, listing.CategoryCode, listing.Title, listing.Description, listing.PriceKopecks, listing.Status, listing.IsDemo, listing.PublishedAt, listing.SoldAt, listing.CreatedAt, listing.UpdatedAt)
	if err != nil {
		return model.Listing{}, mapGameStorageError("create listing", err)
	}
	return r.GetListingForUpdate(ctx, listing.ID)
}

func (r *GameRepository) UpdateListing(ctx context.Context, listing model.Listing) (model.Listing, error) {
	tag, err := executorFromContext(ctx, r.executor).Exec(ctx, `
UPDATE listings SET category_code=$2,title=$3,description=$4,price_kopecks=$5,status=$6,published_at=$7,sold_at=$8,updated_at=$9 WHERE id=$1`, listing.ID, listing.CategoryCode, listing.Title, listing.Description, listing.PriceKopecks, listing.Status, listing.PublishedAt, listing.SoldAt, listing.UpdatedAt)
	if err != nil {
		return model.Listing{}, mapGameStorageError("update listing", err)
	}
	if tag.RowsAffected() != 1 {
		return model.Listing{}, usecase.ErrListingNotFound
	}
	return r.GetListingForUpdate(ctx, listing.ID)
}

func (r *GameRepository) ReplaceListingPhotos(ctx context.Context, listingID uuid.UUID, urls []string, now time.Time) error {
	executor := executorFromContext(ctx, r.executor)
	if _, err := executor.Exec(ctx, `DELETE FROM listing_photos WHERE listing_id = $1`, listingID); err != nil {
		return mapGameStorageError("delete listing photos", err)
	}
	for index, rawURL := range urls {
		if _, err := executor.Exec(ctx, `INSERT INTO listing_photos (id, listing_id, url, sort_order, created_at) VALUES ($1,$2,$3,$4,$5)`, uuid.New(), listingID, rawURL, index, now); err != nil {
			return mapGameStorageError("create listing photo", err)
		}
	}
	return nil
}

func (r *GameRepository) IsListingCategoryActive(ctx context.Context, code string) (bool, error) {
	var active bool
	err := executorFromContext(ctx, r.executor).QueryRow(ctx, `SELECT is_active FROM listing_categories WHERE code = $1`, code).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, mapGameStorageError("get listing category", err)
	}
	return active, nil
}

func (r *GameRepository) AddListingFavorite(ctx context.Context, userID, listingID uuid.UUID, now time.Time) (bool, error) {
	tag, err := executorFromContext(ctx, r.executor).Exec(ctx, `INSERT INTO listing_favorites (user_id, listing_id, created_at) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, userID, listingID, now)
	if err != nil {
		return false, mapGameStorageError("add listing favorite", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *GameRepository) RemoveListingFavorite(ctx context.Context, userID, listingID uuid.UUID) (bool, error) {
	tag, err := executorFromContext(ctx, r.executor).Exec(ctx, `DELETE FROM listing_favorites WHERE user_id = $1 AND listing_id = $2`, userID, listingID)
	if err != nil {
		return false, mapGameStorageError("remove listing favorite", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *GameRepository) ClaimListingFavoriteReward(ctx context.Context, userID, listingID uuid.UUID, now time.Time) (bool, error) {
	tag, err := executorFromContext(ctx, r.executor).Exec(ctx, `
INSERT INTO listing_favorite_rewards (user_id, listing_id, awarded_at)
VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, userID, listingID, now)
	if err != nil {
		return false, mapGameStorageError("claim listing favorite reward", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *GameRepository) RegisterListingView(ctx context.Context, userID, listingID uuid.UUID, now time.Time) (bool, error) {
	tag, err := executorFromContext(ctx, r.executor).Exec(ctx, `
INSERT INTO listing_daily_views (user_id, listing_id, viewed_on, created_at)
VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING`, userID, listingID, now.UTC().Format(time.DateOnly), now)
	if err != nil {
		return false, mapGameStorageError("register listing view", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *GameRepository) CreateFirstListingMessage(ctx context.Context, message model.ListingMessage) (model.ListingMessage, bool, error) {
	row := executorFromContext(ctx, r.executor).QueryRow(ctx, `
INSERT INTO listing_messages (id, listing_id, sender_id, recipient_id, body, created_at)
VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING
RETURNING id, listing_id, sender_id, recipient_id, body, created_at`, message.ID, message.ListingID, message.SenderID, message.RecipientID, message.Body, message.CreatedAt)
	err := row.Scan(&message.ID, &message.ListingID, &message.SenderID, &message.RecipientID, &message.Body, &message.CreatedAt)
	if err == nil {
		return message, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return model.ListingMessage{}, false, mapGameStorageError("create listing message", err)
	}
	err = executorFromContext(ctx, r.executor).QueryRow(ctx, `
SELECT id, listing_id, sender_id, recipient_id, body, created_at FROM listing_messages
WHERE listing_id=$1 AND sender_id=$2`, message.ListingID, message.SenderID).Scan(&message.ID, &message.ListingID, &message.SenderID, &message.RecipientID, &message.Body, &message.CreatedAt)
	if err != nil {
		return model.ListingMessage{}, false, mapMarketplaceStorageError("get first listing message", err)
	}
	return message, false, nil
}

func (r *GameRepository) ListListingMessages(ctx context.Context, userID, listingID uuid.UUID) ([]model.ListingMessage, error) {
	rows, err := executorFromContext(ctx, r.executor).Query(ctx, `
SELECT m.id,m.listing_id,m.sender_id,m.recipient_id,m.body,m.created_at
FROM listing_messages m JOIN listings l ON l.id=m.listing_id
WHERE m.listing_id=$1 AND (m.sender_id=$2 OR m.recipient_id=$2 OR l.owner_id=$2)
ORDER BY m.created_at,m.id`, listingID, userID)
	if err != nil {
		return nil, mapGameStorageError("list listing messages", err)
	}
	defer rows.Close()
	items := make([]model.ListingMessage, 0)
	for rows.Next() {
		var item model.ListingMessage
		if err = rows.Scan(&item.ID, &item.ListingID, &item.SenderID, &item.RecipientID, &item.Body, &item.CreatedAt); err != nil {
			return nil, mapGameStorageError("scan listing message", err)
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return nil, mapGameStorageError("iterate listing messages", err)
	}
	return items, nil
}

func (r *GameRepository) CreateListingDeal(ctx context.Context, deal model.ListingDeal) (model.ListingDeal, error) {
	err := executorFromContext(ctx, r.executor).QueryRow(ctx, `
INSERT INTO listing_deals (id,listing_id,buyer_id,seller_id,delivery_used,completed_at)
VALUES ($1,$2,$3,$4,$5,$6)
RETURNING id,listing_id,buyer_id,seller_id,delivery_used,completed_at`, deal.ID, deal.ListingID, deal.BuyerID, deal.SellerID, deal.DeliveryUsed, deal.CompletedAt).Scan(&deal.ID, &deal.ListingID, &deal.BuyerID, &deal.SellerID, &deal.DeliveryUsed, &deal.CompletedAt)
	if err != nil {
		return model.ListingDeal{}, mapMarketplaceStorageError("create listing deal", err)
	}
	return deal, nil
}

func (r *GameRepository) GetMarketplaceGameRequest(ctx context.Context, eventID uuid.UUID) (model.MarketplaceGameRequest, error) {
	var item model.MarketplaceGameRequest
	err := executorFromContext(ctx, r.executor).QueryRow(ctx, `
SELECT event_id, user_id, listing_id, operation, result, completed_at, created_at
FROM marketplace_game_requests WHERE event_id=$1 FOR UPDATE`, eventID).Scan(&item.EventID, &item.UserID, &item.ListingID, &item.Operation, &item.Result, &item.CompletedAt, &item.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.MarketplaceGameRequest{}, usecase.ErrActionNotFound
	}
	if err != nil {
		return model.MarketplaceGameRequest{}, mapGameStorageError("get marketplace game request", err)
	}
	return item, nil
}

func (r *GameRepository) CreateMarketplaceGameRequest(ctx context.Context, item model.MarketplaceGameRequest) (bool, error) {
	tag, err := executorFromContext(ctx, r.executor).Exec(ctx, `
INSERT INTO marketplace_game_requests (event_id,user_id,listing_id,operation,result,created_at)
VALUES ($1,$2,$3,$4,'{}'::JSONB,$5) ON CONFLICT DO NOTHING`, item.EventID, item.UserID, item.ListingID, item.Operation, item.CreatedAt)
	if err != nil {
		return false, mapGameStorageError("create marketplace game request", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *GameRepository) CompleteMarketplaceGameRequest(ctx context.Context, eventID uuid.UUID, result json.RawMessage, now time.Time) error {
	tag, err := executorFromContext(ctx, r.executor).Exec(ctx, `UPDATE marketplace_game_requests SET result=$2, completed_at=$3 WHERE event_id=$1 AND completed_at IS NULL`, eventID, result, now)
	if err != nil {
		return mapGameStorageError("complete marketplace game request", err)
	}
	if tag.RowsAffected() != 1 {
		return usecase.ErrActionNotFound
	}
	return nil
}

func (r *GameRepository) AwardListingQualityCriteria(ctx context.Context, listingID uuid.UUID, criteria []string, now time.Time) ([]string, error) {
	awarded := make([]string, 0, len(criteria))
	for _, criterion := range criteria {
		tag, err := executorFromContext(ctx, r.executor).Exec(ctx, `INSERT INTO listing_quality_awards (listing_id, criterion, awarded_at) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, listingID, criterion, now)
		if err != nil {
			return nil, mapGameStorageError("award listing quality criterion", err)
		}
		if tag.RowsAffected() == 1 {
			awarded = append(awarded, criterion)
		}
	}
	return awarded, nil
}

func (r *GameRepository) getListing(ctx context.Context, query string, args ...any) (model.Listing, error) {
	listing, err := scanListing(executorFromContext(ctx, r.executor).QueryRow(ctx, query, args...))
	if err != nil {
		return model.Listing{}, err
	}
	photos, err := r.listPhotos(ctx, listing.ID)
	if err != nil {
		return model.Listing{}, err
	}
	listing.Photos = photos
	return listing, nil
}

func (r *GameRepository) collectListings(ctx context.Context, rows pgx.Rows) ([]model.Listing, error) {
	defer rows.Close()
	items := make([]model.Listing, 0)
	for rows.Next() {
		listing, err := scanListing(rows)
		if err != nil {
			return nil, err
		}
		photos, err := r.listPhotos(ctx, listing.ID)
		if err != nil {
			return nil, err
		}
		listing.Photos = photos
		items = append(items, listing)
	}
	if err := rows.Err(); err != nil {
		return nil, mapGameStorageError("iterate listings", err)
	}
	return items, nil
}

func (r *GameRepository) listPhotos(ctx context.Context, listingID uuid.UUID) ([]model.ListingPhoto, error) {
	rows, err := executorFromContext(ctx, r.executor).Query(ctx, `SELECT id,listing_id,url,sort_order FROM listing_photos WHERE listing_id=$1 ORDER BY sort_order,id`, listingID)
	if err != nil {
		return nil, mapGameStorageError("list listing photos", err)
	}
	defer rows.Close()
	photos := make([]model.ListingPhoto, 0)
	for rows.Next() {
		var photo model.ListingPhoto
		if err = rows.Scan(&photo.ID, &photo.ListingID, &photo.URL, &photo.SortOrder); err != nil {
			return nil, mapGameStorageError("scan listing photo", err)
		}
		photos = append(photos, photo)
	}
	if err = rows.Err(); err != nil {
		return nil, mapGameStorageError("iterate listing photos", err)
	}
	return photos, nil
}

func scanListing(row pgx.Row) (model.Listing, error) {
	var listing model.Listing
	err := row.Scan(&listing.ID, &listing.OwnerID, &listing.CategoryCode, &listing.Title, &listing.Description, &listing.PriceKopecks, &listing.Status, &listing.IsDemo, &listing.PublishedAt, &listing.SoldAt, &listing.CreatedAt, &listing.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Listing{}, usecase.ErrListingNotFound
	}
	if err != nil {
		return model.Listing{}, mapGameStorageError("scan listing", err)
	}
	return listing, nil
}
func mapMarketplaceStorageError(operation string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s: %w", operation, usecase.ErrListingNotFound)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode && pgErr.ConstraintName == "listing_deals_listing_id_buyer_id_key" {
		return fmt.Errorf("%s: %w", operation, usecase.ErrDemoPurchaseCompleted)
	}
	return mapGameStorageError(operation, err)
}

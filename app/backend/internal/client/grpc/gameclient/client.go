package gameclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	avitoshav1 "github.com/guitaramust-sudo/Avitosha/app/backend/internal/gen/avitosha/v1"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/realtime"
	internalrpc "github.com/guitaramust-sudo/Avitosha/app/backend/internal/rpc"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

type Client struct{ rpc avitoshav1.GameServiceClient }

func New(rpc avitoshav1.GameServiceClient) *Client { return &Client{rpc: rpc} }

func (c *Client) EnsureProfile(ctx context.Context, userID uuid.UUID, at time.Time) (usecase.GameProfile, error) {
	response, err := c.rpc.EnsureProfile(ctx, userAt(userID, at))
	return decode[usecase.GameProfile](response, err)
}

func (c *Client) RenamePet(ctx context.Context, userID uuid.UUID, name string, at time.Time) (usecase.GameProfile, error) {
	response, err := c.rpc.RenamePet(ctx, &avitoshav1.RenamePetRequest{UserId: userID.String(), Name: name, At: formatTime(at)})
	return decode[usecase.GameProfile](response, err)
}

func (c *Client) ListTasks(ctx context.Context, userID uuid.UUID, at time.Time) ([]model.TaskProgress, error) {
	response, err := c.rpc.ListTasks(ctx, userAt(userID, at))
	return decode[[]model.TaskProgress](response, err)
}

func (c *Client) GetTask(ctx context.Context, userID, taskID uuid.UUID, at time.Time) (model.TaskProgress, error) {
	response, err := c.rpc.GetTask(ctx, taskRequest(userID, taskID, at))
	return decode[model.TaskProgress](response, err)
}

func (c *Client) GetTaskAdvice(ctx context.Context, userID, taskID uuid.UUID, at time.Time) (usecase.TaskAdvice, error) {
	response, err := c.rpc.GetTaskAdvice(ctx, taskRequest(userID, taskID, at))
	return decode[usecase.TaskAdvice](response, err)
}

func (c *Client) GetRoom(ctx context.Context, userID uuid.UUID, at time.Time) ([]model.RoomItemProgress, error) {
	response, err := c.rpc.GetRoom(ctx, userAt(userID, at))
	return decode[[]model.RoomItemProgress](response, err)
}

func (c *Client) GetStory(ctx context.Context, userID uuid.UUID, at time.Time) (model.StorySnapshot, error) {
	response, err := c.rpc.GetStory(ctx, userAt(userID, at))
	return decode[model.StorySnapshot](response, err)
}

func (c *Client) GetDailySummary(ctx context.Context, userID uuid.UUID, at time.Time) (usecase.DailySummary, error) {
	response, err := c.rpc.GetDailySummary(ctx, userAt(userID, at))
	return decode[usecase.DailySummary](response, err)
}

func (c *Client) GetLeaderboard(ctx context.Context, userID uuid.UUID, limit int, at time.Time) (usecase.Leaderboard, error) {
	response, err := c.rpc.GetLeaderboard(ctx, &avitoshav1.LeaderboardRequest{UserId: userID.String(), Limit: int32(limit), At: formatTime(at)})
	return decode[usecase.Leaderboard](response, err)
}

func (c *Client) GetAchievements(ctx context.Context, userID uuid.UUID, at time.Time) ([]model.AchievementProgress, error) {
	response, err := c.rpc.GetAchievements(ctx, userAt(userID, at))
	return decode[[]model.AchievementProgress](response, err)
}

func (c *Client) GetRewardBalances(ctx context.Context, userID uuid.UUID, at time.Time) ([]model.RewardBalance, error) {
	response, err := c.rpc.GetRewardBalances(ctx, userAt(userID, at))
	return decode[[]model.RewardBalance](response, err)
}

func (c *Client) GetRewardWallet(ctx context.Context, userID uuid.UUID, at time.Time) (usecase.RewardWallet, error) {
	response, err := c.rpc.GetRewardWallet(ctx, userAt(userID, at))
	return decode[usecase.RewardWallet](response, err)
}

func (c *Client) ProcessAction(ctx context.Context, command usecase.ProcessActionCommand) (usecase.ProcessActionResult, error) {
	request := &avitoshav1.ProcessActionRequest{
		UserId: command.UserID.String(), EventId: command.EventID.String(), ActionType: string(command.ActionType),
		EntityId: command.EntityID, Category: command.Category, MetadataJson: command.Metadata,
		OccurredAt: formatTime(command.OccurredAt), Now: formatTime(command.Now),
	}
	response, err := c.rpc.ProcessAction(ctx, request)
	return decode[usecase.ProcessActionResult](response, err)
}

func (c *Client) ListCategories(ctx context.Context) ([]model.ListingCategory, error) {
	response, err := c.rpc.ListListingCategories(ctx, &avitoshav1.Empty{})
	return decode[[]model.ListingCategory](response, err)
}
func (c *Client) ListPublic(ctx context.Context, category *string, query string, limit, offset int) (usecase.ListingPage, error) {
	response, err := c.rpc.ListPublicListings(ctx, &avitoshav1.ListingsRequest{Category: category, Query: query, Limit: int32(limit), Offset: int32(offset)})
	return decode[usecase.ListingPage](response, err)
}
func (c *Client) GetPublic(ctx context.Context, listingID uuid.UUID) (model.Listing, error) {
	response, err := c.rpc.GetPublicListing(ctx, &avitoshav1.ListingRequest{ListingId: listingID.String()})
	return decode[model.Listing](response, err)
}
func (c *Client) ListMine(ctx context.Context, userID uuid.UUID, limit, offset int) (usecase.ListingPage, error) {
	response, err := c.rpc.ListMyListings(ctx, &avitoshav1.ListingsRequest{UserId: stringPointer(userID.String()), Limit: int32(limit), Offset: int32(offset)})
	return decode[usecase.ListingPage](response, err)
}
func (c *Client) ListFavorites(ctx context.Context, userID uuid.UUID, limit, offset int) (usecase.ListingPage, error) {
	response, err := c.rpc.ListFavoriteListings(ctx, &avitoshav1.ListingsRequest{UserId: stringPointer(userID.String()), Limit: int32(limit), Offset: int32(offset)})
	return decode[usecase.ListingPage](response, err)
}
func (c *Client) Create(ctx context.Context, command usecase.CreateListingCommand) (model.Listing, error) {
	response, err := c.rpc.CreateListing(ctx, marketplaceCommand(command.OwnerID, uuid.Nil, command.CategoryCode, command.Title, command.Description, command.PriceKopecks, command.PhotoURLs, "", command.Now))
	return decode[model.Listing](response, err)
}
func (c *Client) UpdateWithGame(ctx context.Context, command usecase.UpdateListingCommand) (usecase.MarketplaceActionResult, error) {
	request := marketplaceCommand(command.OwnerID, command.ListingID, command.CategoryCode, command.Title, command.Description, command.PriceKopecks, command.PhotoURLs, "", command.Now)
	request.EventId = command.EventID.String()
	response, err := c.rpc.UpdateListing(ctx, request)
	return decode[usecase.MarketplaceActionResult](response, err)
}
func (c *Client) PublishWithGame(ctx context.Context, userID, listingID, eventID uuid.UUID, now time.Time) (usecase.MarketplaceActionResult, error) {
	request := listingCommand(userID, listingID, now)
	request.EventId = eventID.String()
	response, err := c.rpc.PublishListing(ctx, request)
	return decode[usecase.MarketplaceActionResult](response, err)
}
func (c *Client) Unpublish(ctx context.Context, userID, listingID uuid.UUID, now time.Time) (model.Listing, error) {
	response, err := c.rpc.UnpublishListing(ctx, listingCommand(userID, listingID, now))
	return decode[model.Listing](response, err)
}
func (c *Client) AddFavoriteWithGame(ctx context.Context, userID, listingID, eventID uuid.UUID, now time.Time) (usecase.MarketplaceActionResult, error) {
	request := listingCommand(userID, listingID, now)
	request.EventId = eventID.String()
	response, err := c.rpc.AddListingFavorite(ctx, request)
	return decode[usecase.MarketplaceActionResult](response, err)
}
func (c *Client) RemoveFavorite(ctx context.Context, userID, listingID uuid.UUID) (bool, error) {
	response, err := c.rpc.RemoveListingFavorite(ctx, listingCommand(userID, listingID, time.Now()))
	return decode[bool](response, err)
}
func (c *Client) RegisterViewWithGame(ctx context.Context, userID, listingID, eventID uuid.UUID, now time.Time) (usecase.MarketplaceActionResult, error) {
	request := listingCommand(userID, listingID, now)
	request.EventId = eventID.String()
	response, err := c.rpc.RegisterListingView(ctx, request)
	return decode[usecase.MarketplaceActionResult](response, err)
}
func (c *Client) ContactSellerWithGame(ctx context.Context, userID, listingID, eventID uuid.UUID, body string, now time.Time) (usecase.MarketplaceActionResult, error) {
	request := marketplaceCommand(userID, listingID, "", "", "", 0, nil, body, now)
	request.EventId = eventID.String()
	response, err := c.rpc.ContactSeller(ctx, request)
	return decode[usecase.MarketplaceActionResult](response, err)
}
func (c *Client) ListMessages(ctx context.Context, userID, listingID uuid.UUID) ([]model.ListingMessage, error) {
	response, err := c.rpc.ListListingMessages(ctx, &avitoshav1.ListingRequest{ListingId: listingID.String(), UserId: stringPointer(userID.String())})
	return decode[[]model.ListingMessage](response, err)
}
func (c *Client) PurchaseWithGame(ctx context.Context, command usecase.PurchaseListingCommand) (usecase.MarketplaceActionResult, error) {
	response, err := c.rpc.PurchaseListing(ctx, &avitoshav1.PurchaseListingRequest{UserId: command.BuyerID.String(), ListingId: command.ListingID.String(), DeliveryUsed: command.DeliveryUsed, At: formatTime(command.Now), EventId: command.EventID.String()})
	return decode[usecase.MarketplaceActionResult](response, err)
}

func listingCommand(userID, listingID uuid.UUID, at time.Time) *avitoshav1.ListingCommand {
	return &avitoshav1.ListingCommand{UserId: userID.String(), ListingId: listingID.String(), At: formatTime(at)}
}
func marketplaceCommand(userID, listingID uuid.UUID, category, title, description string, price int64, photos []string, body string, at time.Time) *avitoshav1.MarketplaceCommand {
	request := &avitoshav1.MarketplaceCommand{UserId: userID.String(), CategoryCode: category, Title: title, Description: description, PriceKopecks: price, PhotoUrls: photos, MessageBody: body, At: formatTime(at)}
	if listingID != uuid.Nil {
		request.ListingId = listingID.String()
	}
	return request
}
func stringPointer(value string) *string { return &value }

func (c *Client) Subscribe(userID uuid.UUID) realtime.EventSubscription {
	ctx, cancel := context.WithCancel(context.Background())
	subscription := &interfaceSubscription{messages: make(chan []model.DomainEvent, 16), cancel: cancel}
	stream, err := c.rpc.SubscribeEvents(ctx, &avitoshav1.SubscribeEventsRequest{UserId: userID.String()})
	if err != nil {
		close(subscription.messages)
		return subscription
	}
	go subscription.receive(stream)
	return subscription
}

// interfaceSubscription is returned by value so every WebSocket owns one
// independent gRPC stream. Close remains idempotent through sync.Once.
type interfaceSubscription struct {
	messages chan []model.DomainEvent
	cancel   context.CancelFunc
	once     sync.Once
}

func (s *interfaceSubscription) Messages() <-chan []model.DomainEvent { return s.messages }
func (s *interfaceSubscription) Close()                               { s.once.Do(s.cancel) }

func (s *interfaceSubscription) receive(stream avitoshav1.GameService_SubscribeEventsClient) {
	defer close(s.messages)
	for {
		batch, err := stream.Recv()
		if err != nil {
			if err != io.EOF {
				s.cancel()
			}
			return
		}
		var events []model.DomainEvent
		if json.Unmarshal(batch.GetPayloadJson(), &events) != nil {
			s.cancel()
			return
		}
		select {
		case s.messages <- events:
		case <-stream.Context().Done():
			return
		}
	}
}

func decode[T any](response *avitoshav1.JsonResponse, err error) (T, error) {
	var value T
	if err != nil {
		return value, fmt.Errorf("game grpc request: %w", internalrpc.DecodeGameError(err))
	}
	if response == nil || json.Unmarshal(response.GetPayloadJson(), &value) != nil {
		return value, fmt.Errorf("decode game grpc response: %w", usecase.ErrUnexpectedStorage)
	}
	return value, nil
}

func userAt(userID uuid.UUID, at time.Time) *avitoshav1.UserAtRequest {
	return &avitoshav1.UserAtRequest{UserId: userID.String(), At: formatTime(at)}
}

func taskRequest(userID, taskID uuid.UUID, at time.Time) *avitoshav1.TaskRequest {
	return &avitoshav1.TaskRequest{UserId: userID.String(), TaskId: taskID.String(), At: formatTime(at)}
}

func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

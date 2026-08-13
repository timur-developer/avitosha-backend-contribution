package gameserver

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	avitoshav1 "github.com/guitaramust-sudo/Avitosha/app/backend/internal/gen/avitosha/v1"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/realtime"
	internalrpc "github.com/guitaramust-sudo/Avitosha/app/backend/internal/rpc"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GameUseCase interface {
	EnsureProfile(context.Context, uuid.UUID, time.Time) (usecase.GameProfile, error)
	RenamePet(context.Context, uuid.UUID, string, time.Time) (usecase.GameProfile, error)
	ListTasks(context.Context, uuid.UUID, time.Time) ([]model.TaskProgress, error)
	GetTask(context.Context, uuid.UUID, uuid.UUID, time.Time) (model.TaskProgress, error)
	GetTaskAdvice(context.Context, uuid.UUID, uuid.UUID, time.Time) (usecase.TaskAdvice, error)
	GetRoom(context.Context, uuid.UUID, time.Time) ([]model.RoomItemProgress, error)
	GetStory(context.Context, uuid.UUID, time.Time) (model.StorySnapshot, error)
	GetDailySummary(context.Context, uuid.UUID, time.Time) (usecase.DailySummary, error)
	GetLeaderboard(context.Context, uuid.UUID, int, time.Time) (usecase.Leaderboard, error)
	GetAchievements(context.Context, uuid.UUID, time.Time) ([]model.AchievementProgress, error)
	GetRewardBalances(context.Context, uuid.UUID, time.Time) ([]model.RewardBalance, error)
	GetRewardWallet(context.Context, uuid.UUID, time.Time) (usecase.RewardWallet, error)
	ProcessAction(context.Context, usecase.ProcessActionCommand) (usecase.ProcessActionResult, error)
}

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

type Server struct {
	avitoshav1.UnimplementedGameServiceServer
	game        GameUseCase
	marketplace MarketplaceUseCase
	hub         *realtime.Hub
}

func New(game GameUseCase, marketplace MarketplaceUseCase, hub *realtime.Hub) *Server {
	return &Server{game: game, marketplace: marketplace, hub: hub}
}

func (s *Server) EnsureProfile(ctx context.Context, request *avitoshav1.UserAtRequest) (*avitoshav1.JsonResponse, error) {
	userID, at, err := parseUserAt(request.GetUserId(), request.GetAt())
	if err != nil {
		return nil, err
	}
	return encode(s.game.EnsureProfile(ctx, userID, at))
}

func (s *Server) RenamePet(ctx context.Context, request *avitoshav1.RenamePetRequest) (*avitoshav1.JsonResponse, error) {
	userID, at, err := parseUserAt(request.GetUserId(), request.GetAt())
	if err != nil {
		return nil, err
	}
	return encode(s.game.RenamePet(ctx, userID, request.GetName(), at))
}

func (s *Server) ListTasks(ctx context.Context, request *avitoshav1.UserAtRequest) (*avitoshav1.JsonResponse, error) {
	userID, at, err := parseUserAt(request.GetUserId(), request.GetAt())
	if err != nil {
		return nil, err
	}
	return encode(s.game.ListTasks(ctx, userID, at))
}

func (s *Server) GetTask(ctx context.Context, request *avitoshav1.TaskRequest) (*avitoshav1.JsonResponse, error) {
	userID, taskID, at, err := parseTaskRequest(request)
	if err != nil {
		return nil, err
	}
	return encode(s.game.GetTask(ctx, userID, taskID, at))
}

func (s *Server) GetTaskAdvice(ctx context.Context, request *avitoshav1.TaskRequest) (*avitoshav1.JsonResponse, error) {
	userID, taskID, at, err := parseTaskRequest(request)
	if err != nil {
		return nil, err
	}
	return encode(s.game.GetTaskAdvice(ctx, userID, taskID, at))
}

func (s *Server) GetRoom(ctx context.Context, request *avitoshav1.UserAtRequest) (*avitoshav1.JsonResponse, error) {
	userID, at, err := parseUserAt(request.GetUserId(), request.GetAt())
	if err != nil {
		return nil, err
	}
	return encode(s.game.GetRoom(ctx, userID, at))
}

func (s *Server) GetStory(ctx context.Context, request *avitoshav1.UserAtRequest) (*avitoshav1.JsonResponse, error) {
	userID, at, err := parseUserAt(request.GetUserId(), request.GetAt())
	if err != nil {
		return nil, err
	}
	return encode(s.game.GetStory(ctx, userID, at))
}

func (s *Server) GetDailySummary(ctx context.Context, request *avitoshav1.UserAtRequest) (*avitoshav1.JsonResponse, error) {
	userID, at, err := parseUserAt(request.GetUserId(), request.GetAt())
	if err != nil {
		return nil, err
	}
	return encode(s.game.GetDailySummary(ctx, userID, at))
}

func (s *Server) GetLeaderboard(ctx context.Context, request *avitoshav1.LeaderboardRequest) (*avitoshav1.JsonResponse, error) {
	userID, at, err := parseUserAt(request.GetUserId(), request.GetAt())
	if err != nil {
		return nil, err
	}
	return encode(s.game.GetLeaderboard(ctx, userID, int(request.GetLimit()), at))
}

func (s *Server) GetAchievements(ctx context.Context, request *avitoshav1.UserAtRequest) (*avitoshav1.JsonResponse, error) {
	userID, at, err := parseUserAt(request.GetUserId(), request.GetAt())
	if err != nil {
		return nil, err
	}
	return encode(s.game.GetAchievements(ctx, userID, at))
}

func (s *Server) GetRewardBalances(ctx context.Context, request *avitoshav1.UserAtRequest) (*avitoshav1.JsonResponse, error) {
	userID, at, err := parseUserAt(request.GetUserId(), request.GetAt())
	if err != nil {
		return nil, err
	}
	return encode(s.game.GetRewardBalances(ctx, userID, at))
}

func (s *Server) GetRewardWallet(ctx context.Context, request *avitoshav1.UserAtRequest) (*avitoshav1.JsonResponse, error) {
	userID, at, err := parseUserAt(request.GetUserId(), request.GetAt())
	if err != nil {
		return nil, err
	}
	return encode(s.game.GetRewardWallet(ctx, userID, at))
}

func (s *Server) ProcessAction(ctx context.Context, request *avitoshav1.ProcessActionRequest) (*avitoshav1.JsonResponse, error) {
	userID, err := parseUUID("user_id", request.GetUserId())
	if err != nil {
		return nil, err
	}
	eventID, err := parseUUID("event_id", request.GetEventId())
	if err != nil {
		return nil, err
	}
	occurredAt, err := parseTime("occurred_at", request.GetOccurredAt())
	if err != nil {
		return nil, err
	}
	now, err := parseTime("now", request.GetNow())
	if err != nil {
		return nil, err
	}
	command := usecase.ProcessActionCommand{
		UserID: userID, EventID: eventID, ActionType: model.ActionType(request.GetActionType()),
		EntityID: request.EntityId, Category: request.Category, Metadata: json.RawMessage(request.GetMetadataJson()),
		OccurredAt: occurredAt, Now: now,
	}
	return encode(s.game.ProcessAction(ctx, command))
}

func (s *Server) ListListingCategories(ctx context.Context, _ *avitoshav1.Empty) (*avitoshav1.JsonResponse, error) {
	return encode(s.marketplace.ListCategories(ctx))
}

func (s *Server) ListPublicListings(ctx context.Context, request *avitoshav1.ListingsRequest) (*avitoshav1.JsonResponse, error) {
	return encode(s.marketplace.ListPublic(ctx, request.Category, request.GetQuery(), int(request.GetLimit()), int(request.GetOffset())))
}

func (s *Server) GetPublicListing(ctx context.Context, request *avitoshav1.ListingRequest) (*avitoshav1.JsonResponse, error) {
	listingID, err := parseUUID("listing_id", request.GetListingId())
	if err != nil {
		return nil, err
	}
	return encode(s.marketplace.GetPublic(ctx, listingID))
}

func (s *Server) ListMyListings(ctx context.Context, request *avitoshav1.ListingsRequest) (*avitoshav1.JsonResponse, error) {
	userID, err := parseUUID("user_id", request.GetUserId())
	if err != nil {
		return nil, err
	}
	return encode(s.marketplace.ListMine(ctx, userID, int(request.GetLimit()), int(request.GetOffset())))
}

func (s *Server) CreateListing(ctx context.Context, request *avitoshav1.MarketplaceCommand) (*avitoshav1.JsonResponse, error) {
	userID, now, err := parseUserAt(request.GetUserId(), request.GetAt())
	if err != nil {
		return nil, err
	}
	return encode(s.marketplace.Create(ctx, usecase.CreateListingCommand{OwnerID: userID, CategoryCode: request.GetCategoryCode(), Title: request.GetTitle(), Description: request.GetDescription(), PriceKopecks: request.GetPriceKopecks(), PhotoURLs: request.GetPhotoUrls(), Now: now}))
}

func (s *Server) UpdateListing(ctx context.Context, request *avitoshav1.MarketplaceCommand) (*avitoshav1.JsonResponse, error) {
	userID, now, err := parseUserAt(request.GetUserId(), request.GetAt())
	if err != nil {
		return nil, err
	}
	listingID, err := parseUUID("listing_id", request.GetListingId())
	if err != nil {
		return nil, err
	}
	eventID, err := parseUUID("event_id", request.GetEventId())
	if err != nil {
		return nil, err
	}
	return encode(s.marketplace.UpdateWithGame(ctx, usecase.UpdateListingCommand{OwnerID: userID, ListingID: listingID, CategoryCode: request.GetCategoryCode(), Title: request.GetTitle(), Description: request.GetDescription(), PriceKopecks: request.GetPriceKopecks(), PhotoURLs: request.GetPhotoUrls(), EventID: eventID, Now: now}))
}

func (s *Server) PublishListing(ctx context.Context, request *avitoshav1.ListingCommand) (*avitoshav1.JsonResponse, error) {
	userID, listingID, at, err := parseListingCommand(request)
	if err != nil {
		return nil, err
	}
	eventID, err := parseUUID("event_id", request.GetEventId())
	if err != nil {
		return nil, err
	}
	return encode(s.marketplace.PublishWithGame(ctx, userID, listingID, eventID, at))
}
func (s *Server) UnpublishListing(ctx context.Context, request *avitoshav1.ListingCommand) (*avitoshav1.JsonResponse, error) {
	userID, listingID, at, err := parseListingCommand(request)
	if err != nil {
		return nil, err
	}
	return encode(s.marketplace.Unpublish(ctx, userID, listingID, at))
}
func (s *Server) AddListingFavorite(ctx context.Context, request *avitoshav1.ListingCommand) (*avitoshav1.JsonResponse, error) {
	userID, listingID, at, err := parseListingCommand(request)
	if err != nil {
		return nil, err
	}
	eventID, err := parseUUID("event_id", request.GetEventId())
	if err != nil {
		return nil, err
	}
	return encode(s.marketplace.AddFavoriteWithGame(ctx, userID, listingID, eventID, at))
}
func (s *Server) RemoveListingFavorite(ctx context.Context, request *avitoshav1.ListingCommand) (*avitoshav1.JsonResponse, error) {
	userID, listingID, _, err := parseListingCommand(request)
	if err != nil {
		return nil, err
	}
	return encode(s.marketplace.RemoveFavorite(ctx, userID, listingID))
}
func (s *Server) RegisterListingView(ctx context.Context, request *avitoshav1.ListingCommand) (*avitoshav1.JsonResponse, error) {
	userID, listingID, at, err := parseListingCommand(request)
	if err != nil {
		return nil, err
	}
	eventID, err := parseUUID("event_id", request.GetEventId())
	if err != nil {
		return nil, err
	}
	return encode(s.marketplace.RegisterViewWithGame(ctx, userID, listingID, eventID, at))
}

func (s *Server) ListFavoriteListings(ctx context.Context, request *avitoshav1.ListingsRequest) (*avitoshav1.JsonResponse, error) {
	userID, err := parseUUID("user_id", request.GetUserId())
	if err != nil {
		return nil, err
	}
	return encode(s.marketplace.ListFavorites(ctx, userID, int(request.GetLimit()), int(request.GetOffset())))
}
func (s *Server) ContactSeller(ctx context.Context, request *avitoshav1.MarketplaceCommand) (*avitoshav1.JsonResponse, error) {
	userID, now, err := parseUserAt(request.GetUserId(), request.GetAt())
	if err != nil {
		return nil, err
	}
	listingID, err := parseUUID("listing_id", request.GetListingId())
	if err != nil {
		return nil, err
	}
	eventID, err := parseUUID("event_id", request.GetEventId())
	if err != nil {
		return nil, err
	}
	return encode(s.marketplace.ContactSellerWithGame(ctx, userID, listingID, eventID, request.GetMessageBody(), now))
}
func (s *Server) ListListingMessages(ctx context.Context, request *avitoshav1.ListingRequest) (*avitoshav1.JsonResponse, error) {
	userID, err := parseUUID("user_id", request.GetUserId())
	if err != nil {
		return nil, err
	}
	listingID, err := parseUUID("listing_id", request.GetListingId())
	if err != nil {
		return nil, err
	}
	return encode(s.marketplace.ListMessages(ctx, userID, listingID))
}
func (s *Server) PurchaseListing(ctx context.Context, request *avitoshav1.PurchaseListingRequest) (*avitoshav1.JsonResponse, error) {
	userID, now, err := parseUserAt(request.GetUserId(), request.GetAt())
	if err != nil {
		return nil, err
	}
	listingID, err := parseUUID("listing_id", request.GetListingId())
	if err != nil {
		return nil, err
	}
	eventID, err := parseUUID("event_id", request.GetEventId())
	if err != nil {
		return nil, err
	}
	return encode(s.marketplace.PurchaseWithGame(ctx, usecase.PurchaseListingCommand{BuyerID: userID, ListingID: listingID, DeliveryUsed: request.GetDeliveryUsed(), EventID: eventID, Now: now}))
}

func (s *Server) SubscribeEvents(request *avitoshav1.SubscribeEventsRequest, stream avitoshav1.GameService_SubscribeEventsServer) error {
	userID, err := parseUUID("user_id", request.GetUserId())
	if err != nil {
		return err
	}
	subscription := s.hub.Subscribe(userID)
	defer subscription.Close()
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case events, ok := <-subscription.Messages():
			if !ok {
				return nil
			}
			payload, marshalErr := json.Marshal(events)
			if marshalErr != nil {
				return status.Error(codes.Internal, "encode event batch")
			}
			if sendErr := stream.Send(&avitoshav1.EventBatch{PayloadJson: payload}); sendErr != nil {
				return sendErr
			}
		}
	}
}

func encode[T any](value T, err error) (*avitoshav1.JsonResponse, error) {
	if err != nil {
		return nil, internalrpc.GameError(err)
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, status.Error(codes.Internal, "encode response")
	}
	return &avitoshav1.JsonResponse{PayloadJson: payload}, nil
}

func parseUserAt(userIDValue, atValue string) (uuid.UUID, time.Time, error) {
	userID, err := parseUUID("user_id", userIDValue)
	if err != nil {
		return uuid.Nil, time.Time{}, err
	}
	at, err := parseTime("at", atValue)
	return userID, at, err
}

func parseTaskRequest(request *avitoshav1.TaskRequest) (uuid.UUID, uuid.UUID, time.Time, error) {
	userID, at, err := parseUserAt(request.GetUserId(), request.GetAt())
	if err != nil {
		return uuid.Nil, uuid.Nil, time.Time{}, err
	}
	taskID, err := parseUUID("task_id", request.GetTaskId())
	return userID, taskID, at, err
}

func parseListingCommand(request *avitoshav1.ListingCommand) (uuid.UUID, uuid.UUID, time.Time, error) {
	userID, at, err := parseUserAt(request.GetUserId(), request.GetAt())
	if err != nil {
		return uuid.Nil, uuid.Nil, time.Time{}, err
	}
	listingID, err := parseUUID("listing_id", request.GetListingId())
	return userID, listingID, at, err
}

func parseUUID(field, value string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, status.Errorf(codes.InvalidArgument, "%s must be a UUID", field)
	}
	return parsed, nil
}

func parseTime(field, value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, status.Errorf(codes.InvalidArgument, "%s must be RFC3339", field)
	}
	return parsed, nil
}

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

type fakeGameUseCase struct {
	ensureProfileFunc     func(context.Context, uuid.UUID, time.Time) (usecase.GameProfile, error)
	renamePetFunc         func(context.Context, uuid.UUID, string, time.Time) (usecase.GameProfile, error)
	listTasksFunc         func(context.Context, uuid.UUID, time.Time) ([]model.TaskProgress, error)
	getTaskFunc           func(context.Context, uuid.UUID, uuid.UUID, time.Time) (model.TaskProgress, error)
	getRoomFunc           func(context.Context, uuid.UUID, time.Time) ([]model.RoomItemProgress, error)
	getStoryFunc          func(context.Context, uuid.UUID, time.Time) (model.StorySnapshot, error)
	getDailyFunc          func(context.Context, uuid.UUID, time.Time) (usecase.DailySummary, error)
	getLeaderboardFunc    func(context.Context, uuid.UUID, int, time.Time) (usecase.Leaderboard, error)
	getAchievementsFunc   func(context.Context, uuid.UUID, time.Time) ([]model.AchievementProgress, error)
	getRewardBalancesFunc func(context.Context, uuid.UUID, time.Time) ([]model.RewardBalance, error)
	getRewardWalletFunc   func(context.Context, uuid.UUID, time.Time) (usecase.RewardWallet, error)
	processActionFunc     func(context.Context, usecase.ProcessActionCommand) (usecase.ProcessActionResult, error)
}

func (fake fakeGameUseCase) EnsureProfile(ctx context.Context, userID uuid.UUID, now time.Time) (usecase.GameProfile, error) {
	return fake.ensureProfileFunc(ctx, userID, now)
}

func (fake fakeGameUseCase) RenamePet(ctx context.Context, userID uuid.UUID, name string, now time.Time) (usecase.GameProfile, error) {
	return fake.renamePetFunc(ctx, userID, name, now)
}

func (fake fakeGameUseCase) ListTasks(ctx context.Context, userID uuid.UUID, now time.Time) ([]model.TaskProgress, error) {
	return fake.listTasksFunc(ctx, userID, now)
}

func (fake fakeGameUseCase) GetTask(ctx context.Context, userID, taskID uuid.UUID, now time.Time) (model.TaskProgress, error) {
	return fake.getTaskFunc(ctx, userID, taskID, now)
}

func (fake fakeGameUseCase) GetRoom(ctx context.Context, userID uuid.UUID, now time.Time) ([]model.RoomItemProgress, error) {
	return fake.getRoomFunc(ctx, userID, now)
}

func (fake fakeGameUseCase) GetStory(ctx context.Context, userID uuid.UUID, now time.Time) (model.StorySnapshot, error) {
	return fake.getStoryFunc(ctx, userID, now)
}

func (fake fakeGameUseCase) GetDailySummary(ctx context.Context, userID uuid.UUID, now time.Time) (usecase.DailySummary, error) {
	return fake.getDailyFunc(ctx, userID, now)
}

func (fake fakeGameUseCase) GetLeaderboard(ctx context.Context, userID uuid.UUID, limit int, now time.Time) (usecase.Leaderboard, error) {
	return fake.getLeaderboardFunc(ctx, userID, limit, now)
}

func (fake fakeGameUseCase) GetAchievements(ctx context.Context, userID uuid.UUID, now time.Time) ([]model.AchievementProgress, error) {
	return fake.getAchievementsFunc(ctx, userID, now)
}

func (fake fakeGameUseCase) GetRewardBalances(ctx context.Context, userID uuid.UUID, now time.Time) ([]model.RewardBalance, error) {
	return fake.getRewardBalancesFunc(ctx, userID, now)
}

func (fake fakeGameUseCase) GetRewardWallet(ctx context.Context, userID uuid.UUID, now time.Time) (usecase.RewardWallet, error) {
	return fake.getRewardWalletFunc(ctx, userID, now)
}

func (fake fakeGameUseCase) ProcessAction(ctx context.Context, command usecase.ProcessActionCommand) (usecase.ProcessActionResult, error) {
	return fake.processActionFunc(ctx, command)
}

func TestGameRoutesRequireIdentity(t *testing.T) {
	router := newGameTestRouter(RouterDependencies{})
	for _, route := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/pet"},
		{http.MethodPatch, "/api/v1/pet"},
		{http.MethodGet, "/api/v1/tasks"},
		{http.MethodPost, "/api/v1/actions"},
		{http.MethodGet, "/api/v1/room"},
		{http.MethodGet, "/api/v1/story"},
		{http.MethodGet, "/api/v1/daily-summary"},
		{http.MethodGet, "/api/v1/leaderboard"},
		{http.MethodGet, "/api/v1/achievements"},
		{http.MethodGet, "/api/v1/rewards/balance"},
		{http.MethodGet, "/api/v1/rewards/wallet"},
	} {
		request := httptest.NewRequestWithContext(context.Background(), route.method, route.path, nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s status = %d, want 401", route.method, route.path, recorder.Code)
		}
	}
}

func TestGetGamePetAcceptsXUserIDAndReturnsProductProfile(t *testing.T) {
	userID := uuid.New()
	petID := uuid.New()
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	service := completeFakeGameUseCase()
	service.ensureProfileFunc = func(_ context.Context, gotUserID uuid.UUID, gotNow time.Time) (usecase.GameProfile, error) {
		if gotUserID != userID || !gotNow.Equal(now) {
			t.Fatalf("EnsureProfile(%s, %v)", gotUserID, gotNow)
		}
		return usecase.GameProfile{
			Pet: model.Pet{ID: petID, UserID: userID, Name: "Авитоша", Level: 2, GrowthXP: 130, Mood: model.PetMoodProud},
			Story: model.StorySnapshot{
				Story:    model.Story{Code: "FIRST_ROOM", Title: "Обустроить первую комнату", TotalStages: 5},
				Progress: model.UserStoryProgress{CurrentStage: 2, Status: model.StoryStatusActive},
			},
			NextLevelXP: gameIntPointer(250),
		}, nil
	}
	router := newGameTestRouter(RouterDependencies{GameService: service, Now: func() time.Time { return now }})
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/pet", nil)
	request.Header.Set("X-User-ID", userID.String())
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["growthXp"] != float64(130) || body["mood"] != "PROUD" {
		t.Fatalf("body = %v", body)
	}
}

func TestRenamePetReturnsNormalizedProfile(t *testing.T) {
	userID := uuid.New()
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	service := completeFakeGameUseCase()
	service.renamePetFunc = func(_ context.Context, gotUserID uuid.UUID, name string, gotNow time.Time) (usecase.GameProfile, error) {
		if gotUserID != userID || name != "  мурзик  " || !gotNow.Equal(now) {
			t.Fatalf("RenamePet(%s, %q, %v)", gotUserID, name, gotNow)
		}
		return usecase.GameProfile{
			Pet: model.Pet{ID: uuid.New(), UserID: userID, Name: "Мурзик", Level: 1, Mood: model.PetMoodCalm},
			Story: model.StorySnapshot{
				Story:    model.Story{Code: usecase.FirstRoomStoryCode, TotalStages: 5},
				Progress: model.UserStoryProgress{Status: model.StoryStatusActive},
			},
		}, nil
	}
	router := newGameTestRouter(RouterDependencies{GameService: service, Now: func() time.Time { return now }})
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPatch, "/api/v1/pet", bytes.NewBufferString(`{"name":"  мурзик  "}`))
	request.Header.Set("X-User-ID", userID.String())
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["name"] != "Мурзик" {
		t.Fatalf("body = %v", body)
	}
}

func TestRenamePetRejectsForbiddenName(t *testing.T) {
	service := completeFakeGameUseCase()
	service.renamePetFunc = func(context.Context, uuid.UUID, string, time.Time) (usecase.GameProfile, error) {
		return usecase.GameProfile{}, usecase.ErrForbiddenPetName
	}
	router := newGameTestRouter(RouterDependencies{GameService: service})
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPatch, "/api/v1/pet", bytes.NewBufferString(`{"name":"запрещено"}`))
	request.Header.Set("X-User-ID", uuid.NewString())
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusBadRequest, "forbidden_pet_name", "Это имя нельзя использовать. Выберите доброе имя без оскорблений")
}

func TestRenamePetRejectsUnknownRequestFields(t *testing.T) {
	router := newGameTestRouter(RouterDependencies{GameService: completeFakeGameUseCase()})
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPatch, "/api/v1/pet", bytes.NewBufferString(`{"name":"Мурзик","admin":true}`))
	request.Header.Set("X-User-ID", uuid.NewString())
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestGameReadRoutesReturnTasksRoomAndStory(t *testing.T) {
	userID := uuid.New()
	taskID := uuid.New()
	category := "FURNITURE"
	itemCode := "DESK"
	stage := 1
	task := model.Task{ID: taskID, Code: "VIEW_FURNITURE_ADS", ActionType: model.ActionTypeAdViewed,
		Category: &category, TargetValue: 5, XPReward: 30, RoomItemCode: &itemCode, StoryStage: &stage}
	progress := model.TaskProgress{Task: task, Progress: model.UserTask{TaskID: taskID, Progress: 3, TargetValue: 5, Status: model.TaskStatusActive}}
	service := completeFakeGameUseCase()
	service.listTasksFunc = func(context.Context, uuid.UUID, time.Time) ([]model.TaskProgress, error) {
		return []model.TaskProgress{progress}, nil
	}
	service.getTaskFunc = func(context.Context, uuid.UUID, uuid.UUID, time.Time) (model.TaskProgress, error) {
		return progress, nil
	}
	service.getRoomFunc = func(context.Context, uuid.UUID, time.Time) ([]model.RoomItemProgress, error) {
		return []model.RoomItemProgress{{Item: model.RoomItem{Code: "DESK", Name: "Стол"}, Status: model.RoomItemStatusLocked}}, nil
	}
	service.getStoryFunc = func(context.Context, uuid.UUID, time.Time) (model.StorySnapshot, error) {
		return model.StorySnapshot{
			Story:    model.Story{Code: "FIRST_ROOM", TotalStages: 5},
			Progress: model.UserStoryProgress{CurrentStage: 0, Status: model.StoryStatusActive}, NextTask: &task,
		}, nil
	}
	router := newGameTestRouter(RouterDependencies{GameService: service})
	for _, path := range []string{"/api/v1/tasks", "/api/v1/tasks/" + taskID.String(), "/api/v1/room", "/api/v1/story"} {
		request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
		request.Header.Set("X-User-ID", userID.String())
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, body = %s", path, recorder.Code, recorder.Body.String())
		}
	}
}

func TestGetRewardBalancesReturnsCurrentAndLifetimeAmounts(t *testing.T) {
	userID := uuid.New()
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	service := completeFakeGameUseCase()
	service.getRewardBalancesFunc = func(_ context.Context, gotUserID uuid.UUID, gotNow time.Time) ([]model.RewardBalance, error) {
		if gotUserID != userID || !gotNow.Equal(now) {
			t.Fatalf("GetRewardBalances(%s, %v)", gotUserID, gotNow)
		}
		return []model.RewardBalance{{
			UserID: userID, RewardType: usecase.DefaultRewardType,
			Balance: 30, EarnedTotal: 30, UpdatedAt: now,
		}}, nil
	}
	router := newGameTestRouter(RouterDependencies{GameService: service, Now: func() time.Time { return now }})
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/rewards/balance", nil)
	request.Header.Set("X-User-ID", userID.String())
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var body struct {
		Balances []rewardBalanceDTO `json:"balances"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Balances) != 1 || body.Balances[0].Type != usecase.DefaultRewardType || body.Balances[0].Balance != 30 {
		t.Fatalf("body = %+v", body)
	}
}

func TestGetRewardWalletReturnsCatalogAndNextGoal(t *testing.T) {
	userID := uuid.New()
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	service := completeFakeGameUseCase()
	service.getRewardWalletFunc = func(_ context.Context, gotUserID uuid.UUID, gotNow time.Time) (usecase.RewardWallet, error) {
		if gotUserID != userID || !gotNow.Equal(now) {
			t.Fatalf("GetRewardWallet(%s, %v)", gotUserID, gotNow)
		}
		return usecase.RewardWallet{
			Balance: model.RewardBalance{RewardType: usecase.DefaultRewardType, Balance: 92, EarnedTotal: 92, UpdatedAt: now},
			Catalog: []usecase.RewardCatalogEntry{{
				Item: model.RewardCatalogItem{
					Code: "PROMO_BOOST", Title: "Boost", Description: "Promotion bonus",
					RewardType: usecase.DefaultRewardType, PerkType: "PROMOTION", Threshold: 75,
				},
				Unlocked: true, ProgressCurrent: 75, ProgressTarget: 75,
			}},
			NextGoal: &usecase.RewardGoal{
				Item: model.RewardCatalogItem{
					Code: "AUTOTEKA_CHECK", Title: "Autoteka", RewardType: usecase.DefaultRewardType,
					PerkType: "AUTOTEKA",
				},
				Current: 92, Target: 110, Remaining: 18,
			},
		}, nil
	}
	router := newGameTestRouter(RouterDependencies{GameService: service, Now: func() time.Time { return now }})
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/rewards/wallet", nil)
	request.Header.Set("X-User-ID", userID.String())
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"code":"AUTOTEKA_CHECK"`)) ||
		!bytes.Contains(recorder.Body.Bytes(), []byte(`"earnedTotal":92`)) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}

func TestRoomProgressCountsOnlyFirstStoryItems(t *testing.T) {
	items := make([]model.RoomItemProgress, 0, 9)
	for index, code := range []string{"BOX", "DESK", "LAMP", "CHAIR", "PLANT", "POSTER", "RUG", "SHELF", "CLOCK"} {
		status := model.RoomItemStatusLocked
		if code == "BOX" || code == "RUG" {
			status = model.RoomItemStatusPlaced
		}
		items = append(items, model.RoomItemProgress{
			Item:   model.RoomItem{Code: code, SortOrder: index},
			Status: status,
		})
	}

	room := newRoomDTO(items)
	if room.Progress != "1/6" {
		t.Fatalf("progress = %q, want 1/6", room.Progress)
	}
	if len(room.Items) != 9 {
		t.Fatalf("items = %d, want 9", len(room.Items))
	}
}

func TestProcessActionValidatesAndFlattensDomainEvents(t *testing.T) {
	userID := uuid.New()
	eventID := uuid.New()
	domainEventID := uuid.New()
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	service := completeFakeGameUseCase()
	service.processActionFunc = func(_ context.Context, command usecase.ProcessActionCommand) (usecase.ProcessActionResult, error) {
		if command.UserID != userID || command.EventID != eventID || command.ActionType != model.ActionTypeAdViewed {
			t.Fatalf("command = %+v", command)
		}
		return usecase.ProcessActionResult{
			ActionID: uuid.New(), Events: []model.DomainEvent{{
				ID: domainEventID, Type: model.DomainEventTaskProgressUpdated, OccurredAt: now,
				Payload: json.RawMessage(`{"taskCode":"VIEW_FURNITURE_ADS","progress":1,"target":5}`),
			}},
		}, nil
	}
	router := newGameTestRouter(RouterDependencies{GameService: service, Now: func() time.Time { return now }})
	body := []byte(`{"eventId":"` + eventID.String() + `","type":"AD_VIEWED","entityId":"advert-1","category":"FURNITURE","occurredAt":"2026-08-05T12:00:00Z","metadata":{}}`)
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/actions", bytes.NewReader(body))
	request.Header.Set("X-User-ID", userID.String())
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"taskCode":"VIEW_FURNITURE_ADS"`)) ||
		bytes.Contains(recorder.Body.Bytes(), []byte(`"payload"`)) {
		t.Fatalf("events were not flattened: %s", recorder.Body.String())
	}
}

func TestProcessActionRejectsMalformedBody(t *testing.T) {
	router := newGameTestRouter(RouterDependencies{GameService: completeFakeGameUseCase()})
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/actions", bytes.NewBufferString(`{"type":"AD_VIEWED"}`))
	request.Header.Set("X-User-ID", uuid.NewString())
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestProgressRoutesReturnDailyLeaderboardAndAchievements(t *testing.T) {
	userID := uuid.New()
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	service := completeFakeGameUseCase()
	service.getDailyFunc = func(context.Context, uuid.UUID, time.Time) (usecase.DailySummary, error) {
		return usecase.DailySummary{Progress: model.DailyProgress{
			Date: now, ActionsCount: 7, CompletedTasks: 2, EarnedXP: 60,
			LevelBefore: 1, LevelAfter: 2, UnlockedRoomItems: []string{"DESK", "LAMP"},
			StoryStageBefore: 0, StoryStageAfter: 2, WeeklyScoreDelta: 200, PetMood: model.PetMoodProud,
		}, WeeklyPosition: gameIntPointer(14), Retention: usecase.RetentionOverview{
			Streak: usecase.StreakOverview{
				Current: 4, Longest: 7, LastActiveDate: gameTimePointer(now), ActiveToday: true,
				Reward: usecase.RewardOffer{Type: usecase.DefaultRewardType, Amount: 2, Source: model.RewardSourceStreak},
			},
			DailyQuest: usecase.DailyQuestOverview{
				Date: now, Code: "DAILY_CONTACT", Title: "Contact seller", Description: "Send a message",
				ActionType: model.ActionTypeMessageSent, Progress: 1, Target: 1, Status: model.DailyQuestStatusRewarded,
				Reward: usecase.RewardOffer{Type: usecase.DefaultRewardType, Amount: 8, Source: model.RewardSourceDailyQuest},
			},
			Tomorrow: usecase.TomorrowPreview{
				Date: now.AddDate(0, 0, 1), StreakAfterReturn: 5,
				StreakReward: usecase.RewardOffer{Type: usecase.DefaultRewardType, Amount: 2, Source: model.RewardSourceStreak},
				DailyQuest: usecase.DailyQuestPreview{
					Code: "DAILY_DISCOVER", Title: "Discover", Description: "View 3 ads",
					ActionType: model.ActionTypeAdViewed, Target: 3,
					Reward: usecase.RewardOffer{Type: usecase.DefaultRewardType, Amount: 5, Source: model.RewardSourceDailyQuest},
				},
			},
		}}, nil
	}
	service.getLeaderboardFunc = func(_ context.Context, gotUserID uuid.UUID, limit int, _ time.Time) (usecase.Leaderboard, error) {
		if gotUserID != userID || limit != 10 {
			t.Fatalf("leaderboard user = %s, limit = %d", gotUserID, limit)
		}
		entry := model.LeaderboardEntry{Position: 1, UserID: userID, PetName: "Авитоша", Level: 2, Score: 200}
		return usecase.Leaderboard{WeekStart: time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC), Leaders: []model.LeaderboardEntry{entry}, CurrentUser: entry}, nil
	}
	service.getAchievementsFunc = func(context.Context, uuid.UUID, time.Time) ([]model.AchievementProgress, error) {
		unlockedAt := now
		return []model.AchievementProgress{{
			Achievement: model.Achievement{Code: "FIRST_STEP", Title: "Первый шаг"}, UnlockedAt: &unlockedAt,
		}}, nil
	}
	router := newGameTestRouter(RouterDependencies{GameService: service, Now: func() time.Time { return now }})

	for _, path := range []string{
		"/api/v1/daily-summary", "/api/v1/leaderboard?period=weekly&limit=10", "/api/v1/achievements",
	} {
		request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
		request.Header.Set("X-User-ID", userID.String())
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, body = %s", path, recorder.Code, recorder.Body.String())
		}
	}
}

func TestLeaderboardRejectsUnsupportedPeriodAndLimit(t *testing.T) {
	router := newGameTestRouter(RouterDependencies{GameService: completeFakeGameUseCase()})
	for _, path := range []string{
		"/api/v1/leaderboard?period=all-time", "/api/v1/leaderboard?limit=101",
	} {
		request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
		request.Header.Set("X-User-ID", uuid.NewString())
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("GET %s status = %d, body = %s", path, recorder.Code, recorder.Body.String())
		}
	}
}

func completeFakeGameUseCase() fakeGameUseCase {
	return fakeGameUseCase{
		ensureProfileFunc: func(context.Context, uuid.UUID, time.Time) (usecase.GameProfile, error) {
			return usecase.GameProfile{}, nil
		},
		renamePetFunc: func(context.Context, uuid.UUID, string, time.Time) (usecase.GameProfile, error) {
			return usecase.GameProfile{}, nil
		},
		listTasksFunc: func(context.Context, uuid.UUID, time.Time) ([]model.TaskProgress, error) { return nil, nil },
		getTaskFunc: func(context.Context, uuid.UUID, uuid.UUID, time.Time) (model.TaskProgress, error) {
			return model.TaskProgress{}, nil
		},
		getRoomFunc: func(context.Context, uuid.UUID, time.Time) ([]model.RoomItemProgress, error) { return nil, nil },
		getStoryFunc: func(context.Context, uuid.UUID, time.Time) (model.StorySnapshot, error) {
			return model.StorySnapshot{}, nil
		},
		processActionFunc: func(context.Context, usecase.ProcessActionCommand) (usecase.ProcessActionResult, error) {
			return usecase.ProcessActionResult{}, nil
		},
		getDailyFunc: func(context.Context, uuid.UUID, time.Time) (usecase.DailySummary, error) {
			return usecase.DailySummary{}, nil
		},
		getLeaderboardFunc: func(context.Context, uuid.UUID, int, time.Time) (usecase.Leaderboard, error) {
			return usecase.Leaderboard{}, nil
		},
		getAchievementsFunc: func(context.Context, uuid.UUID, time.Time) ([]model.AchievementProgress, error) {
			return nil, nil
		},
		getRewardBalancesFunc: func(context.Context, uuid.UUID, time.Time) ([]model.RewardBalance, error) {
			return nil, nil
		},
		getRewardWalletFunc: func(context.Context, uuid.UUID, time.Time) (usecase.RewardWallet, error) {
			return usecase.RewardWallet{}, nil
		},
	}
}

func newGameTestRouter(overrides RouterDependencies) http.Handler {
	if overrides.Logger == nil {
		overrides.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if overrides.DB == nil {
		overrides.DB = fakeDatabasePinger{}
	}
	if overrides.FrontendOrigin == "" {
		overrides.FrontendOrigin = "http://localhost:3000"
	}
	if overrides.RefreshTokenTTL == 0 {
		overrides.RefreshTokenTTL = 30 * 24 * time.Hour
	}
	return NewRouter(overrides)
}

func gameIntPointer(value int) *int {
	return &value
}

func gameTimePointer(value time.Time) *time.Time {
	return &value
}

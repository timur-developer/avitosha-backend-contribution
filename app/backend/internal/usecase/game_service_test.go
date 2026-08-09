package usecase

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
)

type gameTestTxManager struct {
	inside bool
}

func (manager *gameTestTxManager) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	if manager.inside {
		return fn(ctx)
	}
	manager.inside = true
	defer func() { manager.inside = false }()
	return fn(ctx)
}

type gameTestPublisher struct {
	txManager *gameTestTxManager
	users     []uuid.UUID
	batches   [][]model.DomainEvent
	insideTx  bool
}

func (publisher *gameTestPublisher) Publish(userID uuid.UUID, events []model.DomainEvent) {
	publisher.insideTx = publisher.insideTx || publisher.txManager.inside
	publisher.users = append(publisher.users, userID)
	publisher.batches = append(publisher.batches, append([]model.DomainEvent(nil), events...))
}

type gameTestRepository struct {
	GameRepository
	pet                 model.Pet
	story               model.UserStoryProgress
	tasksByStage        map[int]model.Task
	userTasks           map[uuid.UUID]model.UserTask
	actions             map[uuid.UUID]model.UserAction
	roomItems           map[string]model.UserRoomItem
	weekly              model.WeeklyProgress
	daily               model.DailyProgress
	scores              model.ActivityScores
	achievements        map[string]model.UserAchievement
	events              []model.DomainEvent
	balances            map[string]model.RewardBalance
	rewarded            map[string]struct{}
	streak              model.UserStreak
	dailyQuest          model.DailyQuestProgress
	catalog             []model.RewardCatalogItem
	templates           []model.DailyQuestTemplate
	questReadsForUpdate int
}

func newGameTestRepository(userID uuid.UUID) *gameTestRepository {
	category := "FURNITURE"
	avitoBonus := DefaultRewardType
	storyCode := FirstRoomStoryCode
	stageOne := 1
	stageTwo := 2
	stageThree := 3
	stageFour := 4
	stageFive := 5
	desk := "DESK"
	lamp := "LAMP"
	chair := "CHAIR"
	plant := "PLANT"
	poster := "POSTER"
	now := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	return &gameTestRepository{
		tasksByStage: map[int]model.Task{
			1: {
				ID: uuid.New(), Code: "VIEW_FURNITURE_ADS", Title: "Стол",
				ActionType: model.ActionTypeAdViewed, Category: &category,
				TargetValue: 5, XPReward: 30, AvitoRewardType: &avitoBonus, AvitoRewardAmount: 10,
				RoomItemCode: &desk,
				StoryCode:    &storyCode, StoryStage: &stageOne, IsActive: true,
			},
			2: {
				ID: uuid.New(), Code: "FAVORITE_FURNITURE_AD", Title: "Лампа",
				ActionType: model.ActionTypeAdFavorited, Category: &category,
				TargetValue: 1, XPReward: 30, AvitoRewardType: &avitoBonus, AvitoRewardAmount: 10,
				RoomItemCode: &lamp,
				StoryCode:    &storyCode, StoryStage: &stageTwo, IsActive: true,
			},
			3: {
				ID: uuid.New(), Code: "MESSAGE_SELLER", Title: "Кресло",
				ActionType: model.ActionTypeMessageSent, TargetValue: 1, XPReward: 40,
				AvitoRewardType: &avitoBonus, AvitoRewardAmount: 15,
				RoomItemCode: &chair, StoryCode: &storyCode, StoryStage: &stageThree, IsActive: true,
			},
			4: {
				ID: uuid.New(), Code: "CREATE_FIRST_AD", Title: "Растение",
				ActionType: model.ActionTypeAdCreated, TargetValue: 1, XPReward: 50,
				AvitoRewardType: &avitoBonus, AvitoRewardAmount: 20,
				RoomItemCode: &plant, StoryCode: &storyCode, StoryStage: &stageFour, IsActive: true,
			},
			5: {
				ID: uuid.New(), Code: "USE_DELIVERY", Title: "Постер",
				ActionType: model.ActionTypeDeliveryUsed, TargetValue: 1, XPReward: 80,
				AvitoRewardType: &avitoBonus, AvitoRewardAmount: 25,
				RoomItemCode: &poster, StoryCode: &storyCode, StoryStage: &stageFive, IsActive: true,
			},
		},
		userTasks: make(map[uuid.UUID]model.UserTask), actions: make(map[uuid.UUID]model.UserAction),
		roomItems: make(map[string]model.UserRoomItem), achievements: make(map[string]model.UserAchievement),
		balances: make(map[string]model.RewardBalance), rewarded: make(map[string]struct{}),
		scores: model.ActivityScores{UserID: userID},
		catalog: []model.RewardCatalogItem{
			{Code: "FREE_DELIVERY_LIGHT", RewardType: avitoBonus, PerkType: "DELIVERY", Threshold: 20, SortOrder: 1},
			{Code: "CATEGORY_DISCOUNT_HOME", RewardType: avitoBonus, PerkType: "CATEGORY_DISCOUNT", Threshold: 45, SortOrder: 2},
			{Code: "PROMO_BOOST", RewardType: avitoBonus, PerkType: "PROMOTION", Threshold: 75, SortOrder: 3},
			{Code: "AUTOTEKA_CHECK", RewardType: avitoBonus, PerkType: "AUTOTEKA", Threshold: 110, SortOrder: 4},
			{Code: "SELLER_LIMIT_PACK", RewardType: avitoBonus, PerkType: "LIMIT_PACK", Threshold: 160, SortOrder: 5},
		},
		templates: []model.DailyQuestTemplate{
			{Code: "DAILY_DISCOVER", Title: "Discover", ActionType: model.ActionTypeAdViewed, TargetValue: 3, RewardType: avitoBonus, RewardAmount: 5, SortOrder: 1, IsActive: true, CreatedAt: now, UpdatedAt: now},
			{Code: "DAILY_FAVORITE", Title: "Favorite", ActionType: model.ActionTypeAdFavorited, TargetValue: 1, RewardType: avitoBonus, RewardAmount: 6, SortOrder: 2, IsActive: true, CreatedAt: now, UpdatedAt: now},
			{Code: "DAILY_CONTACT", Title: "Contact", ActionType: model.ActionTypeMessageSent, TargetValue: 1, RewardType: avitoBonus, RewardAmount: 8, SortOrder: 3, IsActive: true, CreatedAt: now, UpdatedAt: now},
			{Code: "DAILY_SELLER_STEP", Title: "Seller step", ActionType: model.ActionTypeAdCreated, TargetValue: 1, RewardType: avitoBonus, RewardAmount: 10, SortOrder: 4, IsActive: true, CreatedAt: now, UpdatedAt: now},
			{Code: "DAILY_DELIVERY", Title: "Delivery", ActionType: model.ActionTypeDeliveryUsed, TargetValue: 1, RewardType: avitoBonus, RewardAmount: 12, SortOrder: 5, IsActive: true, CreatedAt: now, UpdatedAt: now},
		},
	}
}

func (repository *gameTestRepository) EnsureRewardBalance(
	_ context.Context,
	userID uuid.UUID,
	rewardType string,
	now time.Time,
) (model.RewardBalance, error) {
	balance, exists := repository.balances[rewardType]
	if !exists {
		balance = model.RewardBalance{
			UserID: userID, RewardType: rewardType, CreatedAt: now, UpdatedAt: now,
		}
		repository.balances[rewardType] = balance
	}
	return balance, nil
}

func (repository *gameTestRepository) CreditReward(
	_ context.Context,
	credit model.RewardCredit,
) (model.RewardBalance, bool, error) {
	key := credit.ActionID.String() + ":" + string(credit.SourceKind) + ":" + credit.SourceRef + ":" + credit.RewardType
	if _, exists := repository.rewarded[key]; exists {
		return repository.balances[credit.RewardType], false, nil
	}
	repository.rewarded[key] = struct{}{}
	balance := repository.balances[credit.RewardType]
	balance.Balance += int64(credit.Amount)
	balance.EarnedTotal += int64(credit.Amount)
	balance.UpdatedAt = credit.CreatedAt
	repository.balances[credit.RewardType] = balance
	return balance, true, nil
}

func (repository *gameTestRepository) ListRewardBalances(
	_ context.Context,
	_ uuid.UUID,
) ([]model.RewardBalance, error) {
	balances := make([]model.RewardBalance, 0, len(repository.balances))
	for _, balance := range repository.balances {
		balances = append(balances, balance)
	}
	return balances, nil
}

func (repository *gameTestRepository) ListRewardCatalog(_ context.Context) ([]model.RewardCatalogItem, error) {
	return append([]model.RewardCatalogItem(nil), repository.catalog...), nil
}

func (repository *gameTestRepository) GetOrCreateUserStreak(
	_ context.Context,
	candidate model.UserStreak,
) (model.UserStreak, error) {
	if repository.streak.UserID == uuid.Nil {
		repository.streak = candidate
	}
	return repository.streak, nil
}

func (repository *gameTestRepository) UpdateUserStreak(_ context.Context, streak model.UserStreak) error {
	repository.streak = streak
	return nil
}

func (repository *gameTestRepository) ListActiveDailyQuestTemplates(_ context.Context) ([]model.DailyQuestTemplate, error) {
	return append([]model.DailyQuestTemplate(nil), repository.templates...), nil
}

func (repository *gameTestRepository) ExpireDailyQuestsBefore(
	_ context.Context,
	_ uuid.UUID,
	date time.Time,
	_ time.Time,
) error {
	if repository.dailyQuest.Quest.ID != uuid.Nil && repository.dailyQuest.Quest.QuestDate.Before(date) &&
		(repository.dailyQuest.Quest.Status == model.DailyQuestStatusActive || repository.dailyQuest.Quest.Status == model.DailyQuestStatusCompleted) {
		repository.dailyQuest.Quest.Status = model.DailyQuestStatusExpired
	}
	return nil
}

func (repository *gameTestRepository) AssignDailyQuest(
	_ context.Context,
	quest model.UserDailyQuest,
) (model.UserDailyQuest, error) {
	if repository.dailyQuest.Quest.ID == uuid.Nil || !repository.dailyQuest.Quest.QuestDate.Equal(quest.QuestDate) {
		repository.dailyQuest = model.DailyQuestProgress{
			Template: repository.questTemplateByCode(quest.TemplateCode),
			Quest:    quest,
		}
	}
	return repository.dailyQuest.Quest, nil
}

func (repository *gameTestRepository) GetDailyQuestProgress(
	_ context.Context,
	_ uuid.UUID,
	date time.Time,
) (model.DailyQuestProgress, error) {
	if repository.dailyQuest.Quest.ID == uuid.Nil || !repository.dailyQuest.Quest.QuestDate.Equal(date) {
		return model.DailyQuestProgress{}, ErrDailyQuestNotFound
	}
	return repository.dailyQuest, nil
}

func (repository *gameTestRepository) GetDailyQuestProgressForUpdate(
	_ context.Context,
	_ uuid.UUID,
	date time.Time,
) (model.DailyQuestProgress, error) {
	repository.questReadsForUpdate++
	return repository.GetDailyQuestProgress(context.Background(), uuid.Nil, date)
}

func (repository *gameTestRepository) UpdateDailyQuest(_ context.Context, quest model.UserDailyQuest) error {
	repository.dailyQuest.Quest = quest
	return nil
}

func (repository *gameTestRepository) GetOrCreateGamePet(_ context.Context, candidate model.Pet) (model.Pet, error) {
	if repository.pet.ID == uuid.Nil {
		repository.pet = candidate
	}
	return repository.pet, nil
}

func (repository *gameTestRepository) UpdateGamePet(_ context.Context, pet model.Pet) error {
	repository.pet = pet
	return nil
}

func TestRenamePetPersistsNormalizedName(t *testing.T) {
	userID := uuid.New()
	repository := newGameTestRepository(userID)
	service := NewGameService(GameServiceDependencies{
		Repository: repository, TxManager: &gameTestTxManager{}, IDGenerator: uuid.New,
	})
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)

	profile, err := service.RenamePet(context.Background(), userID, "  БЕЛЫЙ   БИМ ", now)
	if err != nil {
		t.Fatalf("RenamePet() error = %v", err)
	}
	if profile.Pet.Name != "Белый Бим" || repository.pet.Name != "Белый Бим" {
		t.Fatalf("profile name = %q, stored name = %q", profile.Pet.Name, repository.pet.Name)
	}
	if !repository.pet.UpdatedAt.Equal(now) {
		t.Fatalf("updated at = %v, want %v", repository.pet.UpdatedAt, now)
	}
}

func (repository *gameTestRepository) GetOrCreateStoryProgress(
	_ context.Context,
	candidate model.UserStoryProgress,
) (model.UserStoryProgress, error) {
	if repository.story.ID == uuid.Nil {
		repository.story = candidate
	}
	return repository.story, nil
}

func (repository *gameTestRepository) GetStorySnapshot(
	_ context.Context,
	_ uuid.UUID,
	_ string,
) (model.StorySnapshot, error) {
	snapshot := model.StorySnapshot{
		Story:    model.Story{Code: FirstRoomStoryCode, Title: "Первая комната", TotalStages: 5, IsActive: true},
		Progress: repository.story,
	}
	if next, ok := repository.tasksByStage[repository.story.CurrentStage+1]; ok {
		snapshot.NextTask = &next
	}
	return snapshot, nil
}

func (repository *gameTestRepository) UpdateStoryProgress(_ context.Context, progress model.UserStoryProgress) error {
	repository.story = progress
	return nil
}

func (repository *gameTestRepository) EnsureInitialRoomItem(_ context.Context, item model.UserRoomItem) error {
	if _, ok := repository.roomItems[item.ItemCode]; !ok {
		repository.roomItems[item.ItemCode] = item
	}
	return nil
}

func (repository *gameTestRepository) UnlockRoomItem(_ context.Context, item model.UserRoomItem) (bool, error) {
	if _, ok := repository.roomItems[item.ItemCode]; ok {
		return false, nil
	}
	repository.roomItems[item.ItemCode] = item
	return true, nil
}

func (repository *gameTestRepository) AssignStoryTask(
	_ context.Context,
	userID uuid.UUID,
	_ string,
	stage int,
	now time.Time,
) (model.TaskProgress, error) {
	task, ok := repository.tasksByStage[stage]
	if !ok {
		return model.TaskProgress{}, ErrTaskNotFound
	}
	progress, exists := repository.userTasks[task.ID]
	if !exists {
		progress = model.UserTask{
			ID: uuid.New(), UserID: userID, TaskID: task.ID, TargetValue: task.TargetValue,
			Status: model.TaskStatusActive, AssignedAt: now, CreatedAt: now, UpdatedAt: now,
		}
		repository.userTasks[task.ID] = progress
	}
	return model.TaskProgress{Task: task, Progress: progress}, nil
}

func (repository *gameTestRepository) FindMatchingActiveTasksForUpdate(
	_ context.Context,
	_ uuid.UUID,
	actionType model.ActionType,
	category *string,
) ([]model.TaskProgress, error) {
	result := make([]model.TaskProgress, 0, 1)
	for taskID, progress := range repository.userTasks {
		task := repository.taskByID(taskID)
		if progress.Status != model.TaskStatusActive || task.ActionType != actionType {
			continue
		}
		if task.Category != nil && !equalStringPointers(task.Category, category) {
			continue
		}
		result = append(result, model.TaskProgress{Task: task, Progress: progress})
	}
	return result, nil
}

func (repository *gameTestRepository) UpdateTaskProgress(_ context.Context, progress model.UserTask) error {
	repository.userTasks[progress.TaskID] = progress
	return nil
}

func (repository *gameTestRepository) InsertAction(
	_ context.Context,
	candidate model.UserAction,
) (model.UserAction, bool, error) {
	if existing, ok := repository.actions[candidate.EventID]; ok {
		return existing, false, nil
	}
	repository.actions[candidate.EventID] = candidate
	return candidate, true, nil
}

func (repository *gameTestRepository) CompleteAction(
	_ context.Context,
	actionID uuid.UUID,
	processedAt time.Time,
	events []model.DomainEvent,
) error {
	for eventID, action := range repository.actions {
		if action.ID != actionID {
			continue
		}
		action.ProcessedAt = &processedAt
		action.ResultEvents, _ = json.Marshal(events)
		repository.actions[eventID] = action
		return nil
	}
	return ErrActionNotFound
}

func (repository *gameTestRepository) InsertDomainEvents(_ context.Context, events []model.DomainEvent) error {
	repository.events = append(repository.events, events...)
	return nil
}

func (repository *gameTestRepository) UpdateWeeklyProgress(
	_ context.Context,
	userID uuid.UUID,
	weekStart time.Time,
	delta WeeklyProgressDelta,
	now time.Time,
) (model.WeeklyProgress, error) {
	repository.weekly.UserID = userID
	repository.weekly.WeekStart = weekStart
	repository.weekly.EarnedXP += delta.EarnedXP
	repository.weekly.CompletedTasks += delta.CompletedTasks
	repository.weekly.CompletedStages += delta.CompletedStages
	repository.weekly.Score += WeeklyScore(delta)
	repository.weekly.UpdatedAt = now
	return repository.weekly, nil
}

func (repository *gameTestRepository) UpdateDailyProgress(
	_ context.Context,
	userID uuid.UUID,
	date time.Time,
	delta DailyProgressDelta,
	_ time.Time,
) (model.DailyProgress, error) {
	if repository.daily.ActionsCount == 0 {
		repository.daily.LevelBefore = delta.LevelBefore
		repository.daily.StoryStageBefore = delta.StoryStageBefore
	}
	repository.daily.UserID = userID
	repository.daily.Date = date
	repository.daily.ActionsCount += delta.ActionsCount
	repository.daily.CompletedTasks += delta.CompletedTasks
	repository.daily.EarnedXP += delta.EarnedXP
	repository.daily.LevelAfter = delta.LevelAfter
	repository.daily.UnlockedRoomItems = append(repository.daily.UnlockedRoomItems, delta.UnlockedRoomItems...)
	repository.daily.StoryStageAfter = delta.StoryStageAfter
	repository.daily.WeeklyScoreDelta += delta.WeeklyScoreDelta
	repository.daily.PetMood = delta.PetMood
	return repository.daily, nil
}

func (repository *gameTestRepository) GetDailyProgress(
	_ context.Context,
	_ uuid.UUID,
	_ time.Time,
) (model.DailyProgress, error) {
	if repository.daily.ActionsCount == 0 {
		return model.DailyProgress{}, ErrDailyProgressNotFound
	}
	return repository.daily, nil
}

func (repository *gameTestRepository) ListWeeklyLeaders(
	_ context.Context,
	_ time.Time,
	limit int,
) ([]model.LeaderboardEntry, error) {
	entry, err := repository.GetWeeklyPosition(context.Background(), repository.pet.UserID, repository.weekly.WeekStart)
	if err != nil || limit == 0 {
		return nil, err
	}
	return []model.LeaderboardEntry{entry}, nil
}

func (repository *gameTestRepository) GetWeeklyPosition(
	_ context.Context,
	userID uuid.UUID,
	_ time.Time,
) (model.LeaderboardEntry, error) {
	if repository.weekly.UserID == uuid.Nil {
		return model.LeaderboardEntry{}, ErrLeaderboardEntryNotFound
	}
	return model.LeaderboardEntry{
		Position: 1, UserID: userID, PetName: repository.pet.Name, Level: repository.pet.Level,
		Score: repository.weekly.Score, CompletedTasks: repository.weekly.CompletedTasks,
	}, nil
}

func (repository *gameTestRepository) UpdateActivityScores(
	_ context.Context,
	_ uuid.UUID,
	delta ActivityScoreDelta,
	now time.Time,
) (model.ActivityScores, error) {
	repository.scores.BuyerScore += delta.Buyer
	repository.scores.SellerScore += delta.Seller
	repository.scores.AutoScore += delta.Auto
	repository.scores.TravelScore += delta.Travel
	repository.scores.RealEstateScore += delta.RealEstate
	repository.scores.ServicesScore += delta.Services
	repository.scores.UpdatedAt = now
	return repository.scores, nil
}

func (repository *gameTestRepository) UnlockAchievements(
	_ context.Context,
	userID uuid.UUID,
	codes []string,
	now time.Time,
) ([]model.UserAchievement, error) {
	result := make([]model.UserAchievement, 0, len(codes))
	for _, code := range codes {
		if _, exists := repository.achievements[code]; exists {
			continue
		}
		achievement := model.UserAchievement{
			ID: uuid.New(), UserID: userID, AchievementCode: code, UnlockedAt: now,
		}
		repository.achievements[code] = achievement
		result = append(result, achievement)
	}
	return result, nil
}

func (repository *gameTestRepository) ListAchievements(
	_ context.Context,
	_ uuid.UUID,
) ([]model.AchievementProgress, error) {
	result := make([]model.AchievementProgress, 0, len(repository.achievements))
	for code, unlocked := range repository.achievements {
		unlockedAt := unlocked.UnlockedAt
		result = append(result, model.AchievementProgress{
			Achievement: model.Achievement{Code: code, Title: code}, UnlockedAt: &unlockedAt,
		})
	}
	return result, nil
}

func (repository *gameTestRepository) taskByID(id uuid.UUID) model.Task {
	for _, task := range repository.tasksByStage {
		if task.ID == id {
			return task
		}
	}
	return model.Task{}
}

func (repository *gameTestRepository) questTemplateByCode(code string) model.DailyQuestTemplate {
	for _, template := range repository.templates {
		if template.Code == code {
			return template
		}
	}
	return model.DailyQuestTemplate{}
}

func TestProcessActionCompletesFirstRoomStageAndIsIdempotent(t *testing.T) {
	userID := mustGameUUID("00000000-0000-0000-0000-000000000001")
	repository := newGameTestRepository(userID)
	txManager := &gameTestTxManager{}
	publisher := &gameTestPublisher{txManager: txManager}
	service := NewGameService(GameServiceDependencies{
		Repository: repository, TxManager: txManager, IDGenerator: uuid.New, Publisher: publisher,
	})
	now := time.Date(2026, 8, 5, 12, 0, 0, 123456789, time.UTC)
	category := "furniture"

	var lastCommand ProcessActionCommand
	for index := 0; index < 5; index++ {
		lastCommand = ProcessActionCommand{
			UserID: userID, EventID: uuid.New(), ActionType: model.ActionTypeAdViewed,
			EntityID: gameStringPointer("advert-" + string(rune('1'+index))), Category: &category,
			Metadata: json.RawMessage(`{}`), OccurredAt: now.Add(time.Duration(index) * time.Minute),
			Now: now.Add(time.Duration(index) * time.Minute),
		}
		result, err := service.ProcessAction(context.Background(), lastCommand)
		if err != nil {
			t.Fatalf("ProcessAction(%d) error = %v", index+1, err)
		}
		if result.Duplicate {
			t.Fatalf("ProcessAction(%d) unexpectedly duplicate", index+1)
		}
	}

	firstTask := repository.tasksByStage[1]
	progress := repository.userTasks[firstTask.ID]
	if progress.Status != model.TaskStatusRewarded || progress.Progress != 5 || progress.RewardedAt == nil {
		t.Fatalf("task progress = %+v", progress)
	}
	if repository.pet.GrowthXP != 30 || repository.pet.Level != 1 || repository.pet.Mood != model.PetMoodProud {
		t.Fatalf("pet = %+v", repository.pet)
	}
	if repository.story.CurrentStage != 1 || repository.story.Status != model.StoryStatusActive {
		t.Fatalf("story = %+v", repository.story)
	}
	if _, ok := repository.roomItems["DESK"]; !ok {
		t.Fatal("DESK was not unlocked")
	}
	if repository.weekly.Score != 100 || repository.weekly.EarnedXP != 30 {
		t.Fatalf("weekly progress = %+v", repository.weekly)
	}
	if repository.daily.ActionsCount != 5 || repository.daily.CompletedTasks != 1 {
		t.Fatalf("daily progress = %+v", repository.daily)
	}
	if repository.questReadsForUpdate == 0 {
		t.Fatal("daily quest was not read with a lock in the action flow")
	}
	if balance := repository.balances[DefaultRewardType]; balance.Balance != 12 || balance.EarnedTotal != 12 {
		t.Fatalf("reward balance = %+v", balance)
	}
	rewardEventFound := false
	for _, event := range repository.events {
		if event.Type == model.DomainEventAvitoRewardEarned {
			rewardEventFound = true
			break
		}
	}
	if !rewardEventFound {
		t.Fatal("AVITO_REWARD_EARNED event was not created")
	}
	if publisher.insideTx || len(publisher.batches) != 5 {
		t.Fatalf("publisher insideTx = %v, batches = %d", publisher.insideTx, len(publisher.batches))
	}

	duplicate, err := service.ProcessAction(context.Background(), lastCommand)
	if err != nil || !duplicate.Duplicate {
		t.Fatalf("duplicate result = %+v, error = %v", duplicate, err)
	}
	if repository.pet.GrowthXP != 30 || repository.weekly.Score != 100 || repository.daily.ActionsCount != 5 ||
		repository.balances[DefaultRewardType].Balance != 12 {
		t.Fatal("duplicate action changed rewards or aggregates")
	}
	if len(publisher.batches) != 5 {
		t.Fatal("duplicate action was published")
	}
}

func TestEndToEndCompletesFirstRoom(t *testing.T) {
	userID := mustGameUUID("00000000-0000-0000-0000-000000000001")
	repository := newGameTestRepository(userID)
	txManager := &gameTestTxManager{}
	publisher := &gameTestPublisher{txManager: txManager}
	service := NewGameService(GameServiceDependencies{
		Repository: repository, TxManager: txManager, IDGenerator: uuid.New, Publisher: publisher,
	})
	now := time.Date(2026, 8, 5, 9, 0, 0, 0, time.UTC)
	furniture := "FURNITURE"
	actions := []struct {
		actionType model.ActionType
		category   *string
	}{
		{model.ActionTypeAdViewed, &furniture},
		{model.ActionTypeAdViewed, &furniture},
		{model.ActionTypeAdViewed, &furniture},
		{model.ActionTypeAdViewed, &furniture},
		{model.ActionTypeAdViewed, &furniture},
		{model.ActionTypeAdFavorited, &furniture},
		{model.ActionTypeMessageSent, nil},
		{model.ActionTypeAdCreated, nil},
		{model.ActionTypeDeliveryUsed, nil},
	}

	var last ProcessActionCommand
	for index, action := range actions {
		last = ProcessActionCommand{
			UserID: userID, EventID: uuid.New(), ActionType: action.actionType,
			EntityID: gameStringPointer("entity-" + string(rune('a'+index))), Category: action.category,
			Metadata: json.RawMessage(`{}`), OccurredAt: now.Add(time.Duration(index) * time.Minute),
			Now: now.Add(time.Duration(index) * time.Minute),
		}
		if _, err := service.ProcessAction(context.Background(), last); err != nil {
			t.Fatalf("action %d (%s): %v", index+1, action.actionType, err)
		}
	}

	if repository.story.Status != model.StoryStatusCompleted || repository.story.CurrentStage != 5 ||
		repository.story.CompletedAt == nil {
		t.Fatalf("story = %+v", repository.story)
	}
	if repository.pet.GrowthXP != 230 || repository.pet.Level != 2 || repository.pet.Mood != model.PetMoodProud {
		t.Fatalf("pet = %+v", repository.pet)
	}
	if repository.pet.Character == nil || *repository.pet.Character != model.PetCharacterExplorer {
		t.Fatalf("character = %v, want EXPLORER", repository.pet.Character)
	}
	for _, itemCode := range []string{"BOX", "DESK", "LAMP", "CHAIR", "PLANT", "POSTER"} {
		if _, ok := repository.roomItems[itemCode]; !ok {
			t.Errorf("room item %s is missing", itemCode)
		}
	}
	if repository.weekly.EarnedXP != 230 || repository.weekly.CompletedTasks != 5 ||
		repository.weekly.CompletedStages != 5 || repository.weekly.Score != 580 {
		t.Fatalf("weekly = %+v", repository.weekly)
	}
	if repository.daily.ActionsCount != 9 || repository.daily.CompletedTasks != 5 ||
		repository.daily.EarnedXP != 230 || repository.daily.StoryStageAfter != 5 {
		t.Fatalf("daily = %+v", repository.daily)
	}
	if balance := repository.balances[DefaultRewardType]; balance.Balance != 92 || balance.EarnedTotal != 92 {
		t.Fatalf("reward balance = %+v, want 92", balance)
	}
	for _, code := range []string{"FIRST_STEP", "HOUSEWARMING", "EXPLORER", "IN_TOUCH", "FIRST_AD", "ROOM_COMPLETE"} {
		if _, ok := repository.achievements[code]; !ok {
			t.Errorf("achievement %s is missing", code)
		}
	}
	summary, err := service.GetDailySummary(context.Background(), userID, last.Now)
	if err != nil || summary.Progress.EarnedXP != 230 || summary.WeeklyPosition == nil || *summary.WeeklyPosition != 1 ||
		summary.Retention.DailyQuest.Code != "DAILY_SELLER_STEP" || summary.Retention.DailyQuest.Status != model.DailyQuestStatusRewarded {
		t.Fatalf("daily summary = %+v, error = %v", summary, err)
	}
	leaderboard, err := service.GetLeaderboard(context.Background(), userID, 10, last.Now)
	if err != nil || leaderboard.CurrentUser.Score != 580 || len(leaderboard.Leaders) != 1 {
		t.Fatalf("leaderboard = %+v, error = %v", leaderboard, err)
	}
	achievements, err := service.GetAchievements(context.Background(), userID, last.Now)
	if err != nil || len(achievements) != 6 {
		t.Fatalf("achievements = %+v, error = %v", achievements, err)
	}
	balances, err := service.GetRewardBalances(context.Background(), userID, last.Now)
	if err != nil || len(balances) != 1 || balances[0].Balance != 92 {
		t.Fatalf("reward balances = %+v, error = %v", balances, err)
	}
	wallet, err := service.GetRewardWallet(context.Background(), userID, last.Now)
	if err != nil || wallet.Balance.Balance != 92 || wallet.NextGoal == nil || wallet.NextGoal.Item.Code != "AUTOTEKA_CHECK" {
		t.Fatalf("reward wallet = %+v, error = %v", wallet, err)
	}

	duplicate, err := service.ProcessAction(context.Background(), last)
	if err != nil || !duplicate.Duplicate {
		t.Fatalf("duplicate = %+v, error = %v", duplicate, err)
	}
	if repository.weekly.Score != 580 || repository.pet.GrowthXP != 230 || repository.daily.ActionsCount != 9 ||
		repository.balances[DefaultRewardType].Balance != 92 {
		t.Fatal("duplicate final action changed completed room state")
	}
}

func TestGetDailySummaryNormalizesBrokenStreakWithoutWaitingForNewAction(t *testing.T) {
	userID := mustGameUUID("00000000-0000-0000-0000-000000000009")
	repository := newGameTestRepository(userID)
	service := NewGameService(GameServiceDependencies{
		Repository: repository, TxManager: &gameTestTxManager{}, IDGenerator: uuid.New,
	})
	now := time.Date(2026, 8, 6, 9, 0, 0, 0, time.UTC)
	lastActive := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	repository.streak = model.UserStreak{
		UserID: userID, CurrentStreak: 7, LongestStreak: 7, LastActiveDate: &lastActive,
		CreatedAt: lastActive, UpdatedAt: lastActive,
	}

	summary, err := service.GetDailySummary(context.Background(), userID, now)
	if err != nil {
		t.Fatalf("GetDailySummary() error = %v", err)
	}
	if summary.Retention.Streak.Current != 0 || summary.Retention.Streak.ActiveToday ||
		summary.Retention.Streak.Reward.Amount != 0 {
		t.Fatalf("streak = %+v", summary.Retention.Streak)
	}
	if summary.Retention.Tomorrow.StreakAfterReturn != 1 || summary.Retention.Tomorrow.StreakReward.Amount != 2 {
		t.Fatalf("tomorrow preview = %+v", summary.Retention.Tomorrow)
	}
}

func gameStringPointer(value string) *string {
	return &value
}

func mustGameUUID(value string) uuid.UUID {
	id, err := uuid.Parse(value)
	if err != nil {
		panic(err)
	}
	return id
}

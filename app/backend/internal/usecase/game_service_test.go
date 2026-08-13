package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
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
	dailyQuests         []model.DailyQuestProgress
	dailyGoal           model.UserDailyGoal
	catalog             []model.RewardCatalogItem
	templates           []model.DailyQuestTemplate
	questReadsForUpdate int
}

type gameTestAdviceGenerator struct {
	input AdviceGenerationInput
	text  string
	err   error
}

func (generator *gameTestAdviceGenerator) Generate(_ context.Context, input AdviceGenerationInput) (string, error) {
	generator.input = input
	return generator.text, generator.err
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
			{Code: "DAILY_DISCOVER", Title: "Discover", ActionType: model.ActionTypeAdViewed, Role: model.DailyQuestRoleBuyer, TargetValue: 3, XPReward: 8, RewardType: avitoBonus, SortOrder: 1, IsActive: true, CreatedAt: now, UpdatedAt: now},
			{Code: "DAILY_FAVORITE", Title: "Favorite", ActionType: model.ActionTypeAdFavorited, Role: model.DailyQuestRoleBuyer, TargetValue: 1, XPReward: 10, RewardType: avitoBonus, SortOrder: 2, IsActive: true, CreatedAt: now, UpdatedAt: now},
			{Code: "DAILY_SELLER_STEP", Title: "Seller step", ActionType: model.ActionTypeAdCreated, Role: model.DailyQuestRoleSeller, TargetValue: 1, XPReward: 15, RewardType: avitoBonus, SortOrder: 3, IsActive: true, CreatedAt: now, UpdatedAt: now},
			{Code: "DAILY_IMPROVE", Title: "Improve", ActionType: model.ActionTypeListingImproved, Role: model.DailyQuestRoleSeller, TargetValue: 1, XPReward: 12, RewardType: avitoBonus, SortOrder: 4, IsActive: true, CreatedAt: now, UpdatedAt: now},
			{Code: "DAILY_DELIVERY", Title: "Delivery", ActionType: model.ActionTypeDeliveryUsed, Role: model.DailyQuestRoleUniversal, TargetValue: 1, XPReward: 10, RewardType: avitoBonus, SortOrder: 5, IsActive: true, CreatedAt: now, UpdatedAt: now},
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
	for index := range repository.dailyQuests {
		if repository.dailyQuests[index].Quest.QuestDate.Before(date) && repository.dailyQuests[index].Quest.Status == model.DailyQuestStatusActive {
			repository.dailyQuests[index].Quest.Status = model.DailyQuestStatusExpired
		}
	}
	return nil
}

func (repository *gameTestRepository) AssignDailyQuest(
	_ context.Context,
	quest model.UserDailyQuest,
) (model.UserDailyQuest, error) {
	for _, item := range repository.dailyQuests {
		if item.Quest.QuestDate.Equal(quest.QuestDate) && item.Quest.TemplateCode == quest.TemplateCode {
			return item.Quest, nil
		}
	}
	repository.dailyQuests = append(repository.dailyQuests, model.DailyQuestProgress{
		Template: repository.questTemplateByCode(quest.TemplateCode),
		Quest:    quest,
	})
	return quest, nil
}

func (repository *gameTestRepository) ListDailyQuestProgress(
	_ context.Context,
	_ uuid.UUID,
	date time.Time,
) ([]model.DailyQuestProgress, error) {
	items := make([]model.DailyQuestProgress, 0, 5)
	for _, item := range repository.dailyQuests {
		if item.Quest.QuestDate.Equal(date) {
			items = append(items, item)
		}
	}
	return items, nil
}

func (repository *gameTestRepository) ListDailyQuestProgressForUpdate(
	_ context.Context,
	_ uuid.UUID,
	date time.Time,
) ([]model.DailyQuestProgress, error) {
	repository.questReadsForUpdate++
	return repository.ListDailyQuestProgress(context.Background(), uuid.Nil, date)
}

func (repository *gameTestRepository) UpdateDailyQuest(_ context.Context, quest model.UserDailyQuest) error {
	for index := range repository.dailyQuests {
		if repository.dailyQuests[index].Quest.ID == quest.ID {
			repository.dailyQuests[index].Quest = quest
		}
	}
	return nil
}

func (repository *gameTestRepository) GetOrCreateDailyGoal(_ context.Context, candidate model.UserDailyGoal) (model.UserDailyGoal, error) {
	if repository.dailyGoal.ID == uuid.Nil || !repository.dailyGoal.GoalDate.Equal(candidate.GoalDate) {
		repository.dailyGoal = candidate
	}
	return repository.dailyGoal, nil
}

func (repository *gameTestRepository) UpdateDailyGoal(_ context.Context, goal model.UserDailyGoal) error {
	repository.dailyGoal = goal
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

func (repository *gameTestRepository) GetTaskProgress(
	_ context.Context,
	_ uuid.UUID,
	taskID uuid.UUID,
) (model.TaskProgress, error) {
	progress, ok := repository.userTasks[taskID]
	if !ok {
		return model.TaskProgress{}, ErrTaskNotFound
	}
	return model.TaskProgress{Task: repository.taskByID(taskID), Progress: progress}, nil
}

func TestGetTaskAdviceUsesGeneratorContext(t *testing.T) {
	userID := uuid.New()
	repository := newGameTestRepository(userID)
	generator := &gameTestAdviceGenerator{text: "  Сравни фотографии и условия доставки.  "}
	service := NewGameService(GameServiceDependencies{
		Repository: repository, TxManager: &gameTestTxManager{}, IDGenerator: uuid.New, Advice: generator,
	})
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	taskID := repository.tasksByStage[1].ID

	advice, err := service.GetTaskAdvice(context.Background(), userID, taskID, now)
	if err != nil {
		t.Fatalf("GetTaskAdvice() error = %v", err)
	}
	if advice.Text != "Сравни фотографии и условия доставки." || !advice.GeneratedByAI {
		t.Fatalf("advice = %+v", advice)
	}
	if generator.input.PetName != DefaultPetName || generator.input.TaskTitle != "Стол" || generator.input.Target != 5 {
		t.Fatalf("generator input = %+v", generator.input)
	}
}

func TestGetTaskAdviceFallsBackWhenGeneratorFails(t *testing.T) {
	userID := uuid.New()
	repository := newGameTestRepository(userID)
	service := NewGameService(GameServiceDependencies{
		Repository: repository, TxManager: &gameTestTxManager{}, IDGenerator: uuid.New,
		Advice: &gameTestAdviceGenerator{err: errors.New("provider unavailable")},
	})
	taskID := repository.tasksByStage[1].ID

	advice, err := service.GetTaskAdvice(context.Background(), userID, taskID, time.Now())
	if err != nil {
		t.Fatalf("GetTaskAdvice() error = %v", err)
	}
	if advice.GeneratedByAI || advice.Text == "" || !strings.Contains(advice.Text, "5 объявлений") {
		t.Fatalf("fallback advice = %+v", advice)
	}
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

func (repository *gameTestRepository) CountUserActions(_ context.Context, userID uuid.UUID, actionType model.ActionType, category, entityID *string) (int, error) {
	count := 0
	for _, action := range repository.actions {
		if action.UserID == userID && action.ActionType == actionType && (category == nil || equalStringPointers(action.Category, category)) && (entityID == nil || equalStringPointers(action.EntityID, entityID)) { count++ }
	}
	return count, nil
}

func (repository *gameTestRepository) CountDistinctUserActionEntities(_ context.Context, userID uuid.UUID, actionType model.ActionType) (int, error) {
	entities := make(map[string]struct{})
	for _, action := range repository.actions {
		if action.UserID != userID || action.ActionType != actionType || action.EntityID == nil {
			continue
		}
		entities[*action.EntityID] = struct{}{}
	}
	return len(entities), nil
}

func (repository *gameTestRepository) CountUserActionsOnDate(_ context.Context, userID uuid.UUID, actionType model.ActionType, date time.Time, excludeEventID uuid.UUID) (int, error) {
	count := 0
	for _, action := range repository.actions {
		if action.UserID == userID && action.EventID != excludeEventID && action.ActionType == actionType && action.OccurredAt.UTC().Truncate(24*time.Hour).Equal(date.UTC().Truncate(24*time.Hour)) { count++ }
	}
	return count, nil
}

func (repository *gameTestRepository) GetProductActionRule(_ context.Context, actionType model.ActionType) (model.ProductActionRule, error) {
	return model.ProductActionRule{ActionType: actionType, XPReward: 40}, nil
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
	repository.scores.QualityScore += delta.Quality
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
	if repository.pet.GrowthXP != 38 || repository.pet.Level != 1 || repository.pet.Mood != model.PetMoodProud {
		t.Fatalf("pet = %+v", repository.pet)
	}
	if repository.story.CurrentStage != 1 || repository.story.Status != model.StoryStatusActive {
		t.Fatalf("story = %+v", repository.story)
	}
	if _, ok := repository.roomItems["DESK"]; !ok {
		t.Fatal("DESK was not unlocked")
	}
	if repository.weekly.Score != 108 || repository.weekly.EarnedXP != 38 {
		t.Fatalf("weekly progress = %+v", repository.weekly)
	}
	if repository.daily.ActionsCount != 5 || repository.daily.CompletedTasks != 1 {
		t.Fatalf("daily progress = %+v", repository.daily)
	}
	if repository.questReadsForUpdate == 0 {
		t.Fatal("daily quest was not read with a lock in the action flow")
	}
	if balance := repository.balances[DefaultRewardType]; balance.Balance != 10 || balance.EarnedTotal != 10 {
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
	if repository.pet.GrowthXP != 38 || repository.weekly.Score != 108 || repository.daily.ActionsCount != 5 ||
		repository.balances[DefaultRewardType].Balance != 10 {
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
	if repository.pet.GrowthXP != 303 || repository.pet.Level != 3 || repository.pet.Mood != model.PetMoodHappy {
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
	if repository.weekly.EarnedXP != 303 || repository.weekly.CompletedTasks != 5 ||
		repository.weekly.CompletedStages != 5 || repository.weekly.Score != 653 {
		t.Fatalf("weekly = %+v", repository.weekly)
	}
	if repository.daily.ActionsCount != 9 || repository.daily.CompletedTasks != 5 ||
		repository.daily.EarnedXP != 303 || repository.daily.StoryStageAfter != 5 {
		t.Fatalf("daily = %+v", repository.daily)
	}
	if balance := repository.balances[DefaultRewardType]; balance.Balance != 90 || balance.EarnedTotal != 90 {
		t.Fatalf("reward balance = %+v, want 90", balance)
	}
	for _, code := range []string{"FIRST_STEP", "HOUSEWARMING", "EXPLORER", "IN_TOUCH", "FIRST_AD", "ROOM_COMPLETE"} {
		if _, ok := repository.achievements[code]; !ok {
			t.Errorf("achievement %s is missing", code)
		}
	}
	summary, err := service.GetDailySummary(context.Background(), userID, last.Now)
	if err != nil || summary.Progress.EarnedXP != 303 || summary.WeeklyPosition == nil || *summary.WeeklyPosition != 1 ||
		summary.Retention.DailyGoal.Completed < 2 || summary.Retention.DailyGoal.Status != model.DailyGoalStatusRewarded {
		t.Fatalf("daily summary = %+v, error = %v", summary, err)
	}
	roleCounts := map[model.DailyQuestRole]int{}
	for _, quest := range summary.Retention.DailyGoal.Quests {
		roleCounts[quest.Role]++
	}
	if len(summary.Retention.DailyGoal.Quests) != 5 || roleCounts[model.DailyQuestRoleBuyer] != 2 ||
		roleCounts[model.DailyQuestRoleSeller] != 2 || roleCounts[model.DailyQuestRoleUniversal] != 1 {
		t.Fatalf("daily quest roles = %+v", roleCounts)
	}
	if !summary.Retention.DailyGoal.BalancedCompleted || !summary.Retention.Streak.ActiveToday {
		t.Fatalf("retention goal = %+v, streak = %+v", summary.Retention.DailyGoal, summary.Retention.Streak)
	}
	leaderboard, err := service.GetLeaderboard(context.Background(), userID, 10, last.Now)
	if err != nil || leaderboard.CurrentUser.Score != 653 || len(leaderboard.Leaders) != 1 {
		t.Fatalf("leaderboard = %+v, error = %v", leaderboard, err)
	}
	achievements, err := service.GetAchievements(context.Background(), userID, last.Now)
	if err != nil || len(achievements) != 6 {
		t.Fatalf("achievements = %+v, error = %v", achievements, err)
	}
	balances, err := service.GetRewardBalances(context.Background(), userID, last.Now)
	if err != nil || len(balances) != 1 || balances[0].Balance != 90 {
		t.Fatalf("reward balances = %+v, error = %v", balances, err)
	}
	wallet, err := service.GetRewardWallet(context.Background(), userID, last.Now)
	if err != nil || wallet.Balance.Balance != 90 || wallet.NextGoal == nil || wallet.NextGoal.Item.Code != "AUTOTEKA_CHECK" {
		t.Fatalf("reward wallet = %+v, error = %v", wallet, err)
	}

	duplicate, err := service.ProcessAction(context.Background(), last)
	if err != nil || !duplicate.Duplicate {
		t.Fatalf("duplicate = %+v, error = %v", duplicate, err)
	}
	if repository.weekly.Score != 653 || repository.pet.GrowthXP != 303 || repository.daily.ActionsCount != 9 ||
		repository.balances[DefaultRewardType].Balance != 90 {
		t.Fatal("duplicate final action changed completed room state")
	}
}

func TestHistoricalStoryCatchUpUnlocksAchievement(t *testing.T) {
	userID := mustGameUUID("00000000-0000-0000-0000-000000000001")
	repository := newGameTestRepository(userID)
	txManager := &gameTestTxManager{}
	service := NewGameService(GameServiceDependencies{
		Repository: repository, TxManager: txManager, IDGenerator: uuid.New,
	})
	now := time.Date(2026, 8, 5, 9, 0, 0, 0, time.UTC)
	furniture := "FURNITURE"

	// The listing was created before the story reaches CREATE_FIRST_AD.
	createdAt := now.Add(-time.Hour)
	createdEvent := uuid.New()
	repository.actions[uuid.New()] = model.UserAction{
		ID: uuid.New(), UserID: userID, EventID: createdEvent,
		ActionType: model.ActionTypeAdCreated, OccurredAt: createdAt,
		ProcessedAt: &createdAt, Metadata: json.RawMessage(`{"source":"marketplace.publish"}`),
	}

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
	}

	var result ProcessActionResult
	for index, action := range actions {
		result, _ = service.ProcessAction(context.Background(), ProcessActionCommand{
			UserID: userID, EventID: uuid.New(), ActionType: action.actionType,
			EntityID: gameStringPointer("historical-entity-" + string(rune('a'+index))), Category: action.category,
			Metadata: json.RawMessage(`{}`), OccurredAt: now.Add(time.Duration(index) * time.Minute),
			Now: now.Add(time.Duration(index) * time.Minute),
		})
	}

	if repository.story.CurrentStage != 4 {
		t.Fatalf("story stage = %d, want 4 after historical catch-up", repository.story.CurrentStage)
	}
	createTask := repository.tasksByStage[4]
	if progress := repository.userTasks[createTask.ID]; progress.Status != model.TaskStatusRewarded {
		t.Fatalf("CREATE_FIRST_AD progress = %+v, want rewarded", progress)
	}
	if _, ok := repository.roomItems["PLANT"]; !ok {
		t.Fatal("PLANT was not unlocked by historical catch-up")
	}
	if _, ok := repository.achievements["FIRST_AD"]; !ok {
		t.Fatal("FIRST_AD achievement was not unlocked by historical catch-up")
	}
	foundAchievementEvent := false
	for _, event := range result.Events {
		if event.Type != model.DomainEventAchievementUnlocked {
			continue
		}
		var payload struct{ Code string `json:"code"` }
		if err := json.Unmarshal(event.Payload, &payload); err == nil && payload.Code == "FIRST_AD" {
			foundAchievementEvent = true
		}
	}
	if !foundAchievementEvent {
		t.Fatal("FIRST_AD achievement event was not emitted")
	}
}

func TestMarketplacePublicationUnlocksFirstAdAchievementImmediately(t *testing.T) {
	userID := mustGameUUID("00000000-0000-0000-0000-000000000002")
	repository := newGameTestRepository(userID)
	service := NewGameService(GameServiceDependencies{
		Repository: repository, TxManager: &gameTestTxManager{}, IDGenerator: uuid.New,
	})
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)

	result, err := service.ProcessActionWithinTx(context.Background(), ProcessActionCommand{
		UserID: userID, EventID: uuid.New(), ActionType: model.ActionTypeAdCreated,
		EntityID: gameStringPointer("listing-1"), Metadata: json.RawMessage(`{"source":"marketplace.publish"}`),
		OccurredAt: now, Now: now,
	})
	if err != nil {
		t.Fatalf("trusted publication action error = %v", err)
	}
	if _, ok := repository.achievements["FIRST_AD"]; !ok {
		t.Fatal("FIRST_AD was not unlocked at publication time")
	}
	found := false
	for _, event := range result.Events {
		if event.Type != model.DomainEventAchievementUnlocked {
			continue
		}
		var payload struct{ Code string `json:"code"` }
		if json.Unmarshal(event.Payload, &payload) == nil && payload.Code == "FIRST_AD" {
			found = true
		}
	}
	if !found {
		t.Fatal("publication did not emit FIRST_AD achievement event")
	}
}

func TestMarketplaceMessageUnlocksInTouchAchievementImmediately(t *testing.T) {
	userID := uuid.New()
	repository := newGameTestRepository(userID)
	service := NewGameService(GameServiceDependencies{
		Repository: repository, TxManager: &gameTestTxManager{}, IDGenerator: uuid.New,
	})
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)

	if _, err := service.ProcessActionWithinTx(context.Background(), ProcessActionCommand{
		UserID: userID, EventID: uuid.New(), ActionType: model.ActionTypeMessageSent,
		EntityID: gameStringPointer("listing-1"), Metadata: json.RawMessage(`{"source":"marketplace.message"}`),
		OccurredAt: now, Now: now,
	}); err != nil {
		t.Fatalf("marketplace message action error = %v", err)
	}
	if _, ok := repository.achievements["IN_TOUCH"]; !ok {
		t.Fatal("IN_TOUCH was not unlocked after first marketplace message")
	}
}

func TestMarketplaceDistinctViewsUnlockExplorerAchievement(t *testing.T) {
	userID := uuid.New()
	repository := newGameTestRepository(userID)
	service := NewGameService(GameServiceDependencies{
		Repository: repository, TxManager: &gameTestTxManager{}, IDGenerator: uuid.New,
	})
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)

	entities := []string{"listing-1", "listing-2", "listing-3", "listing-1", "listing-4", "listing-5"}
	for index, entity := range entities {
		if _, err := service.ProcessActionWithinTx(context.Background(), ProcessActionCommand{
			UserID: userID, EventID: uuid.New(), ActionType: model.ActionTypeAdViewed,
			EntityID: &entity, Metadata: json.RawMessage(`{"source":"marketplace.view"}`),
			OccurredAt: now.Add(time.Duration(index) * time.Minute),
			Now:       now.Add(time.Duration(index) * time.Minute),
		}); err != nil {
			t.Fatalf("marketplace view %d error = %v", index+1, err)
		}
	}
	if _, ok := repository.achievements["EXPLORER"]; !ok {
		t.Fatal("EXPLORER was not unlocked after five distinct listing views")
	}
}

func TestManualAdCreatedActionDoesNotUnlockFirstAdAchievement(t *testing.T) {
	userID := mustGameUUID("00000000-0000-0000-0000-000000000003")
	repository := newGameTestRepository(userID)
	service := NewGameService(GameServiceDependencies{
		Repository: repository, TxManager: &gameTestTxManager{}, IDGenerator: uuid.New,
	})
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)

	if _, err := service.ProcessAction(context.Background(), ProcessActionCommand{
		UserID: userID, EventID: uuid.New(), ActionType: model.ActionTypeAdCreated,
		EntityID: gameStringPointer("listing-1"), Metadata: json.RawMessage(`{"source":"marketplace.publish"}`),
		OccurredAt: now, Now: now,
	}); err != nil {
		t.Fatalf("manual action error = %v", err)
	}
	if _, ok := repository.achievements["FIRST_AD"]; ok {
		t.Fatal("untrusted action unlocked FIRST_AD")
	}
}

func TestCharacterChangesFromExplorerToLeadingSellerOrQualityActivity(t *testing.T) {
	tests := []struct {
		name       string
		actionType model.ActionType
		want       model.PetCharacter
	}{
		{name: "seller", actionType: model.ActionTypeAdCreated, want: model.PetCharacterEntrepreneur},
		{name: "quality", actionType: model.ActionTypeListingImproved, want: model.PetCharacterCraftsperson},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := uuid.New()
			repository := newGameTestRepository(userID)
			service := NewGameService(GameServiceDependencies{
				Repository: repository, TxManager: &gameTestTxManager{}, IDGenerator: uuid.New,
			})
			now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)

		for index := 0; index < CharacterUnlockTarget; index++ {
			if _, err := service.ProcessAction(context.Background(), ProcessActionCommand{
				UserID: userID, EventID: uuid.New(), ActionType: model.ActionTypeAdViewed,
				OccurredAt: now.Add(time.Duration(index) * time.Minute),
				Now:        now.Add(time.Duration(index) * time.Minute),
			}); err != nil {
				t.Fatalf("explorer action %d error = %v", index+1, err)
			}
		}
		if repository.pet.Character == nil || *repository.pet.Character != model.PetCharacterExplorer {
			t.Fatalf("initial character = %v, want EXPLORER", repository.pet.Character)
		}

		for index := 0; index < CharacterUnlockTarget+1; index++ {
			if _, err := service.ProcessAction(context.Background(), ProcessActionCommand{
				UserID: userID, EventID: uuid.New(), ActionType: tt.actionType,
				OccurredAt: now.Add(time.Duration(CharacterUnlockTarget+index) * time.Minute),
				Now:        now.Add(time.Duration(CharacterUnlockTarget+index) * time.Minute),
			}); err != nil {
				t.Fatalf("%s action %d error = %v", tt.name, index+1, err)
			}
		}
		if repository.pet.Character == nil || *repository.pet.Character != tt.want {
			t.Fatalf("final character = %v, want %s", repository.pet.Character, tt.want)
		}
	})
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

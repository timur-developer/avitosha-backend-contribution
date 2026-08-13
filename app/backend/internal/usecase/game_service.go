package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
)

type DomainEventPublisher interface {
	Publish(uuid.UUID, []model.DomainEvent)
}

type AdviceGenerator interface {
	Generate(context.Context, AdviceGenerationInput) (string, error)
}

const (
	DefaultPetName    = "Авитоша"
	DefaultRewardType = "AVITO_BONUS"
)

type IDGenerator func() uuid.UUID

type GameServiceDependencies struct {
	Repository  GameRepository
	TxManager   TxManager
	IDGenerator IDGenerator
	Publisher   DomainEventPublisher
	Advice      AdviceGenerator
}

type GameService struct {
	repository  GameRepository
	txManager   TxManager
	idGenerator IDGenerator
	publisher   DomainEventPublisher
	advice      AdviceGenerator
}

type GameProfile struct {
	Pet            model.Pet
	Story          model.StorySnapshot
	ActivityScores model.ActivityScores
	Character      CharacterProfile
	NextLevelXP    *int
}

type CharacterProfile struct {
	Code         model.PetCharacter
	Name         string
	Description  string
	IconKey      string
	VisualDetail string
	Progress     int
	Target       int
	Unlocked     bool
}

type DailySummary struct {
	Progress       model.DailyProgress
	WeeklyPosition *int
	Retention      RetentionOverview
}

type Leaderboard struct {
	WeekStart   time.Time
	Leaders     []model.LeaderboardEntry
	CurrentUser model.LeaderboardEntry
}

type ProcessActionCommand struct {
	UserID     uuid.UUID
	EventID    uuid.UUID
	ActionType model.ActionType
	EntityID   *string
	Category   *string
	Metadata   json.RawMessage
	OccurredAt time.Time
	Now        time.Time
}

type ProcessActionResult struct {
	ActionID  uuid.UUID           `json:"actionId"`
	Duplicate bool                `json:"duplicate"`
	Events    []model.DomainEvent `json:"events"`
}

type AdviceGenerationInput struct {
	PetName           string
	PetMood           model.PetMood
	CharacterName     string
	TaskTitle         string
	TaskDescription   string
	ActionType        model.ActionType
	Progress          int
	Target            int
	Status            model.TaskStatus
	XPReward          int
	RoomItemCode      string
	AvitoRewardType   string
	AvitoRewardAmount int
}

type TaskAdvice struct {
	TaskID        uuid.UUID
	Text          string
	GeneratedByAI bool
}

func NewGameService(deps GameServiceDependencies) *GameService {
	idGenerator := deps.IDGenerator
	if idGenerator == nil {
		idGenerator = uuid.New
	}
	return &GameService{
		repository: deps.Repository, txManager: deps.TxManager,
		idGenerator: idGenerator, publisher: deps.Publisher, advice: deps.Advice,
	}
}

func (service *GameService) EnsureProfile(ctx context.Context, userID uuid.UUID, now time.Time) (GameProfile, error) {
	now = now.UTC()
	var profile GameProfile
	err := service.txManager.WithinTx(ctx, func(txCtx context.Context) error {
		pet, err := service.repository.GetOrCreateGamePet(txCtx, model.Pet{
			ID: service.idGenerator(), UserID: userID, Name: DefaultPetName,
			Level: 1, GrowthXP: 0, Mood: model.PetMoodCalm, CreatedAt: now, UpdatedAt: now,
		})
		if err != nil {
			return fmt.Errorf("get or create game pet: %w", err)
		}

		_, err = service.repository.GetOrCreateStoryProgress(txCtx, model.UserStoryProgress{
			ID: service.idGenerator(), UserID: userID, StoryCode: FirstRoomStoryCode,
			CurrentStage: 0, Status: model.StoryStatusActive,
			StartedAt: now, CreatedAt: now, UpdatedAt: now,
		})
		if err != nil {
			return fmt.Errorf("get or create first room story: %w", err)
		}

		placedAt := now
		if err = service.repository.EnsureInitialRoomItem(txCtx, model.UserRoomItem{
			ID: service.idGenerator(), UserID: userID, ItemCode: InitialRoomItemCode,
			Status: model.RoomItemStatusPlaced, UnlockedAt: now, PlacedAt: &placedAt,
			CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			return fmt.Errorf("ensure initial room item: %w", err)
		}

		if _, err = service.repository.AssignStoryTask(txCtx, userID, FirstRoomStoryCode, 1, now); err != nil {
			return fmt.Errorf("assign first story task: %w", err)
		}
		scores, err := service.repository.UpdateActivityScores(txCtx, userID, ActivityScoreDelta{}, now)
		if err != nil {
			return fmt.Errorf("ensure activity scores: %w", err)
		}
		if _, err = service.repository.EnsureRewardBalance(txCtx, userID, DefaultRewardType, now); err != nil {
			return fmt.Errorf("ensure reward balance: %w", err)
		}
		profile.Pet = pet
		profile.ActivityScores = scores
		return nil
	})
	if err != nil {
		return GameProfile{}, fmt.Errorf("ensure game profile transaction: %w", err)
	}

	story, err := service.repository.GetStorySnapshot(ctx, userID, FirstRoomStoryCode)
	if err != nil {
		return GameProfile{}, fmt.Errorf("get game profile story: %w", err)
	}
	profile.Story = story
	profile.Character = BuildCharacterProfile(profile.Pet, profile.ActivityScores)
	profile.NextLevelXP = NextLevelXP(profile.Pet.Level)
	return profile, nil
}

func (service *GameService) RenamePet(
	ctx context.Context,
	userID uuid.UUID,
	name string,
	now time.Time,
) (GameProfile, error) {
	normalizedName, err := ValidatePetName(name)
	if err != nil {
		return GameProfile{}, err
	}

	var profile GameProfile
	err = service.txManager.WithinTx(ctx, func(txCtx context.Context) error {
		profile, err = service.EnsureProfile(txCtx, userID, now)
		if err != nil {
			return err
		}
		profile.Pet.Name = normalizedName
		profile.Pet.UpdatedAt = now.UTC()
		if err = service.repository.UpdateGamePet(txCtx, profile.Pet); err != nil {
			return fmt.Errorf("rename pet: %w", err)
		}
		return nil
	})
	if err != nil {
		return GameProfile{}, fmt.Errorf("rename pet transaction: %w", err)
	}
	return profile, nil
}

func (service *GameService) GetDailySummary(
	ctx context.Context,
	userID uuid.UUID,
	now time.Time,
) (DailySummary, error) {
	profile, err := service.EnsureProfile(ctx, userID, now)
	if err != nil {
		return DailySummary{}, err
	}
	date := utcDate(now)
	progress, err := service.repository.GetDailyProgress(ctx, userID, date)
	if errors.Is(err, ErrDailyProgressNotFound) {
		progress = model.DailyProgress{
			Date: date, UserID: userID, LevelBefore: profile.Pet.Level, LevelAfter: profile.Pet.Level,
			StoryStageBefore: profile.Story.Progress.CurrentStage,
			StoryStageAfter:  profile.Story.Progress.CurrentStage, PetMood: profile.Pet.Mood,
			UnlockedRoomItems: []string{},
		}
	} else if err != nil {
		return DailySummary{}, fmt.Errorf("get daily progress: %w", err)
	}

	result := DailySummary{Progress: progress}
	position, err := service.repository.GetWeeklyPosition(ctx, userID, WeekStart(now))
	if err == nil {
		result.WeeklyPosition = &position.Position
	} else if !errors.Is(err, ErrLeaderboardEntryNotFound) {
		return DailySummary{}, fmt.Errorf("get daily leaderboard position: %w", err)
	}
	retention, err := service.buildRetentionOverview(ctx, userID, now)
	if err != nil {
		return DailySummary{}, fmt.Errorf("build retention overview: %w", err)
	}
	result.Retention = retention
	return result, nil
}

func (service *GameService) GetLeaderboard(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
	now time.Time,
) (Leaderboard, error) {
	if limit < 1 || limit > 100 {
		return Leaderboard{}, ErrInvalidAction
	}
	if _, err := service.EnsureProfile(ctx, userID, now); err != nil {
		return Leaderboard{}, err
	}
	weekStart := WeekStart(now)
	if _, err := service.repository.UpdateWeeklyProgress(
		ctx, userID, weekStart, WeeklyProgressDelta{}, now.UTC(),
	); err != nil {
		return Leaderboard{}, fmt.Errorf("ensure weekly leaderboard entry: %w", err)
	}
	leaders, err := service.repository.ListWeeklyLeaders(ctx, weekStart, limit)
	if err != nil {
		return Leaderboard{}, fmt.Errorf("list weekly leaders: %w", err)
	}
	current, err := service.repository.GetWeeklyPosition(ctx, userID, weekStart)
	if err != nil {
		return Leaderboard{}, fmt.Errorf("get current weekly position: %w", err)
	}
	return Leaderboard{WeekStart: weekStart, Leaders: leaders, CurrentUser: current}, nil
}

func (service *GameService) GetAchievements(
	ctx context.Context,
	userID uuid.UUID,
	now time.Time,
) ([]model.AchievementProgress, error) {
	if _, err := service.EnsureProfile(ctx, userID, now); err != nil {
		return nil, err
	}
	items, err := service.repository.ListAchievements(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list achievements: %w", err)
	}
	return items, nil
}

func (service *GameService) GetRewardBalances(
	ctx context.Context,
	userID uuid.UUID,
	now time.Time,
) ([]model.RewardBalance, error) {
	if _, err := service.EnsureProfile(ctx, userID, now); err != nil {
		return nil, err
	}
	balances, err := service.repository.ListRewardBalances(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list reward balances: %w", err)
	}
	return balances, nil
}

func (service *GameService) ListTasks(ctx context.Context, userID uuid.UUID, now time.Time) ([]model.TaskProgress, error) {
	if _, err := service.EnsureProfile(ctx, userID, now); err != nil {
		return nil, err
	}
	return service.repository.ListTaskProgress(ctx, userID)
}

func (service *GameService) GetTask(
	ctx context.Context,
	userID uuid.UUID,
	taskID uuid.UUID,
	now time.Time,
) (model.TaskProgress, error) {
	if _, err := service.EnsureProfile(ctx, userID, now); err != nil {
		return model.TaskProgress{}, err
	}
	return service.repository.GetTaskProgress(ctx, userID, taskID)
}

func (service *GameService) GetRoom(ctx context.Context, userID uuid.UUID, now time.Time) ([]model.RoomItemProgress, error) {
	if _, err := service.EnsureProfile(ctx, userID, now); err != nil {
		return nil, err
	}
	return service.repository.ListRoomItems(ctx, userID)
}

func (service *GameService) GetStory(ctx context.Context, userID uuid.UUID, now time.Time) (model.StorySnapshot, error) {
	if _, err := service.EnsureProfile(ctx, userID, now); err != nil {
		return model.StorySnapshot{}, err
	}
	return service.repository.GetStorySnapshot(ctx, userID, FirstRoomStoryCode)
}

func (service *GameService) ProcessAction(
	ctx context.Context,
	command ProcessActionCommand,
) (ProcessActionResult, error) {
	return service.processAction(ctx, command, true)
}

// ProcessActionWithinTx reuses the current action pipeline in a caller-owned transaction.
// The caller must publish result events only after its outer transaction commits.
func (service *GameService) ProcessActionWithinTx(
	ctx context.Context,
	command ProcessActionCommand,
) (ProcessActionResult, error) {
	return service.processAction(ctx, command, false)
}

func (service *GameService) PublishActionResult(userID uuid.UUID, result ProcessActionResult) {
	if !result.Duplicate && service.publisher != nil && len(result.Events) > 0 {
		service.publisher.Publish(userID, result.Events)
	}
}

func (service *GameService) processAction(ctx context.Context, command ProcessActionCommand, ownTransaction bool) (ProcessActionResult, error) {
	command = normalizeActionCommand(command)
	if err := validateActionCommand(command); err != nil {
		return ProcessActionResult{}, err
	}

	var result ProcessActionResult
	process := func(txCtx context.Context) error {
		profile, err := service.EnsureProfile(txCtx, command.UserID, command.Now)
		if err != nil {
			return err
		}

		action, inserted, err := service.repository.InsertAction(txCtx, model.UserAction{
			ID: service.idGenerator(), UserID: command.UserID, EventID: command.EventID,
			ActionType: command.ActionType, EntityID: command.EntityID, Category: command.Category,
			Metadata: command.Metadata, OccurredAt: command.OccurredAt, CreatedAt: command.Now,
		})
		if err != nil {
			return fmt.Errorf("insert user action: %w", err)
		}
		if !inserted {
			if !sameAction(action, command) {
				return ErrEventIDConflict
			}
			if action.ProcessedAt == nil {
				return fmt.Errorf("idempotent action is not processed: %w", ErrUnexpectedStorage)
			}
			if err := json.Unmarshal(action.ResultEvents, &result.Events); err != nil {
				return fmt.Errorf("decode idempotent action result: %w", err)
			}
			result.ActionID = action.ID
			result.Duplicate = true
			return nil
		}

		result.ActionID = action.ID
		pet := profile.Pet
		story := profile.Story.Progress
		levelBefore := pet.Level
		storyStageBefore := story.CurrentStage
		moodBefore := pet.Mood
		events := make([]model.DomainEvent, 0, 16)
		weeklyDelta := WeeklyProgressDelta{}
		unlockedItems := make([]string, 0, 1)
		achievementCodes := make([]string, 0, 4)
		rule := model.ProductActionRule{ActionType: command.ActionType}
		marketplaceAction := isTrustedMarketplaceAction(command.Metadata, ownTransaction)
		// FIRST_AD is tied to the real marketplace publication, not to the
		// order in which the story task happens to be assigned. Story catch-up
		// still covers publications that happened before this check was added.
		if marketplaceAction && command.ActionType == model.ActionTypeAdCreated {
			achievementCodes = append(achievementCodes, "FIRST_AD")
		}
		if marketplaceAction {
			switch command.ActionType {
			case model.ActionTypeMessageSent:
				// ContactSellerWithGame emits MESSAGE_SENT only for the first
				// message in a listing conversation.
				achievementCodes = append(achievementCodes, "IN_TOUCH")
			case model.ActionTypeAdViewed:
				// EXPLORER is based on five distinct listings, independently of
				// the category-specific story task.
				count, countErr := service.repository.CountDistinctUserActionEntities(
					txCtx, command.UserID, model.ActionTypeAdViewed,
				)
				if countErr != nil {
					return fmt.Errorf("count distinct viewed listings: %w", countErr)
				}
				if count >= ExplorerAchievementTarget {
					achievementCodes = append(achievementCodes, "EXPLORER")
				}
			}
		}
		if marketplaceAction {
			rules, ok := service.repository.(interface {
				GetProductActionRule(context.Context, model.ActionType) (model.ProductActionRule, error)
			})
			if !ok {
				return ErrUnexpectedStorage
			}
			var ruleErr error
			rule, ruleErr = rules.GetProductActionRule(txCtx, command.ActionType)
			if ruleErr != nil {
				return fmt.Errorf("get product action rule: %w", ruleErr)
			}
			if command.ActionType == model.ActionTypeMessageSent || command.ActionType == model.ActionTypeAdFavorited {
				count, countErr := service.repository.CountUserActionsOnDate(txCtx, command.UserID, command.ActionType, retentionDate(command.Now), command.EventID)
				if countErr != nil { return fmt.Errorf("count daily messages: %w", countErr) }
				if count > 0 { rule.XPReward = 0 }
			}
		}
		retentionState, _, err := service.ensureRetentionState(txCtx, command.UserID, command.Now, true)
		if err != nil {
			return fmt.Errorf("ensure retention state: %w", err)
		}
		dailyActionCompleted := dailyQuestCompletedForActionOnDate(
			retentionState.Quests, command, retentionDate(command.Now),
		)
		// Message product XP is independently limited to one per day; it must
		// not be suppressed merely because the daily message quest completes.
		if rule.XPReward > 0 && (!dailyActionCompleted || command.ActionType == model.ActionTypeMessageSent || command.ActionType == model.ActionTypeAdFavorited) {
			previousLevel := pet.Level
			pet.GrowthXP += rule.XPReward
			level, levelErr := CalculateLevel(pet.GrowthXP)
			if levelErr != nil {
				return fmt.Errorf("calculate product action level: %w", levelErr)
			}
			pet.Level = level
			pet.Mood = model.PetMoodHappy
			weeklyDelta.EarnedXP += rule.XPReward
			events = append(events, service.event(action.ID, command.UserID, model.DomainEventXPEarned, command.Now, map[string]any{"amount": rule.XPReward, "totalXp": pet.GrowthXP}))
			if pet.Level > previousLevel {
				events = append(events, service.event(action.ID, command.UserID, model.DomainEventPetLevelUp, command.Now, map[string]any{"previousLevel": previousLevel, "level": pet.Level}))
			}
		}
		if productEvent := service.productEvent(action.ID, command.UserID, command.ActionType, command.Now, command.EntityID, command.Category, marketplaceAction); productEvent.ID != uuid.Nil {
			events = append(events, productEvent)
		}

		tasks, err := service.repository.FindMatchingActiveTasksForUpdate(
			txCtx, command.UserID, command.ActionType, command.Category,
		)
		if err != nil {
			return fmt.Errorf("find matching active tasks: %w", err)
		}
		for _, taskProgress := range tasks {
			updated, completed := ApplyTaskProgress(taskProgress.Progress, command.Now)
			events = append(events, service.event(action.ID, command.UserID, model.DomainEventTaskProgressUpdated, command.Now, map[string]any{
				"taskId": taskProgress.Task.ID, "taskCode": taskProgress.Task.Code,
				"progress": updated.Progress, "target": updated.TargetValue,
			}))
			if updated.Progress == 1 && updated.TargetValue > 1 {
				pet.Mood = model.PetMoodCurious
			} else {
				pet.Mood = model.PetMoodHappy
			}

			if completed {
				rewardResult, rewardErr := service.rewardCompletedTask(
					txCtx, action.ID, command.UserID, taskProgress.Task, updated, pet, story, command.Now,
				)
				if rewardErr != nil {
					return rewardErr
				}
				updated = rewardResult.Progress
				pet = rewardResult.Pet
				story = rewardResult.Story
				events = append(events, rewardResult.Events...)
				weeklyDelta.EarnedXP += taskProgress.Task.XPReward
				weeklyDelta.CompletedTasks++
				if rewardResult.StoryAdvanced {
					weeklyDelta.CompletedStages++
				}
				weeklyDelta.EarnedXP += rewardResult.EarnedXP - taskProgress.Task.XPReward
				weeklyDelta.CompletedTasks += rewardResult.CompletedTasks
				weeklyDelta.CompletedStages += rewardResult.CompletedStages
				unlockedItems = append(unlockedItems, rewardResult.UnlockedItems...)
				achievementCodes = append(achievementCodes, rewardResult.AchievementCodes...)
				achievementCodes = append(achievementCodes, AchievementCodesForTask(
					taskProgress.Task.Code, len(rewardResult.UnlockedItems) > 0, story.Status == model.StoryStatusCompleted,
				)...)
			}

			if err := service.repository.UpdateTaskProgress(txCtx, updated); err != nil {
				return fmt.Errorf("save task progress: %w", err)
			}
		}
		retentionResult, err := service.applyRetentionForAction(txCtx, action.ID, command, retentionState)
		if err != nil {
			return err
		}
		events = append(events, retentionResult.Events...)
		if retentionResult.XPEarned > 0 {
			previousLevel := pet.Level
			pet.GrowthXP += retentionResult.XPEarned
			level, levelErr := CalculateLevel(pet.GrowthXP)
			if levelErr != nil {
				return fmt.Errorf("calculate retention level: %w", levelErr)
			}
			pet.Level = level
			pet.Mood = model.PetMoodHappy
			weeklyDelta.EarnedXP += retentionResult.XPEarned
			events = append(events, service.event(action.ID, command.UserID, model.DomainEventXPEarned, command.Now, map[string]any{
				"amount": retentionResult.XPEarned, "totalXp": pet.GrowthXP, "source": "DAILY_QUESTS",
			}))
			if pet.Level > previousLevel {
				events = append(events, service.event(action.ID, command.UserID, model.DomainEventPetLevelUp, command.Now, map[string]any{
					"previousLevel": previousLevel, "level": pet.Level,
				}))
			}
		}

		scores, err := service.repository.UpdateActivityScores(
			txCtx, command.UserID, ActivityDelta(command.ActionType, command.Category), command.Now,
		)
		if err != nil {
			return fmt.Errorf("update activity scores: %w", err)
		}
		character, _ := CharacterFromScores(scores)
		if character != nil && (pet.Character == nil || *pet.Character != *character) {
			wasUnlocked := pet.Character != nil
			pet.Character = character
			if !wasUnlocked {
				events = append(events, service.event(action.ID, command.UserID, model.DomainEventPetCharacterUnlocked, command.Now, map[string]any{
					"character": *character,
				}))
			}
		}

		pet.UpdatedAt = command.Now
		if err := service.repository.UpdateGamePet(txCtx, pet); err != nil {
			return fmt.Errorf("save pet progress: %w", err)
		}
		if pet.Mood != moodBefore {
			events = append(events, service.event(action.ID, command.UserID, model.DomainEventPetMoodChanged, command.Now, map[string]any{
				"previousMood": moodBefore, "mood": pet.Mood,
			}))
		}

		weekly, err := service.repository.UpdateWeeklyProgress(
			txCtx, command.UserID, WeekStart(command.Now), weeklyDelta, command.Now,
		)
		if err != nil {
			return fmt.Errorf("update weekly progress: %w", err)
		}
		weeklyScoreDelta := WeeklyScore(weeklyDelta)
		if weeklyScoreDelta > 0 {
			events = append(events, service.event(action.ID, command.UserID, model.DomainEventLeaderboardScoreUpdated, command.Now, map[string]any{
				"score": weekly.Score, "delta": weeklyScoreDelta,
			}))
		}

		unlockedAchievements, err := service.repository.UnlockAchievements(
			txCtx, command.UserID, uniqueStrings(achievementCodes), command.Now,
		)
		if err != nil {
			return fmt.Errorf("unlock achievements: %w", err)
		}
		for _, achievement := range unlockedAchievements {
			events = append(events, service.event(action.ID, command.UserID, model.DomainEventAchievementUnlocked, command.Now, map[string]any{
				"code": achievement.AchievementCode,
			}))
		}

		_, err = service.repository.UpdateDailyProgress(txCtx, command.UserID, utcDate(command.Now), DailyProgressDelta{
			ActionsCount: 1, CompletedTasks: weeklyDelta.CompletedTasks, EarnedXP: weeklyDelta.EarnedXP,
			LevelBefore: levelBefore, LevelAfter: pet.Level, UnlockedRoomItems: unlockedItems,
			StoryStageBefore: storyStageBefore, StoryStageAfter: story.CurrentStage,
			WeeklyScoreDelta: weeklyScoreDelta, PetMood: pet.Mood,
		}, command.Now)
		if err != nil {
			return fmt.Errorf("update daily progress: %w", err)
		}

		if err := service.repository.InsertDomainEvents(txCtx, events); err != nil {
			return fmt.Errorf("persist domain events: %w", err)
		}
		if err := service.repository.CompleteAction(txCtx, action.ID, command.Now, events); err != nil {
			return fmt.Errorf("complete user action: %w", err)
		}
		result.Events = events
		return nil
	}
	var err error
	if ownTransaction {
		err = service.txManager.WithinTx(ctx, process)
	} else {
		err = process(ctx)
	}
	if err != nil {
		return ProcessActionResult{}, fmt.Errorf("process action transaction: %w", err)
	}

	if ownTransaction {
		service.PublishActionResult(command.UserID, result)
	}
	return result, nil
}

func (service *GameService) productEvent(actionID, userID uuid.UUID, actionType model.ActionType, now time.Time, entityID, category *string, marketplaceAction bool) model.DomainEvent {
	if !marketplaceAction {
		return model.DomainEvent{}
	}
	types := map[model.ActionType]model.DomainEventType{
		model.ActionTypeAdViewed: model.DomainEventListingViewed, model.ActionTypeAdFavorited: model.DomainEventListingFavorited,
		model.ActionTypeMessageSent: model.DomainEventSellerContacted, model.ActionTypeAdCreated: model.DomainEventListingPublished,
		model.ActionTypeListingImproved: model.DomainEventListingImproved, model.ActionTypeListingSold: model.DomainEventListingSold,
		model.ActionTypeDeliveryUsed: model.DomainEventDeliveryUsed,
	}
	eventType, ok := types[actionType]
	if !ok {
		return model.DomainEvent{}
	}
	payload := map[string]any{"actionType": actionType}
	if entityID != nil {
		payload["listingId"] = *entityID
	}
	if category != nil {
		payload["category"] = *category
	}
	return service.event(actionID, userID, eventType, now, payload)
}

func isMarketplaceAction(metadata json.RawMessage) bool {
	var value struct {
		Source string `json:"source"`
	}
	return json.Unmarshal(metadata, &value) == nil && strings.HasPrefix(value.Source, "marketplace.")
}

func isTrustedMarketplaceAction(metadata json.RawMessage, ownTransaction bool) bool {
	return !ownTransaction && isMarketplaceAction(metadata)
}

type taskRewardResult struct {
	Progress      model.UserTask
	Pet           model.Pet
	Story         model.UserStoryProgress
	Events        []model.DomainEvent
	UnlockedItems []string
	StoryAdvanced bool
	CompletedTasks int
	CompletedStages int
	EarnedXP int
	AchievementCodes []string
}

func (service *GameService) rewardCompletedTask(
	ctx context.Context,
	actionID, userID uuid.UUID,
	task model.Task,
	progress model.UserTask,
	pet model.Pet,
	story model.UserStoryProgress,
	now time.Time,
) (taskRewardResult, error) {
	result := taskRewardResult{Progress: RewardTask(progress, now), Pet: pet, Story: story, EarnedXP: task.XPReward}
	result.Events = append(result.Events, service.event(actionID, userID, model.DomainEventTaskCompleted, now, map[string]any{
		"taskId": task.ID, "taskCode": task.Code, "taskTitle": task.Title,
		"xpReward": task.XPReward, "avitoRewardAmount": task.AvitoRewardAmount,
	}))

	previousLevel := result.Pet.Level
	result.Pet.GrowthXP += task.XPReward
	level, err := CalculateLevel(result.Pet.GrowthXP)
	if err != nil {
		return taskRewardResult{}, fmt.Errorf("calculate pet level: %w", err)
	}
	result.Pet.Level = level
	result.Pet.Mood = model.PetMoodExcited
	result.Events = append(result.Events, service.event(actionID, userID, model.DomainEventXPEarned, now, map[string]any{
		"amount": task.XPReward, "totalXp": result.Pet.GrowthXP,
	}))
	if task.AvitoRewardType != nil && task.AvitoRewardAmount > 0 {
		title := task.Title
		balance, credited, creditErr := service.repository.CreditReward(ctx, model.RewardCredit{
			ID: service.idGenerator(), UserID: userID, ActionID: actionID, TaskID: &task.ID,
			RewardType: *task.AvitoRewardType, Amount: task.AvitoRewardAmount,
			SourceKind: model.RewardSourceTaskCompletion, SourceRef: task.ID.String(),
			SourceTitle: &title, CreatedAt: now,
		})
		if creditErr != nil {
			return taskRewardResult{}, fmt.Errorf("credit task reward: %w", creditErr)
		}
		if credited {
			rewardEvents, rewardErr := service.rewardEventsForCredit(ctx, actionID, userID, model.RewardCredit{
				TaskID: &task.ID, RewardType: *task.AvitoRewardType, Amount: task.AvitoRewardAmount,
				SourceKind: model.RewardSourceTaskCompletion, SourceRef: task.ID.String(), SourceTitle: &title,
			}, balance, now)
			if rewardErr != nil {
				return taskRewardResult{}, rewardErr
			}
			result.Events = append(result.Events, rewardEvents...)
		}
	}
	if result.Pet.Level > previousLevel {
		result.Events = append(result.Events, service.event(actionID, userID, model.DomainEventPetLevelUp, now, map[string]any{
			"previousLevel": previousLevel, "level": result.Pet.Level,
		}))
	}

	if task.RoomItemCode != nil {
		placedAt := now
		unlocked, unlockErr := service.repository.UnlockRoomItem(ctx, model.UserRoomItem{
			ID: service.idGenerator(), UserID: userID, ItemCode: *task.RoomItemCode,
			Status: model.RoomItemStatusPlaced, SourceTaskID: &task.ID,
			UnlockedAt: now, PlacedAt: &placedAt, CreatedAt: now, UpdatedAt: now,
		})
		if unlockErr != nil {
			return taskRewardResult{}, fmt.Errorf("unlock room item: %w", unlockErr)
		}
		if unlocked {
			result.UnlockedItems = append(result.UnlockedItems, *task.RoomItemCode)
			result.Pet.Mood = model.PetMoodProud
			result.Events = append(result.Events, service.event(actionID, userID, model.DomainEventRoomItemUnlocked, now, map[string]any{
				"itemCode": *task.RoomItemCode,
			}))
		}
	}

	if task.StoryCode != nil && task.StoryStage != nil {
		if result.Story.StoryCode != *task.StoryCode || *task.StoryStage != result.Story.CurrentStage+1 {
			return taskRewardResult{}, ErrOutOfOrderStoryStage
		}
		result.Story.CurrentStage = *task.StoryStage
		result.Story.UpdatedAt = now
		result.StoryAdvanced = true
		result.Events = append(result.Events, service.event(actionID, userID, model.DomainEventStoryStageCompleted, now, map[string]any{
			"storyCode": *task.StoryCode, "stage": *task.StoryStage,
		}))

		storySnapshot, storyErr := service.repository.GetStorySnapshot(ctx, userID, *task.StoryCode)
		if storyErr != nil {
			return taskRewardResult{}, fmt.Errorf("get story total stages: %w", storyErr)
		}
		if result.Story.CurrentStage >= storySnapshot.Story.TotalStages {
			completedAt := now
			result.Story.Status = model.StoryStatusCompleted
			result.Story.CompletedAt = &completedAt
			result.Events = append(result.Events, service.event(actionID, userID, model.DomainEventStoryCompleted, now, map[string]any{
				"storyCode": *task.StoryCode,
			}))
		} else {
			next, assignErr := service.repository.AssignStoryTask(
				ctx, userID, *task.StoryCode, result.Story.CurrentStage+1, now,
			)
			if assignErr != nil {
				return taskRewardResult{}, fmt.Errorf("assign next story task: %w", assignErr)
			}
			if next.Task.TargetValue == 1 {
				count, countErr := service.repository.CountUserActions(ctx, userID, next.Task.ActionType, next.Task.Category, nil)
				if countErr != nil { return taskRewardResult{}, fmt.Errorf("check historical story action: %w", countErr) }
				if count > 0 {
					next.Progress.Progress = 1
					completedAt := now
					next.Progress.CompletedAt = &completedAt
					next.Progress.Status = model.TaskStatusCompleted
					nextResult, rewardErr := service.rewardCompletedTask(ctx, actionID, userID, next.Task, next.Progress, result.Pet, result.Story, now)
					if rewardErr != nil { return taskRewardResult{}, rewardErr }
					result.Pet, result.Story = nextResult.Pet, nextResult.Story
					result.Events = append(result.Events, nextResult.Events...)
					result.UnlockedItems = append(result.UnlockedItems, nextResult.UnlockedItems...)
					// The current task already advanced the story. Keep that flag set
					// while carrying only the nested task counts upward.
					result.StoryAdvanced = true
					result.EarnedXP += nextResult.EarnedXP
					result.CompletedTasks = 1 + nextResult.CompletedTasks
					result.CompletedStages = 1 + nextResult.CompletedStages
					result.AchievementCodes = append(result.AchievementCodes, nextResult.AchievementCodes...)
					result.AchievementCodes = append(result.AchievementCodes, AchievementCodesForTask(
						next.Task.Code, len(nextResult.UnlockedItems) > 0, nextResult.Story.Status == model.StoryStatusCompleted,
					)...)
					if err := service.repository.UpdateTaskProgress(ctx, nextResult.Progress); err != nil { return taskRewardResult{}, err }
				}
			}
		}
		if err := service.repository.UpdateStoryProgress(ctx, result.Story); err != nil {
			return taskRewardResult{}, fmt.Errorf("save story progress: %w", err)
		}
	}

	return result, nil
}

func (service *GameService) event(
	actionID, userID uuid.UUID,
	eventType model.DomainEventType,
	now time.Time,
	payload map[string]any,
) model.DomainEvent {
	encoded, err := json.Marshal(payload)
	if err != nil {
		panic(fmt.Sprintf("marshal static domain event payload: %v", err))
	}
	return model.DomainEvent{
		ID: service.idGenerator(), Type: eventType, UserID: userID, ActionID: actionID,
		OccurredAt: now, Payload: encoded,
	}
}

func normalizeActionCommand(command ProcessActionCommand) ProcessActionCommand {
	command.Now = command.Now.UTC().Truncate(time.Microsecond)
	command.OccurredAt = command.OccurredAt.UTC().Truncate(time.Microsecond)
	command.ActionType = model.ActionType(strings.ToUpper(strings.TrimSpace(string(command.ActionType))))
	command.EntityID = normalizedStringPointer(command.EntityID)
	command.Category = normalizedUpperStringPointer(command.Category)
	if len(command.Metadata) == 0 {
		command.Metadata = json.RawMessage(`{}`)
	}
	return command
}

func validateActionCommand(command ProcessActionCommand) error {
	if command.UserID == uuid.Nil || command.EventID == uuid.Nil || !ValidActionType(command.ActionType) {
		return ErrInvalidAction
	}
	if command.Now.IsZero() || command.OccurredAt.IsZero() {
		return ErrInvalidAction
	}
	var metadata any
	if err := json.Unmarshal(command.Metadata, &metadata); err != nil {
		return ErrInvalidAction
	}
	if _, ok := metadata.(map[string]any); !ok {
		return ErrInvalidAction
	}
	return nil
}

func sameAction(action model.UserAction, command ProcessActionCommand) bool {
	if action.UserID != command.UserID || action.ActionType != command.ActionType ||
		!equalStringPointers(action.EntityID, command.EntityID) ||
		!equalStringPointers(action.Category, command.Category) ||
		!action.OccurredAt.Equal(command.OccurredAt) {
		return false
	}
	var storedMetadata, requestedMetadata any
	if json.Unmarshal(action.Metadata, &storedMetadata) != nil || json.Unmarshal(command.Metadata, &requestedMetadata) != nil {
		return false
	}
	return reflect.DeepEqual(storedMetadata, requestedMetadata)
}

func normalizedStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil
	}
	return &normalized
}

func normalizedUpperStringPointer(value *string) *string {
	normalized := normalizedStringPointer(value)
	if normalized == nil {
		return nil
	}
	upper := strings.ToUpper(*normalized)
	return &upper
}

func equalStringPointers(first, second *string) bool {
	if first == nil || second == nil {
		return first == nil && second == nil
	}
	return *first == *second
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

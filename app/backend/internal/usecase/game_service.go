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
}

type GameService struct {
	repository  GameRepository
	txManager   TxManager
	idGenerator IDGenerator
	publisher   DomainEventPublisher
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

func NewGameService(deps GameServiceDependencies) *GameService {
	idGenerator := deps.IDGenerator
	if idGenerator == nil {
		idGenerator = uuid.New
	}
	return &GameService{
		repository: deps.Repository, txManager: deps.TxManager,
		idGenerator: idGenerator, publisher: deps.Publisher,
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
	command = normalizeActionCommand(command)
	if err := validateActionCommand(command); err != nil {
		return ProcessActionResult{}, err
	}

	var result ProcessActionResult
	err := service.txManager.WithinTx(ctx, func(txCtx context.Context) error {
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
		retentionState, _, err := service.ensureRetentionState(txCtx, command.UserID, command.Now, true)
		if err != nil {
			return fmt.Errorf("ensure retention state: %w", err)
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
				if rewardResult.UnlockedItem != "" {
					unlockedItems = append(unlockedItems, rewardResult.UnlockedItem)
				}
				achievementCodes = append(achievementCodes, AchievementCodesForTask(
					taskProgress.Task.Code, rewardResult.UnlockedItem != "", story.Status == model.StoryStatusCompleted,
				)...)
			}

			if err := service.repository.UpdateTaskProgress(txCtx, updated); err != nil {
				return fmt.Errorf("save task progress: %w", err)
			}
		}
		_, retentionEvents, err := service.applyRetentionForAction(txCtx, action.ID, command, retentionState)
		if err != nil {
			return err
		}
		events = append(events, retentionEvents...)

		scores, err := service.repository.UpdateActivityScores(
			txCtx, command.UserID, ActivityDelta(command.ActionType, command.Category), command.Now,
		)
		if err != nil {
			return fmt.Errorf("update activity scores: %w", err)
		}
		character, _ := CharacterFromScores(scores)
		if pet.Character == nil && character != nil {
			pet.Character = character
			events = append(events, service.event(action.ID, command.UserID, model.DomainEventPetCharacterUnlocked, command.Now, map[string]any{
				"character": *character,
			}))
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
	})
	if err != nil {
		return ProcessActionResult{}, fmt.Errorf("process action transaction: %w", err)
	}

	if !result.Duplicate && service.publisher != nil && len(result.Events) > 0 {
		service.publisher.Publish(command.UserID, result.Events)
	}
	return result, nil
}

type taskRewardResult struct {
	Progress      model.UserTask
	Pet           model.Pet
	Story         model.UserStoryProgress
	Events        []model.DomainEvent
	UnlockedItem  string
	StoryAdvanced bool
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
	result := taskRewardResult{Progress: RewardTask(progress, now), Pet: pet, Story: story}
	result.Events = append(result.Events, service.event(actionID, userID, model.DomainEventTaskCompleted, now, map[string]any{
		"taskId": task.ID, "taskCode": task.Code,
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
			result.UnlockedItem = *task.RoomItemCode
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
			if _, assignErr := service.repository.AssignStoryTask(
				ctx, userID, *task.StoryCode, result.Story.CurrentStage+1, now,
			); assignErr != nil {
				return taskRewardResult{}, fmt.Errorf("assign next story task: %w", assignErr)
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

package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
)

type WeeklyProgressDelta struct {
	EarnedXP        int
	CompletedTasks  int
	CompletedStages int
}

type DailyProgressDelta struct {
	ActionsCount      int
	CompletedTasks    int
	EarnedXP          int
	LevelBefore       int
	LevelAfter        int
	UnlockedRoomItems []string
	StoryStageBefore  int
	StoryStageAfter   int
	WeeklyScoreDelta  int
	PetMood           model.PetMood
}

type ActivityScoreDelta struct {
	Buyer      int
	Seller     int
	Auto       int
	Travel     int
	RealEstate int
	Services   int
}

type GameRepository interface {
	GetOrCreateGamePet(context.Context, model.Pet) (model.Pet, error)
	GetGamePetByUserID(context.Context, uuid.UUID) (model.Pet, error)
	GetGamePetByUserIDForUpdate(context.Context, uuid.UUID) (model.Pet, error)
	UpdateGamePet(context.Context, model.Pet) error

	GetOrCreateStoryProgress(context.Context, model.UserStoryProgress) (model.UserStoryProgress, error)
	GetStorySnapshot(context.Context, uuid.UUID, string) (model.StorySnapshot, error)
	GetStoryProgressForUpdate(context.Context, uuid.UUID, string) (model.UserStoryProgress, error)
	UpdateStoryProgress(context.Context, model.UserStoryProgress) error

	EnsureInitialRoomItem(context.Context, model.UserRoomItem) error
	UnlockRoomItem(context.Context, model.UserRoomItem) (bool, error)
	ListRoomItems(context.Context, uuid.UUID) ([]model.RoomItemProgress, error)

	AssignStoryTask(context.Context, uuid.UUID, string, int, time.Time) (model.TaskProgress, error)
	ListTaskProgress(context.Context, uuid.UUID) ([]model.TaskProgress, error)
	GetTaskProgress(context.Context, uuid.UUID, uuid.UUID) (model.TaskProgress, error)
	FindMatchingActiveTasksForUpdate(context.Context, uuid.UUID, model.ActionType, *string) ([]model.TaskProgress, error)
	UpdateTaskProgress(context.Context, model.UserTask) error

	InsertAction(context.Context, model.UserAction) (model.UserAction, bool, error)
	CompleteAction(context.Context, uuid.UUID, time.Time, []model.DomainEvent) error
	InsertDomainEvents(context.Context, []model.DomainEvent) error

	UpdateWeeklyProgress(context.Context, uuid.UUID, time.Time, WeeklyProgressDelta, time.Time) (model.WeeklyProgress, error)
	UpdateDailyProgress(context.Context, uuid.UUID, time.Time, DailyProgressDelta, time.Time) (model.DailyProgress, error)
	GetDailyProgress(context.Context, uuid.UUID, time.Time) (model.DailyProgress, error)
	ListWeeklyLeaders(context.Context, time.Time, int) ([]model.LeaderboardEntry, error)
	GetWeeklyPosition(context.Context, uuid.UUID, time.Time) (model.LeaderboardEntry, error)

	UpdateActivityScores(context.Context, uuid.UUID, ActivityScoreDelta, time.Time) (model.ActivityScores, error)
	GetActivityScores(context.Context, uuid.UUID) (model.ActivityScores, error)
	UnlockAchievements(context.Context, uuid.UUID, []string, time.Time) ([]model.UserAchievement, error)
	ListAchievements(context.Context, uuid.UUID) ([]model.AchievementProgress, error)

	EnsureRewardBalance(context.Context, uuid.UUID, string, time.Time) (model.RewardBalance, error)
	CreditReward(context.Context, model.RewardCredit) (model.RewardBalance, bool, error)
	ListRewardBalances(context.Context, uuid.UUID) ([]model.RewardBalance, error)
	ListRewardCatalog(context.Context) ([]model.RewardCatalogItem, error)

	GetOrCreateUserStreak(context.Context, model.UserStreak) (model.UserStreak, error)
	UpdateUserStreak(context.Context, model.UserStreak) error

	ListActiveDailyQuestTemplates(context.Context) ([]model.DailyQuestTemplate, error)
	ExpireDailyQuestsBefore(context.Context, uuid.UUID, time.Time, time.Time) error
	AssignDailyQuest(context.Context, model.UserDailyQuest) (model.UserDailyQuest, error)
	GetDailyQuestProgress(context.Context, uuid.UUID, time.Time) (model.DailyQuestProgress, error)
	GetDailyQuestProgressForUpdate(context.Context, uuid.UUID, time.Time) (model.DailyQuestProgress, error)
	UpdateDailyQuest(context.Context, model.UserDailyQuest) error
}

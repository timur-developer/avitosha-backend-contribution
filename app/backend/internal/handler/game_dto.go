package handler

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

type gamePetDTO struct {
	ID               uuid.UUID           `json:"id"`
	Name             string              `json:"name"`
	Level            int                 `json:"level"`
	GrowthXP         int                 `json:"growthXp"`
	NextLevelXP      *int                `json:"nextLevelXp"`
	Mood             model.PetMood       `json:"mood"`
	Character        *model.PetCharacter `json:"character"`
	CharacterProfile characterProfileDTO `json:"characterProfile"`
	CurrentStory     currentStoryDTO     `json:"currentStory"`
}

type characterProfileDTO struct {
	Code         model.PetCharacter `json:"code"`
	Name         string             `json:"name"`
	Description  string             `json:"description"`
	IconKey      string             `json:"iconKey"`
	VisualDetail string             `json:"visualDetail"`
	Progress     int                `json:"progress"`
	Target       int                `json:"target"`
	Unlocked     bool               `json:"unlocked"`
}

type currentStoryDTO struct {
	Code         string `json:"code"`
	Title        string `json:"title"`
	CurrentStage int    `json:"currentStage"`
	TotalStages  int    `json:"totalStages"`
	Status       string `json:"status"`
}

func newGamePetDTO(profile usecase.GameProfile) gamePetDTO {
	return gamePetDTO{
		ID: profile.Pet.ID, Name: profile.Pet.Name, Level: profile.Pet.Level,
		GrowthXP: profile.Pet.GrowthXP, NextLevelXP: profile.NextLevelXP,
		Mood: profile.Pet.Mood, Character: profile.Pet.Character,
		CharacterProfile: characterProfileDTO{
			Code: profile.Character.Code, Name: profile.Character.Name,
			Description: profile.Character.Description, IconKey: profile.Character.IconKey,
			VisualDetail: profile.Character.VisualDetail, Progress: profile.Character.Progress,
			Target: profile.Character.Target, Unlocked: profile.Character.Unlocked,
		},
		CurrentStory: currentStoryDTO{
			Code: profile.Story.Story.Code, Title: profile.Story.Story.Title,
			CurrentStage: profile.Story.Progress.CurrentStage,
			TotalStages:  profile.Story.Story.TotalStages, Status: string(profile.Story.Progress.Status),
		},
	}
}

type taskListDTO struct {
	Tasks []taskDTO `json:"tasks"`
}

type taskDTO struct {
	ID                uuid.UUID        `json:"id"`
	Code              string           `json:"code"`
	Title             string           `json:"title"`
	Description       string           `json:"description"`
	PetPhrase         string           `json:"petPhrase"`
	ActionType        model.ActionType `json:"actionType"`
	Category          *string          `json:"category"`
	Progress          int              `json:"progress"`
	Target            int              `json:"target"`
	Status            model.TaskStatus `json:"status"`
	XPReward          int              `json:"xpReward"`
	RoomItemCode      *string          `json:"roomItemCode"`
	AvitoRewardType   *string          `json:"avitoRewardType"`
	AvitoRewardAmount int              `json:"avitoRewardAmount"`
	StoryStage        *int             `json:"storyStage"`
}

func newTaskDTOs(tasks []model.TaskProgress) []taskDTO {
	result := make([]taskDTO, len(tasks))
	for index, task := range tasks {
		result[index] = newTaskDTO(task)
	}
	return result
}

func newTaskDTO(task model.TaskProgress) taskDTO {
	return taskDTO{
		ID: task.Task.ID, Code: task.Task.Code, Title: task.Task.Title,
		Description: task.Task.Description, PetPhrase: task.Task.PetPhrase,
		ActionType: task.Task.ActionType, Category: task.Task.Category,
		Progress: task.Progress.Progress, Target: task.Progress.TargetValue,
		Status: task.Progress.Status, XPReward: task.Task.XPReward,
		RoomItemCode: task.Task.RoomItemCode, AvitoRewardType: task.Task.AvitoRewardType,
		AvitoRewardAmount: task.Task.AvitoRewardAmount, StoryStage: task.Task.StoryStage,
	}
}

type actionResultDTO struct {
	ActionID  uuid.UUID        `json:"actionId"`
	Duplicate bool             `json:"duplicate"`
	Events    []map[string]any `json:"events"`
}

func newActionResultDTO(result usecase.ProcessActionResult) actionResultDTO {
	events := make([]map[string]any, len(result.Events))
	for index, event := range result.Events {
		events[index] = newDomainEventDTO(event)
	}
	return actionResultDTO{ActionID: result.ActionID, Duplicate: result.Duplicate, Events: events}
}

func newDomainEventDTO(event model.DomainEvent) map[string]any {
	payload := make(map[string]any)
	_ = json.Unmarshal(event.Payload, &payload)
	payload["id"] = event.ID
	payload["type"] = event.Type
	payload["occurredAt"] = event.OccurredAt.UTC().Format(time.RFC3339Nano)
	return payload
}

type roomDTO struct {
	StoryCode string        `json:"storyCode"`
	Progress  string        `json:"progress"`
	Items     []roomItemDTO `json:"items"`
}

type roomItemDTO struct {
	Code           string               `json:"code"`
	Name           string               `json:"name"`
	Description    string               `json:"description"`
	Status         model.RoomItemStatus `json:"status"`
	AssetKey       string               `json:"assetKey"`
	PositionKey    string               `json:"positionKey"`
	UnlockTaskCode *string              `json:"unlockTaskCode"`
}

func newRoomDTO(items []model.RoomItemProgress) roomDTO {
	dtos := make([]roomItemDTO, len(items))
	placed := 0
	storyItems := 0
	for index, item := range items {
		dtos[index] = roomItemDTO{
			Code: item.Item.Code, Name: item.Item.Name, Description: item.Item.Description,
			Status: item.Status, AssetKey: item.Item.AssetKey, PositionKey: item.Item.PositionKey,
			UnlockTaskCode: item.SourceTaskCode,
		}
		if item.Item.SortOrder <= 5 {
			storyItems++
		}
		if item.Item.SortOrder <= 5 && item.Status == model.RoomItemStatusPlaced {
			placed++
		}
	}
	return roomDTO{
		StoryCode: usecase.FirstRoomStoryCode,
		Progress:  fmtProgress(placed, storyItems), Items: dtos,
	}
}

func fmtProgress(current, total int) string {
	return fmt.Sprintf("%d/%d", current, total)
}

type storyDTO struct {
	Code         string            `json:"code"`
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	CurrentStage int               `json:"currentStage"`
	TotalStages  int               `json:"totalStages"`
	Status       model.StoryStatus `json:"status"`
	NextTask     *taskPreviewDTO   `json:"nextTask"`
}

type taskPreviewDTO struct {
	ID           uuid.UUID `json:"id"`
	Code         string    `json:"code"`
	Title        string    `json:"title"`
	RoomItemCode *string   `json:"roomItemCode"`
}

func newStoryDTO(snapshot model.StorySnapshot) storyDTO {
	result := storyDTO{
		Code: snapshot.Story.Code, Title: snapshot.Story.Title,
		Description: snapshot.Story.Description, CurrentStage: snapshot.Progress.CurrentStage,
		TotalStages: snapshot.Story.TotalStages, Status: snapshot.Progress.Status,
	}
	if snapshot.NextTask != nil {
		result.NextTask = &taskPreviewDTO{
			ID: snapshot.NextTask.ID, Code: snapshot.NextTask.Code,
			Title: snapshot.NextTask.Title, RoomItemCode: snapshot.NextTask.RoomItemCode,
		}
	}
	return result
}

type dailySummaryDTO struct {
	Date              string        `json:"date"`
	ActionsCount      int           `json:"actionsCount"`
	CompletedTasks    int           `json:"completedTasks"`
	EarnedXP          int           `json:"earnedXp"`
	LevelBefore       int           `json:"levelBefore"`
	LevelAfter        int           `json:"levelAfter"`
	UnlockedRoomItems []string      `json:"unlockedRoomItems"`
	StoryStageBefore  int           `json:"storyStageBefore"`
	StoryStageAfter   int           `json:"storyStageAfter"`
	WeeklyScoreDelta  int           `json:"weeklyScoreDelta"`
	WeeklyPosition    *int          `json:"weeklyPosition"`
	PetMood           model.PetMood `json:"petMood"`
	Retention         retentionDTO  `json:"retention"`
}

func newDailySummaryDTO(summary usecase.DailySummary) dailySummaryDTO {
	return dailySummaryDTO{
		Date:         summary.Progress.Date.UTC().Format(time.DateOnly),
		ActionsCount: summary.Progress.ActionsCount, CompletedTasks: summary.Progress.CompletedTasks,
		EarnedXP: summary.Progress.EarnedXP, LevelBefore: summary.Progress.LevelBefore,
		LevelAfter:        summary.Progress.LevelAfter,
		UnlockedRoomItems: append([]string(nil), summary.Progress.UnlockedRoomItems...),
		StoryStageBefore:  summary.Progress.StoryStageBefore,
		StoryStageAfter:   summary.Progress.StoryStageAfter,
		WeeklyScoreDelta:  summary.Progress.WeeklyScoreDelta,
		WeeklyPosition:    summary.WeeklyPosition, PetMood: summary.Progress.PetMood,
		Retention: newRetentionDTO(summary.Retention),
	}
}

type retentionDTO struct {
	Streak     streakDTO          `json:"streak"`
	DailyQuest dailyQuestDTO      `json:"dailyQuest"`
	Tomorrow   tomorrowPreviewDTO `json:"tomorrow"`
}

type streakDTO struct {
	Current        int            `json:"current"`
	Longest        int            `json:"longest"`
	LastActiveDate *string        `json:"lastActiveDate"`
	ActiveToday    bool           `json:"activeToday"`
	Reward         rewardOfferDTO `json:"reward"`
}

type dailyQuestDTO struct {
	Date        string                 `json:"date"`
	Code        string                 `json:"code"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	ActionType  model.ActionType       `json:"actionType"`
	Category    *string                `json:"category"`
	Progress    int                    `json:"progress"`
	Target      int                    `json:"target"`
	Status      model.DailyQuestStatus `json:"status"`
	Reward      rewardOfferDTO         `json:"reward"`
}

type rewardOfferDTO struct {
	Type   string                 `json:"type"`
	Amount int                    `json:"amount"`
	Source model.RewardSourceKind `json:"source"`
}

type tomorrowPreviewDTO struct {
	Date              string                `json:"date"`
	StreakAfterReturn int                   `json:"streakAfterReturn"`
	StreakReward      rewardOfferDTO        `json:"streakReward"`
	DailyQuest        tomorrowDailyQuestDTO `json:"dailyQuest"`
	NextGoal          *rewardGoalDTO        `json:"nextGoal"`
}

type tomorrowDailyQuestDTO struct {
	Code        string           `json:"code"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	ActionType  model.ActionType `json:"actionType"`
	Category    *string          `json:"category"`
	Target      int              `json:"target"`
	Reward      rewardOfferDTO   `json:"reward"`
}

func newRetentionDTO(retention usecase.RetentionOverview) retentionDTO {
	var lastActiveDate *string
	if retention.Streak.LastActiveDate != nil {
		formatted := retention.Streak.LastActiveDate.UTC().Format(time.DateOnly)
		lastActiveDate = &formatted
	}
	return retentionDTO{
		Streak: streakDTO{
			Current: retention.Streak.Current, Longest: retention.Streak.Longest,
			LastActiveDate: lastActiveDate, ActiveToday: retention.Streak.ActiveToday,
			Reward: newRewardOfferDTO(retention.Streak.Reward),
		},
		DailyQuest: dailyQuestDTO{
			Date: retention.DailyQuest.Date.UTC().Format(time.DateOnly),
			Code: retention.DailyQuest.Code, Title: retention.DailyQuest.Title,
			Description: retention.DailyQuest.Description, ActionType: retention.DailyQuest.ActionType,
			Category: retention.DailyQuest.Category, Progress: retention.DailyQuest.Progress,
			Target: retention.DailyQuest.Target, Status: retention.DailyQuest.Status,
			Reward: newRewardOfferDTO(retention.DailyQuest.Reward),
		},
		Tomorrow: tomorrowPreviewDTO{
			Date:              retention.Tomorrow.Date.UTC().Format(time.DateOnly),
			StreakAfterReturn: retention.Tomorrow.StreakAfterReturn,
			StreakReward:      newRewardOfferDTO(retention.Tomorrow.StreakReward),
			DailyQuest: tomorrowDailyQuestDTO{
				Code: retention.Tomorrow.DailyQuest.Code, Title: retention.Tomorrow.DailyQuest.Title,
				Description: retention.Tomorrow.DailyQuest.Description,
				ActionType:  retention.Tomorrow.DailyQuest.ActionType,
				Category:    retention.Tomorrow.DailyQuest.Category,
				Target:      retention.Tomorrow.DailyQuest.Target,
				Reward:      newRewardOfferDTO(retention.Tomorrow.DailyQuest.Reward),
			},
			NextGoal: newRewardGoalDTO(retention.Tomorrow.NextGoal),
		},
	}
}

func newRewardOfferDTO(offer usecase.RewardOffer) rewardOfferDTO {
	return rewardOfferDTO{Type: offer.Type, Amount: offer.Amount, Source: offer.Source}
}

type leaderboardDTO struct {
	WeekStart   string                `json:"weekStart"`
	Leaders     []leaderboardEntryDTO `json:"leaders"`
	CurrentUser leaderboardEntryDTO   `json:"currentUser"`
}

type leaderboardEntryDTO struct {
	Position       int       `json:"position"`
	UserID         uuid.UUID `json:"userId"`
	PetName        string    `json:"petName"`
	Level          int       `json:"level"`
	Score          int       `json:"score"`
	CompletedTasks int       `json:"completedTasks"`
}

func newLeaderboardDTO(leaderboard usecase.Leaderboard) leaderboardDTO {
	leaders := make([]leaderboardEntryDTO, len(leaderboard.Leaders))
	for index, entry := range leaderboard.Leaders {
		leaders[index] = newLeaderboardEntryDTO(entry)
	}
	return leaderboardDTO{
		WeekStart: leaderboard.WeekStart.UTC().Format(time.DateOnly), Leaders: leaders,
		CurrentUser: newLeaderboardEntryDTO(leaderboard.CurrentUser),
	}
}

func newLeaderboardEntryDTO(entry model.LeaderboardEntry) leaderboardEntryDTO {
	return leaderboardEntryDTO{
		Position: entry.Position, UserID: entry.UserID, PetName: entry.PetName,
		Level: entry.Level, Score: entry.Score, CompletedTasks: entry.CompletedTasks,
	}
}

type achievementsDTO struct {
	Achievements []achievementDTO `json:"achievements"`
}

type achievementDTO struct {
	Code        string     `json:"code"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	IconKey     string     `json:"iconKey"`
	Unlocked    bool       `json:"unlocked"`
	UnlockedAt  *time.Time `json:"unlockedAt"`
}

type rewardBalanceListDTO struct {
	Balances []rewardBalanceDTO `json:"balances"`
}

type rewardBalanceDTO struct {
	Type        string    `json:"type"`
	Balance     int64     `json:"balance"`
	EarnedTotal int64     `json:"earnedTotal"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func newRewardBalanceDTOs(balances []model.RewardBalance) []rewardBalanceDTO {
	result := make([]rewardBalanceDTO, len(balances))
	for index, balance := range balances {
		result[index] = rewardBalanceDTO{
			Type: balance.RewardType, Balance: balance.Balance,
			EarnedTotal: balance.EarnedTotal, UpdatedAt: balance.UpdatedAt,
		}
	}
	return result
}

type rewardWalletDTO struct {
	Balance  rewardBalanceDTO       `json:"balance"`
	Catalog  []rewardCatalogItemDTO `json:"catalog"`
	NextGoal *rewardGoalDTO         `json:"nextGoal"`
}

type rewardCatalogItemDTO struct {
	Code            string `json:"code"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	RewardType      string `json:"rewardType"`
	PerkType        string `json:"perkType"`
	Threshold       int64  `json:"threshold"`
	Unlocked        bool   `json:"unlocked"`
	ProgressCurrent int64  `json:"progressCurrent"`
	ProgressTarget  int64  `json:"progressTarget"`
	Remaining       int64  `json:"remaining"`
}

type rewardGoalDTO struct {
	Code       string `json:"code"`
	Title      string `json:"title"`
	RewardType string `json:"rewardType"`
	PerkType   string `json:"perkType"`
	Current    int64  `json:"current"`
	Target     int64  `json:"target"`
	Remaining  int64  `json:"remaining"`
}

func newRewardWalletDTO(wallet usecase.RewardWallet) rewardWalletDTO {
	catalog := make([]rewardCatalogItemDTO, len(wallet.Catalog))
	for index, item := range wallet.Catalog {
		catalog[index] = rewardCatalogItemDTO{
			Code: item.Item.Code, Title: item.Item.Title, Description: item.Item.Description,
			RewardType: item.Item.RewardType, PerkType: item.Item.PerkType,
			Threshold: item.Item.Threshold, Unlocked: item.Unlocked,
			ProgressCurrent: item.ProgressCurrent, ProgressTarget: item.ProgressTarget,
			Remaining: item.Remaining,
		}
	}
	return rewardWalletDTO{
		Balance:  rewardBalanceDTO{Type: wallet.Balance.RewardType, Balance: wallet.Balance.Balance, EarnedTotal: wallet.Balance.EarnedTotal, UpdatedAt: wallet.Balance.UpdatedAt},
		Catalog:  catalog,
		NextGoal: newRewardGoalDTO(wallet.NextGoal),
	}
}

func newRewardGoalDTO(goal *usecase.RewardGoal) *rewardGoalDTO {
	if goal == nil {
		return nil
	}
	return &rewardGoalDTO{
		Code: goal.Item.Code, Title: goal.Item.Title, RewardType: goal.Item.RewardType,
		PerkType: goal.Item.PerkType, Current: goal.Current, Target: goal.Target, Remaining: goal.Remaining,
	}
}

func newAchievementsDTO(items []model.AchievementProgress) achievementsDTO {
	result := make([]achievementDTO, len(items))
	for index, item := range items {
		result[index] = achievementDTO{
			Code: item.Achievement.Code, Title: item.Achievement.Title,
			Description: item.Achievement.Description, IconKey: item.Achievement.IconKey,
			Unlocked: item.UnlockedAt != nil, UnlockedAt: item.UnlockedAt,
		}
	}
	return achievementsDTO{Achievements: result}
}

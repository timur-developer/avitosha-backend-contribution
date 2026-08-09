package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
)

const dailyStreakBaseReward = 2

var streakMilestoneRewards = map[int]int{
	3:  4,
	7:  8,
	14: 14,
}

type RewardWallet struct {
	Balance  model.RewardBalance
	Catalog  []RewardCatalogEntry
	NextGoal *RewardGoal
}

type RewardCatalogEntry struct {
	Item            model.RewardCatalogItem
	Unlocked        bool
	ProgressCurrent int64
	ProgressTarget  int64
	Remaining       int64
}

type RewardGoal struct {
	Item      model.RewardCatalogItem
	Current   int64
	Target    int64
	Remaining int64
}

type RetentionOverview struct {
	Streak     StreakOverview
	DailyQuest DailyQuestOverview
	Tomorrow   TomorrowPreview
}

type StreakOverview struct {
	Current        int
	Longest        int
	LastActiveDate *time.Time
	ActiveToday    bool
	Reward         RewardOffer
}

type DailyQuestOverview struct {
	Date        time.Time
	Code        string
	Title       string
	Description string
	ActionType  model.ActionType
	Category    *string
	Progress    int
	Target      int
	Status      model.DailyQuestStatus
	Reward      RewardOffer
}

type RewardOffer struct {
	Type   string
	Amount int
	Source model.RewardSourceKind
}

type TomorrowPreview struct {
	Date              time.Time
	StreakAfterReturn int
	StreakReward      RewardOffer
	DailyQuest        DailyQuestPreview
	NextGoal          *RewardGoal
}

type DailyQuestPreview struct {
	Code        string
	Title       string
	Description string
	ActionType  model.ActionType
	Category    *string
	Target      int
	Reward      RewardOffer
}

type rewardCatalogUnlock struct {
	Code      string
	Title     string
	PerkType  string
	Threshold int64
}

type retentionState struct {
	Streak model.UserStreak
	Quest  model.DailyQuestProgress
}

func (service *GameService) GetRewardWallet(
	ctx context.Context,
	userID uuid.UUID,
	now time.Time,
) (RewardWallet, error) {
	if _, err := service.EnsureProfile(ctx, userID, now); err != nil {
		return RewardWallet{}, err
	}
	balances, err := service.repository.ListRewardBalances(ctx, userID)
	if err != nil {
		return RewardWallet{}, fmt.Errorf("list reward balances: %w", err)
	}
	balance := rewardBalanceByType(balances, DefaultRewardType)
	return service.buildRewardWallet(ctx, balance)
}

func (service *GameService) buildRetentionOverview(
	ctx context.Context,
	userID uuid.UUID,
	now time.Time,
) (RetentionOverview, error) {
	now = now.UTC()
	state, templates, err := service.ensureRetentionState(ctx, userID, now, false)
	if err != nil {
		return RetentionOverview{}, err
	}
	balances, err := service.repository.ListRewardBalances(ctx, userID)
	if err != nil {
		return RetentionOverview{}, fmt.Errorf("list reward balances for retention: %w", err)
	}
	wallet, err := service.buildRewardWallet(ctx, rewardBalanceByType(balances, DefaultRewardType))
	if err != nil {
		return RetentionOverview{}, err
	}

	today := utcDate(now)
	tomorrow := today.AddDate(0, 0, 1)
	tomorrowTemplate := selectDailyQuestTemplate(userID, tomorrow, templates)
	effectiveStreak := normalizedStreakForRead(state.Streak, today)
	projectedStreak := projectedTomorrowStreak(effectiveStreak, today)
	streakReward := streakRewardAmount(projectedStreak)

	return RetentionOverview{
		Streak: StreakOverview{
			Current: effectiveStreak.CurrentStreak, Longest: effectiveStreak.LongestStreak,
			LastActiveDate: cloneTimePointer(effectiveStreak.LastActiveDate),
			ActiveToday:    sameDatePointer(effectiveStreak.LastActiveDate, today),
			Reward: RewardOffer{
				Type: DefaultRewardType, Amount: streakRewardAmountForDisplay(effectiveStreak.CurrentStreak),
				Source: model.RewardSourceStreak,
			},
		},
		DailyQuest: DailyQuestOverview{
			Date: state.Quest.Quest.QuestDate, Code: state.Quest.Template.Code, Title: state.Quest.Template.Title,
			Description: state.Quest.Template.Description, ActionType: state.Quest.Template.ActionType,
			Category: state.Quest.Template.Category, Progress: state.Quest.Quest.Progress,
			Target: state.Quest.Quest.TargetValue, Status: state.Quest.Quest.Status,
			Reward: RewardOffer{
				Type: state.Quest.Quest.RewardType, Amount: state.Quest.Quest.RewardAmount,
				Source: model.RewardSourceDailyQuest,
			},
		},
		Tomorrow: TomorrowPreview{
			Date: tomorrow, StreakAfterReturn: projectedStreak,
			StreakReward: RewardOffer{Type: DefaultRewardType, Amount: streakReward, Source: model.RewardSourceStreak},
			DailyQuest: DailyQuestPreview{
				Code: tomorrowTemplate.Code, Title: tomorrowTemplate.Title, Description: tomorrowTemplate.Description,
				ActionType: tomorrowTemplate.ActionType, Category: tomorrowTemplate.Category,
				Target: tomorrowTemplate.TargetValue,
				Reward: RewardOffer{
					Type: tomorrowTemplate.RewardType, Amount: tomorrowTemplate.RewardAmount,
					Source: model.RewardSourceDailyQuest,
				},
			},
			NextGoal: wallet.NextGoal,
		},
	}, nil
}

func (service *GameService) ensureRetentionState(
	ctx context.Context,
	userID uuid.UUID,
	now time.Time,
	lockQuest bool,
) (retentionState, []model.DailyQuestTemplate, error) {
	streak, err := service.repository.GetOrCreateUserStreak(ctx, model.UserStreak{
		UserID: userID, CurrentStreak: 0, LongestStreak: 0, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return retentionState{}, nil, fmt.Errorf("ensure user streak: %w", err)
	}
	templates, err := service.repository.ListActiveDailyQuestTemplates(ctx)
	if err != nil {
		return retentionState{}, nil, fmt.Errorf("list daily quest templates: %w", err)
	}
	if len(templates) == 0 {
		return retentionState{}, nil, fmt.Errorf("list daily quest templates: %w", ErrUnexpectedStorage)
	}

	date := utcDate(now)
	if err := service.repository.ExpireDailyQuestsBefore(ctx, userID, date, now); err != nil {
		return retentionState{}, nil, fmt.Errorf("expire stale daily quests: %w", err)
	}
	quest, err := service.getDailyQuestProgress(ctx, userID, date, lockQuest)
	if errors.Is(err, ErrDailyQuestNotFound) {
		template := selectDailyQuestTemplate(userID, date, templates)
		assigned, assignErr := service.repository.AssignDailyQuest(ctx, model.UserDailyQuest{
			ID: service.idGenerator(), UserID: userID, QuestDate: date, TemplateCode: template.Code,
			TargetValue: template.TargetValue, Status: model.DailyQuestStatusActive,
			RewardType: template.RewardType, RewardAmount: template.RewardAmount,
			AssignedAt: now, CreatedAt: now, UpdatedAt: now,
		})
		if assignErr != nil {
			return retentionState{}, nil, fmt.Errorf("assign daily quest: %w", assignErr)
		}
		quest, err = service.getDailyQuestProgress(ctx, userID, assigned.QuestDate, lockQuest)
	}
	if err != nil {
		return retentionState{}, nil, fmt.Errorf("get daily quest progress: %w", err)
	}
	return retentionState{Streak: streak, Quest: quest}, templates, nil
}

func (service *GameService) applyRetentionForAction(
	ctx context.Context,
	actionID uuid.UUID,
	command ProcessActionCommand,
	state retentionState,
) (retentionState, []model.DomainEvent, error) {
	now := command.Now.UTC()
	today := utcDate(now)
	events := make([]model.DomainEvent, 0, 6)

	streakChanged, streakReset, streakReward := advanceStreak(&state.Streak, today, now)
	if streakChanged {
		if err := service.repository.UpdateUserStreak(ctx, state.Streak); err != nil {
			return retentionState{}, nil, fmt.Errorf("update streak: %w", err)
		}
		events = append(events, service.event(actionID, command.UserID, model.DomainEventStreakUpdated, now, map[string]any{
			"current":        state.Streak.CurrentStreak,
			"longest":        state.Streak.LongestStreak,
			"lastActiveDate": today.Format(time.DateOnly),
			"reset":          streakReset,
			"reward": map[string]any{
				"type": DefaultRewardType, "amount": streakReward,
			},
		}))
		if streakReward > 0 {
			title := fmt.Sprintf("Серия %d дней", state.Streak.CurrentStreak)
			rewardEvents, rewardErr := service.creditAdditionalReward(
				ctx, actionID, command.UserID, model.RewardCredit{
					ID: service.idGenerator(), UserID: command.UserID, ActionID: actionID,
					RewardType: DefaultRewardType, Amount: streakReward, SourceKind: model.RewardSourceStreak,
					SourceRef: today.Format(time.DateOnly), SourceTitle: &title, CreatedAt: now,
				}, now,
			)
			if rewardErr != nil {
				return retentionState{}, nil, rewardErr
			}
			events = append(events, rewardEvents...)
		}
	}

	if !dailyQuestMatchesAction(state.Quest.Template, command.ActionType, command.Category) ||
		state.Quest.Quest.Status != model.DailyQuestStatusActive {
		return state, events, nil
	}

	state.Quest.Quest.Progress = min(state.Quest.Quest.Progress+1, state.Quest.Quest.TargetValue)
	state.Quest.Quest.UpdatedAt = now
	payload := map[string]any{
		"code": state.Quest.Template.Code, "title": state.Quest.Template.Title,
		"progress": state.Quest.Quest.Progress, "target": state.Quest.Quest.TargetValue,
		"status": state.Quest.Quest.Status,
		"reward": map[string]any{
			"type": state.Quest.Quest.RewardType, "amount": state.Quest.Quest.RewardAmount,
		},
	}

	if state.Quest.Quest.Progress >= state.Quest.Quest.TargetValue {
		completedAt := now
		state.Quest.Quest.Status = model.DailyQuestStatusRewarded
		state.Quest.Quest.CompletedAt = &completedAt
		state.Quest.Quest.RewardedAt = &completedAt
		payload["status"] = state.Quest.Quest.Status
		events = append(events, service.event(actionID, command.UserID, model.DomainEventDailyQuestCompleted, now, map[string]any{
			"code": state.Quest.Template.Code, "title": state.Quest.Template.Title,
			"rewardType": state.Quest.Quest.RewardType, "rewardAmount": state.Quest.Quest.RewardAmount,
		}))

		title := state.Quest.Template.Title
		rewardEvents, rewardErr := service.creditAdditionalReward(
			ctx, actionID, command.UserID, model.RewardCredit{
				ID: service.idGenerator(), UserID: command.UserID, ActionID: actionID,
				RewardType: state.Quest.Quest.RewardType, Amount: state.Quest.Quest.RewardAmount,
				SourceKind: model.RewardSourceDailyQuest, SourceRef: state.Quest.Quest.ID.String(),
				SourceTitle: &title, CreatedAt: now,
			}, now,
		)
		if rewardErr != nil {
			return retentionState{}, nil, rewardErr
		}
		events = append(events, rewardEvents...)
	}

	if err := service.repository.UpdateDailyQuest(ctx, state.Quest.Quest); err != nil {
		return retentionState{}, nil, fmt.Errorf("update daily quest: %w", err)
	}
	events = append(events, service.event(actionID, command.UserID, model.DomainEventDailyQuestUpdated, now, payload))
	return state, events, nil
}

func (service *GameService) creditAdditionalReward(
	ctx context.Context,
	actionID uuid.UUID,
	userID uuid.UUID,
	credit model.RewardCredit,
	now time.Time,
) ([]model.DomainEvent, error) {
	balance, credited, err := service.repository.CreditReward(ctx, credit)
	if err != nil {
		return nil, fmt.Errorf("credit retention reward: %w", err)
	}
	if !credited {
		return nil, nil
	}
	return service.rewardEventsForCredit(ctx, actionID, userID, credit, balance, now)
}

func (service *GameService) getDailyQuestProgress(
	ctx context.Context,
	userID uuid.UUID,
	date time.Time,
	lock bool,
) (model.DailyQuestProgress, error) {
	if lock {
		return service.repository.GetDailyQuestProgressForUpdate(ctx, userID, date)
	}
	return service.repository.GetDailyQuestProgress(ctx, userID, date)
}

func (service *GameService) rewardEventsForCredit(
	ctx context.Context,
	actionID uuid.UUID,
	userID uuid.UUID,
	credit model.RewardCredit,
	balance model.RewardBalance,
	now time.Time,
) ([]model.DomainEvent, error) {
	wallet, err := service.buildRewardWallet(ctx, balance)
	if err != nil {
		return nil, err
	}

	unlocks := catalogUnlocksForCredit(wallet.Catalog, balance.EarnedTotal-int64(credit.Amount), balance.EarnedTotal)
	payload := map[string]any{
		"rewardType":  balance.RewardType,
		"amount":      credit.Amount,
		"balance":     balance.Balance,
		"earnedTotal": balance.EarnedTotal,
		"sourceKind":  credit.SourceKind,
		"sourceRef":   credit.SourceRef,
	}
	if credit.SourceTitle != nil {
		payload["sourceTitle"] = *credit.SourceTitle
	}
	if wallet.NextGoal != nil {
		payload["nextGoal"] = map[string]any{
			"code":      wallet.NextGoal.Item.Code,
			"title":     wallet.NextGoal.Item.Title,
			"current":   wallet.NextGoal.Current,
			"target":    wallet.NextGoal.Target,
			"remaining": wallet.NextGoal.Remaining,
		}
	}
	if len(unlocks) > 0 {
		items := make([]map[string]any, len(unlocks))
		for index, unlock := range unlocks {
			items[index] = map[string]any{
				"code":      unlock.Code,
				"title":     unlock.Title,
				"perkType":  unlock.PerkType,
				"threshold": unlock.Threshold,
			}
		}
		payload["catalogUnlocks"] = items
	}

	events := []model.DomainEvent{
		service.event(actionID, userID, model.DomainEventAvitoRewardEarned, now, payload),
	}
	for _, unlock := range unlocks {
		events = append(events, service.event(actionID, userID, model.DomainEventRewardCatalogUnlocked, now, map[string]any{
			"code":      unlock.Code,
			"title":     unlock.Title,
			"perkType":  unlock.PerkType,
			"threshold": unlock.Threshold,
		}))
	}
	return events, nil
}

func (service *GameService) buildRewardWallet(
	ctx context.Context,
	balance model.RewardBalance,
) (RewardWallet, error) {
	catalog, err := service.repository.ListRewardCatalog(ctx)
	if err != nil {
		return RewardWallet{}, fmt.Errorf("list reward catalog: %w", err)
	}
	entries := make([]RewardCatalogEntry, 0, len(catalog))
	var nextGoal *RewardGoal
	for _, item := range catalog {
		if item.RewardType != balance.RewardType {
			continue
		}
		current := min(balance.EarnedTotal, item.Threshold)
		entry := RewardCatalogEntry{
			Item: item, Unlocked: balance.EarnedTotal >= item.Threshold,
			ProgressCurrent: current, ProgressTarget: item.Threshold,
			Remaining: max(item.Threshold-balance.EarnedTotal, 0),
		}
		entries = append(entries, entry)
		if !entry.Unlocked && nextGoal == nil {
			nextGoal = &RewardGoal{
				Item: item, Current: current, Target: item.Threshold,
				Remaining: item.Threshold - balance.EarnedTotal,
			}
		}
	}
	return RewardWallet{Balance: balance, Catalog: entries, NextGoal: nextGoal}, nil
}

func rewardBalanceByType(balances []model.RewardBalance, rewardType string) model.RewardBalance {
	for _, balance := range balances {
		if balance.RewardType == rewardType {
			return balance
		}
	}
	return model.RewardBalance{RewardType: rewardType}
}

func selectDailyQuestTemplate(
	userID uuid.UUID,
	date time.Time,
	templates []model.DailyQuestTemplate,
) model.DailyQuestTemplate {
	indexSeed := date.YearDay()
	for _, part := range userID {
		indexSeed += int(part)
	}
	return templates[indexSeed%len(templates)]
}

func projectedTomorrowStreak(streak model.UserStreak, today time.Time) int {
	tomorrow := today.AddDate(0, 0, 1)
	copy := streak
	advanceStreak(&copy, tomorrow, tomorrow)
	return copy.CurrentStreak
}

func normalizedStreakForRead(streak model.UserStreak, today time.Time) model.UserStreak {
	if streak.LastActiveDate == nil {
		return streak
	}
	if sameDatePointer(streak.LastActiveDate, today) || sameDatePointer(streak.LastActiveDate, today.AddDate(0, 0, -1)) {
		return streak
	}
	normalized := streak
	normalized.CurrentStreak = 0
	return normalized
}

func advanceStreak(streak *model.UserStreak, today time.Time, now time.Time) (bool, bool, int) {
	if sameDatePointer(streak.LastActiveDate, today) {
		return false, false, 0
	}
	reset := false
	switch {
	case streak.LastActiveDate == nil:
		streak.CurrentStreak = 1
	case sameDatePointer(streak.LastActiveDate, today.AddDate(0, 0, -1)):
		streak.CurrentStreak++
	default:
		streak.CurrentStreak = 1
		reset = true
	}
	streak.LongestStreak = max(streak.LongestStreak, streak.CurrentStreak)
	activeDate := today
	streak.LastActiveDate = &activeDate
	streak.UpdatedAt = now
	return true, reset, streakRewardAmount(streak.CurrentStreak)
}

func streakRewardAmount(current int) int {
	reward := dailyStreakBaseReward
	if bonus, ok := streakMilestoneRewards[current]; ok {
		reward += bonus
	}
	return reward
}

func streakRewardAmountForDisplay(current int) int {
	if current <= 0 {
		return 0
	}
	return streakRewardAmount(current)
}

func dailyQuestMatchesAction(template model.DailyQuestTemplate, actionType model.ActionType, category *string) bool {
	if template.ActionType != actionType {
		return false
	}
	if template.Category == nil {
		return true
	}
	return equalStringPointers(template.Category, category)
}

func catalogUnlocksForCredit(
	catalog []RewardCatalogEntry,
	before int64,
	after int64,
) []rewardCatalogUnlock {
	result := make([]rewardCatalogUnlock, 0, 1)
	for _, item := range catalog {
		if before >= item.Item.Threshold || after < item.Item.Threshold {
			continue
		}
		result = append(result, rewardCatalogUnlock{
			Code: item.Item.Code, Title: item.Item.Title,
			PerkType: item.Item.PerkType, Threshold: item.Item.Threshold,
		})
	}
	return result
}

func sameDatePointer(value *time.Time, target time.Time) bool {
	return value != nil && value.UTC().Equal(target)
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := value.UTC()
	return &copy
}

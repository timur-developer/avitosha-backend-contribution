package usecase

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
)

const dailyStreakBaseReward = 2

const (
	dailyQuestRequired       = 2
	dailyGoalXPReward        = 30
	dailyGoalBonusReward     = 5
	dailyBalancedBonusReward = 3
)

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
	Streak    StreakOverview
	DailyGoal DailyGoalOverview
	Tomorrow  TomorrowPreview
}

type StreakOverview struct {
	Current        int
	Longest        int
	Protections    int
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
	Role        model.DailyQuestRole
	Category    *string
	Progress    int
	Target      int
	Status      model.DailyQuestStatus
	XPReward    int
}

type DailyGoalOverview struct {
	Date              time.Time
	Quests            []DailyQuestOverview
	Completed         int
	Required          int
	Status            model.DailyGoalStatus
	XPReward          int
	Reward            RewardOffer
	BalancedCompleted bool
	BalancedReward    RewardOffer
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
	TasksCount        int
	Required          int
	BuyerTasks        int
	SellerTasks       int
	UniversalTasks    int
	XPReward          int
	Reward            RewardOffer
	BalancedReward    RewardOffer
	NextGoal          *RewardGoal
}

type rewardCatalogUnlock struct {
	Code      string
	Title     string
	PerkType  string
	Threshold int64
}

type retentionState struct {
	Streak model.UserStreak
	Goal   model.UserDailyGoal
	Quests []model.DailyQuestProgress
}

type retentionActionResult struct {
	State    retentionState
	Events   []model.DomainEvent
	XPEarned int
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
	state, _, err := service.ensureRetentionState(ctx, userID, now, false)
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

	today := retentionDate(now)
	tomorrow := today.AddDate(0, 0, 1)
	effectiveStreak := normalizedStreakForRead(state.Streak, today)
	projectedStreak := projectedTomorrowStreak(effectiveStreak, today)
	streakReward := streakRewardAmount(projectedStreak)

	return RetentionOverview{
		Streak: StreakOverview{
			Current: effectiveStreak.CurrentStreak, Longest: effectiveStreak.LongestStreak,
			Protections:    effectiveStreak.ProtectionCount,
			LastActiveDate: cloneTimePointer(effectiveStreak.LastActiveDate),
			ActiveToday:    sameDatePointer(effectiveStreak.LastActiveDate, today),
			Reward: RewardOffer{
				Type: DefaultRewardType, Amount: streakRewardAmountForDisplay(effectiveStreak.CurrentStreak),
				Source: model.RewardSourceStreak,
			},
		},
		DailyGoal: DailyGoalOverview{
			Date: state.Goal.GoalDate, Quests: dailyQuestOverviews(state.Quests),
			Completed: state.Goal.CompletedCount, Required: state.Goal.RequiredCompleted,
			Status: state.Goal.Status, XPReward: state.Goal.XPReward,
			Reward:            RewardOffer{Type: state.Goal.RewardType, Amount: state.Goal.RewardAmount, Source: model.RewardSourceDailyGoal},
			BalancedCompleted: state.Goal.BalancedRewardedAt != nil,
			BalancedReward:    RewardOffer{Type: state.Goal.RewardType, Amount: state.Goal.BalancedRewardAmount, Source: model.RewardSourceBalancedDay},
		},
		Tomorrow: TomorrowPreview{
			Date: tomorrow, StreakAfterReturn: projectedStreak,
			StreakReward: RewardOffer{Type: DefaultRewardType, Amount: streakReward, Source: model.RewardSourceStreak},
			TasksCount:   5, Required: dailyQuestRequired, BuyerTasks: 2, SellerTasks: 2, UniversalTasks: 1,
			XPReward:       dailyGoalXPReward,
			Reward:         RewardOffer{Type: DefaultRewardType, Amount: dailyGoalBonusReward, Source: model.RewardSourceDailyGoal},
			BalancedReward: RewardOffer{Type: DefaultRewardType, Amount: dailyBalancedBonusReward, Source: model.RewardSourceBalancedDay},
			NextGoal:       wallet.NextGoal,
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

	date := retentionDate(now)
	if err := service.repository.ExpireDailyQuestsBefore(ctx, userID, date, now); err != nil {
		return retentionState{}, nil, fmt.Errorf("expire stale daily quests: %w", err)
	}
	goal, err := service.repository.GetOrCreateDailyGoal(ctx, model.UserDailyGoal{
		ID: service.idGenerator(), UserID: userID, GoalDate: date, RequiredCompleted: dailyQuestRequired,
		Status: model.DailyGoalStatusActive, XPReward: dailyGoalXPReward, RewardType: DefaultRewardType,
		RewardAmount: dailyGoalBonusReward, BalancedRewardAmount: dailyBalancedBonusReward,
		CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return retentionState{}, nil, fmt.Errorf("ensure daily goal: %w", err)
	}
	quests, err := service.getDailyQuestProgress(ctx, userID, date, lockQuest)
	if err != nil {
		return retentionState{}, nil, fmt.Errorf("list daily quest progress: %w", err)
	}
	assignedCodes := make(map[string]struct{}, len(quests))
	for _, quest := range quests {
		assignedCodes[quest.Template.Code] = struct{}{}
	}
	selected, err := selectDailyQuestSet(userID, date, templates)
	if err != nil {
		return retentionState{}, nil, err
	}
	for _, template := range selected {
		if _, exists := assignedCodes[template.Code]; exists {
			continue
		}
		_, assignErr := service.repository.AssignDailyQuest(ctx, model.UserDailyQuest{
			ID: service.idGenerator(), UserID: userID, QuestDate: date, TemplateCode: template.Code,
			TargetValue: template.TargetValue, Status: model.DailyQuestStatusActive,
			RewardType: template.RewardType, RewardAmount: template.RewardAmount,
			AssignedAt: now, CreatedAt: now, UpdatedAt: now,
		})
		if assignErr != nil {
			return retentionState{}, nil, fmt.Errorf("assign daily quest set: %w", assignErr)
		}
	}
	quests, err = service.getDailyQuestProgress(ctx, userID, date, lockQuest)
	if err != nil {
		return retentionState{}, nil, fmt.Errorf("reload daily quest set: %w", err)
	}
	goal.CompletedCount = completedDailyQuestCount(quests)
	return retentionState{Streak: streak, Goal: goal, Quests: quests}, templates, nil
}

func (service *GameService) applyRetentionForAction(
	ctx context.Context,
	actionID uuid.UUID,
	command ProcessActionCommand,
	state retentionState,
) (retentionActionResult, error) {
	now := command.Now.UTC()
	today := retentionDate(now)
	events := make([]model.DomainEvent, 0, 6)
	xpEarned := 0
	for index := range state.Quests {
		quest := &state.Quests[index]
		if quest.Quest.Status != model.DailyQuestStatusActive ||
			rewardedOnRetentionDate(quest.Quest.RewardedAt, today) ||
			!dailyQuestMatchesAction(quest.Template, command.ActionType, command.Category) {
			continue
		}
		quest.Quest.Progress = min(quest.Quest.Progress+1, quest.Quest.TargetValue)
		quest.Quest.UpdatedAt = now
		if quest.Quest.Progress >= quest.Quest.TargetValue {
			completedAt := now
			quest.Quest.Status = model.DailyQuestStatusRewarded
			quest.Quest.CompletedAt, quest.Quest.RewardedAt = &completedAt, &completedAt
			xpEarned += quest.Template.XPReward
			events = append(events, service.event(actionID, command.UserID, model.DomainEventDailyQuestCompleted, now, map[string]any{
				"code": quest.Template.Code, "title": quest.Template.Title, "role": quest.Template.Role, "xpReward": quest.Template.XPReward,
			}))
		}
		if err := service.repository.UpdateDailyQuest(ctx, quest.Quest); err != nil {
			return retentionActionResult{}, fmt.Errorf("update daily quest: %w", err)
		}
		events = append(events, service.event(actionID, command.UserID, model.DomainEventDailyQuestUpdated, now, map[string]any{
			"code": quest.Template.Code, "role": quest.Template.Role, "progress": quest.Quest.Progress,
			"target": quest.Quest.TargetValue, "status": quest.Quest.Status,
		}))
	}

	state.Goal.CompletedCount = completedDailyQuestCount(state.Quests)
	if state.Goal.Status == model.DailyGoalStatusActive &&
		!rewardedOnRetentionDate(state.Goal.RewardedAt, today) &&
		state.Goal.CompletedCount >= state.Goal.RequiredCompleted {
		completedAt := now
		state.Goal.Status, state.Goal.RewardedAt = model.DailyGoalStatusRewarded, &completedAt
		xpEarned += state.Goal.XPReward
		events = append(events, service.event(actionID, command.UserID, model.DomainEventDailyGoalCompleted, now, map[string]any{
			"completed": state.Goal.CompletedCount, "required": state.Goal.RequiredCompleted,
			"xpReward": state.Goal.XPReward, "rewardType": state.Goal.RewardType, "rewardAmount": state.Goal.RewardAmount,
		}))
		rewardEvents, err := service.creditGoalReward(ctx, actionID, command.UserID, state.Goal, model.RewardSourceDailyGoal, state.Goal.RewardAmount, "Дневная цель", now)
		if err != nil {
			return retentionActionResult{}, err
		}
		events = append(events, rewardEvents...)
		streakEvents, err := service.completeDailyStreak(ctx, actionID, command.UserID, &state.Streak, today, now)
		if err != nil {
			return retentionActionResult{}, err
		}
		events = append(events, streakEvents...)
	}
	if !rewardedOnRetentionDate(state.Goal.BalancedRewardedAt, today) && completedBuyerAndSeller(state.Quests) {
		balancedAt := now
		state.Goal.BalancedRewardedAt = &balancedAt
		events = append(events, service.event(actionID, command.UserID, model.DomainEventBalancedDayCompleted, now, map[string]any{
			"rewardType": state.Goal.RewardType, "rewardAmount": state.Goal.BalancedRewardAmount,
		}))
		rewardEvents, err := service.creditGoalReward(ctx, actionID, command.UserID, state.Goal, model.RewardSourceBalancedDay, state.Goal.BalancedRewardAmount, "Сбалансированный день", now)
		if err != nil {
			return retentionActionResult{}, err
		}
		events = append(events, rewardEvents...)
	}
	state.Goal.UpdatedAt = now
	if err := service.repository.UpdateDailyGoal(ctx, state.Goal); err != nil {
		return retentionActionResult{}, fmt.Errorf("update daily goal: %w", err)
	}
	return retentionActionResult{State: state, Events: events, XPEarned: xpEarned}, nil
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
) ([]model.DailyQuestProgress, error) {
	if lock {
		return service.repository.ListDailyQuestProgressForUpdate(ctx, userID, date)
	}
	return service.repository.ListDailyQuestProgress(ctx, userID, date)
}

func (service *GameService) creditGoalReward(
	ctx context.Context, actionID, userID uuid.UUID, goal model.UserDailyGoal,
	source model.RewardSourceKind, amount int, title string, now time.Time,
) ([]model.DomainEvent, error) {
	return service.creditAdditionalReward(ctx, actionID, userID, model.RewardCredit{
		ID: service.idGenerator(), UserID: userID, ActionID: actionID, RewardType: goal.RewardType,
		Amount: amount, SourceKind: source, SourceRef: goal.ID.String(), SourceTitle: &title, CreatedAt: now,
	}, now)
}

func (service *GameService) completeDailyStreak(
	ctx context.Context, actionID, userID uuid.UUID, streak *model.UserStreak, today, now time.Time,
) ([]model.DomainEvent, error) {
	changed, reset, reward, protectionUsed, protectionEarned := advanceStreak(streak, today, now)
	if !changed {
		return nil, nil
	}
	if err := service.repository.UpdateUserStreak(ctx, *streak); err != nil {
		return nil, fmt.Errorf("update streak: %w", err)
	}
	events := []model.DomainEvent{service.event(actionID, userID, model.DomainEventStreakUpdated, now, map[string]any{
		"current": streak.CurrentStreak, "longest": streak.LongestStreak,
		"lastActiveDate": today.Format(time.DateOnly), "reset": reset,
		"protections": streak.ProtectionCount, "protectionUsed": protectionUsed, "protectionEarned": protectionEarned,
		"reward": map[string]any{"type": DefaultRewardType, "amount": reward},
	})}
	title := fmt.Sprintf("Серия %d дней", streak.CurrentStreak)
	rewardEvents, err := service.creditAdditionalReward(ctx, actionID, userID, model.RewardCredit{
		ID: service.idGenerator(), UserID: userID, ActionID: actionID, RewardType: DefaultRewardType,
		Amount: reward, SourceKind: model.RewardSourceStreak, SourceRef: today.Format(time.DateOnly),
		SourceTitle: &title, CreatedAt: now,
	}, now)
	if err != nil {
		return nil, err
	}
	return append(events, rewardEvents...), nil
}

func dailyQuestOverviews(items []model.DailyQuestProgress) []DailyQuestOverview {
	result := make([]DailyQuestOverview, 0, len(items))
	for _, item := range items {
		result = append(result, DailyQuestOverview{
			Date: item.Quest.QuestDate, Code: item.Template.Code, Title: item.Template.Title,
			Description: item.Template.Description, ActionType: item.Template.ActionType, Role: item.Template.Role,
			Category: item.Template.Category, Progress: item.Quest.Progress, Target: item.Quest.TargetValue,
			Status: item.Quest.Status, XPReward: item.Template.XPReward,
		})
	}
	return result
}

func completedDailyQuestCount(items []model.DailyQuestProgress) int {
	count := 0
	for _, item := range items {
		if item.Quest.Status == model.DailyQuestStatusCompleted || item.Quest.Status == model.DailyQuestStatusRewarded {
			count++
		}
	}
	return count
}

func completedBuyerAndSeller(items []model.DailyQuestProgress) bool {
	buyer, seller := false, false
	for _, item := range items {
		if item.Quest.Status != model.DailyQuestStatusCompleted && item.Quest.Status != model.DailyQuestStatusRewarded {
			continue
		}
		buyer = buyer || item.Template.Role == model.DailyQuestRoleBuyer
		seller = seller || item.Template.Role == model.DailyQuestRoleSeller
	}
	return buyer && seller
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

func selectDailyQuestSet(
	userID uuid.UUID,
	date time.Time,
	templates []model.DailyQuestTemplate,
) ([]model.DailyQuestTemplate, error) {
	seed := date.Year()*1000 + date.YearDay()
	for _, part := range userID {
		seed += int(part)
	}
	result := make([]model.DailyQuestTemplate, 0, 5)
	for index, spec := range []struct {
		role  model.DailyQuestRole
		count int
	}{
		{model.DailyQuestRoleBuyer, 2}, {model.DailyQuestRoleSeller, 2}, {model.DailyQuestRoleUniversal, 1},
	} {
		candidates := make([]model.DailyQuestTemplate, 0)
		for _, template := range templates {
			if template.Role == spec.role {
				candidates = append(candidates, template)
			}
		}
		sort.Slice(candidates, func(i, j int) bool { return candidates[i].SortOrder < candidates[j].SortOrder })
		selected := rotatedUniqueActionTemplates(candidates, spec.count, seed+index*17)
		if len(selected) != spec.count {
			return nil, fmt.Errorf("daily quest pool for %s: %w", spec.role, ErrUnexpectedStorage)
		}
		result = append(result, selected...)
	}
	return result, nil
}

func rotatedUniqueActionTemplates(candidates []model.DailyQuestTemplate, count, seed int) []model.DailyQuestTemplate {
	result := make([]model.DailyQuestTemplate, 0, count)
	usedActions := make(map[model.ActionType]struct{}, count)
	if len(candidates) == 0 {
		return result
	}
	start := seed % len(candidates)
	for offset := 0; offset < len(candidates) && len(result) < count; offset++ {
		candidate := candidates[(start+offset)%len(candidates)]
		if _, exists := usedActions[candidate.ActionType]; exists {
			continue
		}
		usedActions[candidate.ActionType] = struct{}{}
		result = append(result, candidate)
	}
	return result
}

func retentionDate(value time.Time) time.Time {
	moscow := time.FixedZone("Europe/Moscow", 3*60*60)
	local := value.In(moscow)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
}

func rewardedOnRetentionDate(rewardedAt *time.Time, date time.Time) bool {
	return rewardedAt != nil && retentionDate(*rewardedAt).Equal(date)
}

func dailyQuestCompletedForActionOnDate(
	quests []model.DailyQuestProgress,
	command ProcessActionCommand,
	date time.Time,
) bool {
	for _, quest := range quests {
		if !quest.Quest.QuestDate.Equal(date) ||
			!dailyQuestMatchesAction(quest.Template, command.ActionType, command.Category) {
			continue
		}
		if quest.Quest.Status == model.DailyQuestStatusRewarded ||
			rewardedOnRetentionDate(quest.Quest.RewardedAt, date) {
			return true
		}
	}
	return false
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
	if streak.ProtectionCount > 0 && sameDatePointer(streak.LastActiveDate, today.AddDate(0, 0, -2)) {
		return streak
	}
	normalized := streak
	normalized.CurrentStreak = 0
	return normalized
}

func advanceStreak(streak *model.UserStreak, today time.Time, now time.Time) (bool, bool, int, bool, bool) {
	if sameDatePointer(streak.LastActiveDate, today) {
		return false, false, 0, false, false
	}
	reset := false
	protectionUsed := false
	switch {
	case streak.LastActiveDate == nil:
		streak.CurrentStreak = 1
	case sameDatePointer(streak.LastActiveDate, today.AddDate(0, 0, -1)):
		streak.CurrentStreak++
	case sameDatePointer(streak.LastActiveDate, today.AddDate(0, 0, -2)) && streak.ProtectionCount > 0:
		streak.CurrentStreak++
		streak.ProtectionCount--
		protectionUsed = true
	default:
		streak.CurrentStreak = 1
		reset = true
	}
	protectionEarned := streak.CurrentStreak%7 == 0
	if protectionEarned {
		streak.ProtectionCount++
	}
	streak.LongestStreak = max(streak.LongestStreak, streak.CurrentStreak)
	activeDate := today
	streak.LastActiveDate = &activeDate
	streak.UpdatedAt = now
	return true, reset, streakRewardAmount(streak.CurrentStreak), protectionUsed, protectionEarned
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

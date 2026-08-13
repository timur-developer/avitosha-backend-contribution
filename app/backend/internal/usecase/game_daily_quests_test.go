package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
)

func TestSelectDailyQuestSetUsesTwoBuyerTwoSellerAndOneUniversal(t *testing.T) {
	templates := []model.DailyQuestTemplate{
		{Code: "BUY_VIEW", Role: model.DailyQuestRoleBuyer, ActionType: model.ActionTypeAdViewed, SortOrder: 1},
		{Code: "BUY_FAVORITE", Role: model.DailyQuestRoleBuyer, ActionType: model.ActionTypeAdFavorited, SortOrder: 2},
		{Code: "BUY_CONTACT", Role: model.DailyQuestRoleBuyer, ActionType: model.ActionTypeMessageSent, SortOrder: 3},
		{Code: "SELL_CREATE", Role: model.DailyQuestRoleSeller, ActionType: model.ActionTypeAdCreated, SortOrder: 4},
		{Code: "SELL_IMPROVE", Role: model.DailyQuestRoleSeller, ActionType: model.ActionTypeListingImproved, SortOrder: 5},
		{Code: "ANY_DELIVERY", Role: model.DailyQuestRoleUniversal, ActionType: model.ActionTypeDeliveryUsed, SortOrder: 6},
	}
	set, err := selectDailyQuestSet(uuid.MustParse("00000000-0000-0000-0000-000000000001"), time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC), templates)
	if err != nil {
		t.Fatalf("select set: %v", err)
	}
	counts := map[model.DailyQuestRole]int{}
	actions := map[model.ActionType]bool{}
	for _, quest := range set {
		counts[quest.Role]++
		if actions[quest.ActionType] {
			t.Fatalf("duplicate action type %s in set", quest.ActionType)
		}
		actions[quest.ActionType] = true
	}
	if len(set) != 5 || counts[model.DailyQuestRoleBuyer] != 2 || counts[model.DailyQuestRoleSeller] != 2 || counts[model.DailyQuestRoleUniversal] != 1 {
		t.Fatalf("set = %+v, counts = %+v", set, counts)
	}
}

func TestRetentionDateChangesAtMoscowMidnight(t *testing.T) {
	before := retentionDate(time.Date(2026, 8, 11, 20, 59, 59, 0, time.UTC))
	after := retentionDate(time.Date(2026, 8, 11, 21, 0, 0, 0, time.UTC))
	if before.Format(time.DateOnly) != "2026-08-11" || after.Format(time.DateOnly) != "2026-08-12" {
		t.Fatalf("before=%s after=%s", before.Format(time.DateOnly), after.Format(time.DateOnly))
	}
}

func TestRewardedOnRetentionDateUsesMoscowCalendarDay(t *testing.T) {
	today := retentionDate(time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC))
	rewardedToday := time.Date(2026, 8, 11, 21, 30, 0, 0, time.UTC)
	rewardedYesterday := time.Date(2026, 8, 11, 20, 30, 0, 0, time.UTC)

	if !rewardedOnRetentionDate(&rewardedToday, today) {
		t.Fatal("reward granted after Moscow midnight must belong to today")
	}
	if rewardedOnRetentionDate(&rewardedYesterday, today) {
		t.Fatal("reward granted before Moscow midnight must not belong to today")
	}
	if rewardedOnRetentionDate(nil, today) {
		t.Fatal("missing reward timestamp must not be treated as rewarded today")
	}
}

func TestApplyRetentionForActionDoesNotRewardQuestTwiceOnSameDay(t *testing.T) {
	userID := uuid.New()
	repository := newGameTestRepository(userID)
	service := NewGameService(GameServiceDependencies{
		Repository:  repository,
		TxManager:   &gameTestTxManager{},
		IDGenerator: uuid.New,
	})
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	rewardedAt := now.Add(-time.Hour)
	questID := uuid.New()
	state := retentionState{
		Goal: model.UserDailyGoal{
			ID: uuid.New(), UserID: userID, GoalDate: retentionDate(now),
			RequiredCompleted: 2, Status: model.DailyGoalStatusActive,
		},
		Quests: []model.DailyQuestProgress{{
			Template: model.DailyQuestTemplate{
				Code: "DAILY_VIEW", ActionType: model.ActionTypeAdViewed,
				Role: model.DailyQuestRoleBuyer, TargetValue: 1, XPReward: 10,
			},
			Quest: model.UserDailyQuest{
				ID: questID, UserID: userID, QuestDate: retentionDate(now),
				TemplateCode: "DAILY_VIEW", TargetValue: 1,
				Status: model.DailyQuestStatusActive, RewardedAt: &rewardedAt,
			},
		}},
	}

	result, err := service.applyRetentionForAction(context.Background(), uuid.New(), ProcessActionCommand{
		UserID: userID, ActionType: model.ActionTypeAdViewed, Now: now,
	}, state)
	if err != nil {
		t.Fatalf("apply retention: %v", err)
	}
	if result.XPEarned != 0 || len(result.Events) != 0 {
		t.Fatalf("duplicate daily reward result = %+v", result)
	}
	if result.State.Quests[0].Quest.Progress != 0 {
		t.Fatalf("rewarded quest progressed again: %+v", result.State.Quests[0].Quest)
	}
}

func TestDailyQuestCompletedForActionOnDateStopsProductXP(t *testing.T) {
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	today := retentionDate(now)
	rewardedAt := now.Add(-time.Hour)

	for _, actionType := range []model.ActionType{
		model.ActionTypeAdFavorited,
		model.ActionTypeAdCreated,
	} {
		quests := []model.DailyQuestProgress{{
			Template: model.DailyQuestTemplate{ActionType: actionType},
			Quest: model.UserDailyQuest{
				QuestDate: today, Status: model.DailyQuestStatusRewarded,
				RewardedAt: &rewardedAt,
			},
		}}
		if !dailyQuestCompletedForActionOnDate(quests, ProcessActionCommand{
			ActionType: actionType,
		}, today) {
			t.Fatalf("completed %s daily quest did not stop product XP", actionType)
		}
	}
}

func TestAdvanceStreakEarnsAndConsumesOneDayProtection(t *testing.T) {
	lastActive := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	streak := model.UserStreak{CurrentStreak: 6, LongestStreak: 6, LastActiveDate: &lastActive}
	changed, reset, _, used, earned := advanceStreak(&streak, lastActive.AddDate(0, 0, 1), lastActive.AddDate(0, 0, 1))
	if !changed || reset || used || !earned || streak.CurrentStreak != 7 || streak.ProtectionCount != 1 {
		t.Fatalf("earned protection streak = %+v", streak)
	}
	changed, reset, _, used, earned = advanceStreak(&streak, lastActive.AddDate(0, 0, 3), lastActive.AddDate(0, 0, 3))
	if !changed || reset || !used || earned || streak.CurrentStreak != 8 || streak.ProtectionCount != 0 {
		t.Fatalf("consumed protection streak = %+v", streak)
	}
}

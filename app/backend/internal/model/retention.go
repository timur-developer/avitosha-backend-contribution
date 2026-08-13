package model

import (
	"time"

	"github.com/google/uuid"
)

type UserStreak struct {
	UserID          uuid.UUID
	CurrentStreak   int
	LongestStreak   int
	ProtectionCount int
	LastActiveDate  *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type DailyQuestStatus string

type DailyQuestRole string

const (
	DailyQuestStatusActive    DailyQuestStatus = "ACTIVE"
	DailyQuestStatusCompleted DailyQuestStatus = "COMPLETED"
	DailyQuestStatusRewarded  DailyQuestStatus = "REWARDED"
	DailyQuestStatusExpired   DailyQuestStatus = "EXPIRED"
)

const (
	DailyQuestRoleBuyer     DailyQuestRole = "BUYER"
	DailyQuestRoleSeller    DailyQuestRole = "SELLER"
	DailyQuestRoleUniversal DailyQuestRole = "UNIVERSAL"
)

type DailyQuestTemplate struct {
	Code         string
	Title        string
	Description  string
	ActionType   ActionType
	Role         DailyQuestRole
	Category     *string
	TargetValue  int
	XPReward     int
	RewardType   string
	RewardAmount int
	SortOrder    int
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserDailyQuest struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	QuestDate    time.Time
	TemplateCode string
	Progress     int
	TargetValue  int
	Status       DailyQuestStatus
	RewardType   string
	RewardAmount int
	AssignedAt   time.Time
	CompletedAt  *time.Time
	RewardedAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type DailyQuestProgress struct {
	Template DailyQuestTemplate
	Quest    UserDailyQuest
}

type DailyGoalStatus string

const (
	DailyGoalStatusActive   DailyGoalStatus = "ACTIVE"
	DailyGoalStatusRewarded DailyGoalStatus = "REWARDED"
	DailyGoalStatusExpired  DailyGoalStatus = "EXPIRED"
)

type UserDailyGoal struct {
	ID                   uuid.UUID
	UserID               uuid.UUID
	GoalDate             time.Time
	RequiredCompleted    int
	CompletedCount       int
	Status               DailyGoalStatus
	XPReward             int
	RewardType           string
	RewardAmount         int
	BalancedRewardAmount int
	RewardedAt           *time.Time
	BalancedRewardedAt   *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

package model

import (
	"time"

	"github.com/google/uuid"
)

type UserStreak struct {
	UserID         uuid.UUID
	CurrentStreak  int
	LongestStreak  int
	LastActiveDate *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type DailyQuestStatus string

const (
	DailyQuestStatusActive    DailyQuestStatus = "ACTIVE"
	DailyQuestStatusCompleted DailyQuestStatus = "COMPLETED"
	DailyQuestStatusRewarded  DailyQuestStatus = "REWARDED"
	DailyQuestStatusExpired   DailyQuestStatus = "EXPIRED"
)

type DailyQuestTemplate struct {
	Code         string
	Title        string
	Description  string
	ActionType   ActionType
	Category     *string
	TargetValue  int
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

package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type DomainEventType string

const (
	DomainEventTaskProgressUpdated     DomainEventType = "TASK_PROGRESS_UPDATED"
	DomainEventTaskCompleted           DomainEventType = "TASK_COMPLETED"
	DomainEventXPEarned                DomainEventType = "XP_EARNED"
	DomainEventPetLevelUp              DomainEventType = "PET_LEVEL_UP"
	DomainEventPetMoodChanged          DomainEventType = "PET_MOOD_CHANGED"
	DomainEventRoomItemUnlocked        DomainEventType = "ROOM_ITEM_UNLOCKED"
	DomainEventStoryStageCompleted     DomainEventType = "STORY_STAGE_COMPLETED"
	DomainEventStoryCompleted          DomainEventType = "STORY_COMPLETED"
	DomainEventLeaderboardScoreUpdated DomainEventType = "LEADERBOARD_SCORE_UPDATED"
	DomainEventAchievementUnlocked     DomainEventType = "ACHIEVEMENT_UNLOCKED"
	DomainEventPetCharacterUnlocked    DomainEventType = "PET_CHARACTER_UNLOCKED"
	DomainEventAvitoRewardEarned       DomainEventType = "AVITO_REWARD_EARNED"
	DomainEventRewardCatalogUnlocked   DomainEventType = "REWARD_CATALOG_UNLOCKED"
	DomainEventDailyQuestUpdated       DomainEventType = "DAILY_QUEST_UPDATED"
	DomainEventDailyQuestCompleted     DomainEventType = "DAILY_QUEST_COMPLETED"
	DomainEventStreakUpdated           DomainEventType = "STREAK_UPDATED"
)

type DomainEvent struct {
	ID         uuid.UUID       `json:"id"`
	Type       DomainEventType `json:"type"`
	UserID     uuid.UUID       `json:"-"`
	ActionID   uuid.UUID       `json:"-"`
	OccurredAt time.Time       `json:"occurredAt"`
	Payload    json.RawMessage `json:"payload"`
}

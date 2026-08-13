package handler

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

var benchmarkActionResultJSON []byte

func BenchmarkMarshalActionResultDTO(b *testing.B) {
	result := benchmarkActionResult()
	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		body, err := json.Marshal(newActionResultDTO(result))
		if err != nil {
			b.Fatal(err)
		}
		benchmarkActionResultJSON = body
	}
}

func benchmarkActionResult() usecase.ProcessActionResult {
	userID := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	actionID := uuid.MustParse("20000000-0000-0000-0000-000000000001")
	now := time.Date(2026, 8, 8, 12, 0, 0, 123456789, time.UTC)
	payloads := []struct {
		eventType model.DomainEventType
		payload   string
	}{
		{model.DomainEventTaskProgressUpdated, `{"taskId":"30000000-0000-0000-0000-000000000001","taskCode":"VIEW_FURNITURE_ADS","progress":5,"target":5}`},
		{model.DomainEventTaskCompleted, `{"taskId":"30000000-0000-0000-0000-000000000001","taskCode":"VIEW_FURNITURE_ADS"}`},
		{model.DomainEventXPEarned, `{"amount":30,"totalXp":130}`},
		{model.DomainEventPetLevelUp, `{"previousLevel":1,"level":2}`},
		{model.DomainEventRoomItemUnlocked, `{"itemCode":"DESK"}`},
		{model.DomainEventStoryStageCompleted, `{"storyCode":"FIRST_ROOM","stage":1}`},
		{model.DomainEventLeaderboardScoreUpdated, `{"score":100,"delta":100}`},
		{model.DomainEventAvitoRewardEarned, `{"rewardType":"AVITO_BONUS","amount":10,"balance":20,"earnedTotal":20,"sourceKind":"TASK_COMPLETION","sourceRef":"VIEW_FURNITURE_ADS"}`},
	}

	events := make([]model.DomainEvent, len(payloads))
	for index, item := range payloads {
		events[index] = model.DomainEvent{
			ID:   uuid.NewSHA1(uuid.NameSpaceOID, []byte{byte(index)}),
			Type: item.eventType, UserID: userID, ActionID: actionID,
			OccurredAt: now, Payload: json.RawMessage(item.payload),
		}
	}
	return usecase.ProcessActionResult{ActionID: actionID, Events: events}
}

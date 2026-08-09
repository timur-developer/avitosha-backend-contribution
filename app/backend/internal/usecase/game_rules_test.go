package usecase

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
)

func TestApplyTaskProgressCapsAndCompletesOnce(t *testing.T) {
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	progress := model.UserTask{
		ID: uuid.New(), Progress: 4, TargetValue: 5, Status: model.TaskStatusActive,
	}

	completed, justCompleted := ApplyTaskProgress(progress, now)
	if !justCompleted || completed.Progress != 5 || completed.Status != model.TaskStatusCompleted {
		t.Fatalf("completed progress = %+v, justCompleted = %v", completed, justCompleted)
	}
	rewarded := RewardTask(completed, now)
	if rewarded.Status != model.TaskStatusRewarded || rewarded.RewardedAt == nil {
		t.Fatalf("rewarded progress = %+v", rewarded)
	}
	unchanged, completedAgain := ApplyTaskProgress(rewarded, now.Add(time.Minute))
	if completedAgain || unchanged.Progress != 5 || unchanged.RewardedAt == nil {
		t.Fatalf("duplicate progress = %+v, completedAgain = %v", unchanged, completedAgain)
	}
}

func TestCalculateLevelUsesProductThresholds(t *testing.T) {
	for xp, want := range map[int]int{
		0: 1, 99: 1, 100: 2, 249: 2, 250: 3, 449: 3, 450: 4, 699: 4, 700: 5, 1200: 5,
	} {
		got, err := CalculateLevel(xp)
		if err != nil || got != want {
			t.Fatalf("CalculateLevel(%d) = %d, %v; want %d", xp, got, err, want)
		}
	}
}

func TestWeeklyScoreAndMondayBoundary(t *testing.T) {
	delta := WeeklyProgressDelta{EarnedXP: 30, CompletedTasks: 1, CompletedStages: 1}
	if score := WeeklyScore(delta); score != 100 {
		t.Fatalf("WeeklyScore() = %d, want 100", score)
	}
	wednesday := time.Date(2026, 8, 5, 19, 0, 0, 0, time.FixedZone("UTC+3", 3*60*60))
	want := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	if got := WeekStart(wednesday); !got.Equal(want) {
		t.Fatalf("WeekStart() = %v, want %v", got, want)
	}
}

func TestCharacterUnlockUsesStableHighestScore(t *testing.T) {
	locked, progress := CharacterFromScores(model.ActivityScores{BuyerScore: 4, SellerScore: 3})
	if locked != nil || progress != 4 {
		t.Fatalf("locked character = %v, progress = %d", locked, progress)
	}
	unlocked, progress := CharacterFromScores(model.ActivityScores{BuyerScore: 5, SellerScore: 5})
	if unlocked == nil || *unlocked != model.PetCharacterExplorer || progress != 5 {
		t.Fatalf("unlocked character = %v, progress = %d", unlocked, progress)
	}
}

func TestBuildCharacterProfileDescribesLockedAndUnlockedState(t *testing.T) {
	locked := BuildCharacterProfile(model.Pet{}, model.ActivityScores{SellerScore: 3})
	if locked.Code != model.PetCharacterEntrepreneur || locked.Name != "Предприниматель" ||
		locked.Progress != 3 || locked.Unlocked {
		t.Fatalf("locked profile = %+v", locked)
	}
	character := model.PetCharacterTraveler
	unlocked := BuildCharacterProfile(
		model.Pet{Character: &character},
		model.ActivityScores{BuyerScore: 8, TravelScore: 5},
	)
	if unlocked.Code != model.PetCharacterTraveler || unlocked.VisualDetail != "suitcase-badge" || !unlocked.Unlocked {
		t.Fatalf("unlocked profile = %+v", unlocked)
	}
}

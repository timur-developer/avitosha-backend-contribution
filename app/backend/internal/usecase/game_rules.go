package usecase

import (
	"strings"
	"time"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
)

const (
	FirstRoomStoryCode      = "FIRST_ROOM"
	InitialRoomItemCode     = "BOX"
	CharacterUnlockTarget   = 5
	DefaultLeaderboardLimit = 10
)

func NextLevelXP(level int) *int {
	thresholds := map[int]int{1: 100, 2: 250, 3: 450, 4: 700}
	threshold, ok := thresholds[level]
	if !ok {
		return nil
	}
	return &threshold
}

func ApplyTaskProgress(progress model.UserTask, now time.Time) (model.UserTask, bool) {
	if progress.Status != model.TaskStatusActive || progress.Progress >= progress.TargetValue {
		return progress, false
	}

	progress.Progress = min(progress.Progress+1, progress.TargetValue)
	progress.UpdatedAt = now
	if progress.Progress < progress.TargetValue {
		return progress, false
	}

	completedAt := now
	progress.Status = model.TaskStatusCompleted
	progress.CompletedAt = &completedAt
	return progress, true
}

func RewardTask(progress model.UserTask, now time.Time) model.UserTask {
	if progress.Status != model.TaskStatusCompleted || progress.RewardedAt != nil {
		return progress
	}
	rewardedAt := now
	progress.Status = model.TaskStatusRewarded
	progress.RewardedAt = &rewardedAt
	progress.UpdatedAt = now
	return progress
}

func WeeklyScore(delta WeeklyProgressDelta) int {
	return delta.EarnedXP + delta.CompletedTasks*20 + delta.CompletedStages*50
}

func WeekStart(value time.Time) time.Time {
	date := utcDate(value)
	daysSinceMonday := (int(date.Weekday()) + 6) % 7
	return date.AddDate(0, 0, -daysSinceMonday)
}

func utcDate(value time.Time) time.Time {
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func ActivityDelta(actionType model.ActionType, category *string) ActivityScoreDelta {
	var delta ActivityScoreDelta
	switch actionType {
	case model.ActionTypeAdViewed, model.ActionTypeAdFavorited, model.ActionTypeMessageSent,
		model.ActionTypeDeliveryUsed, model.ActionTypeReviewLeft, model.ActionTypeBookingCreated:
		delta.Buyer = 1
	case model.ActionTypeAdCreated:
		delta.Seller = 1
	}

	if category == nil {
		return delta
	}
	switch strings.ToUpper(strings.TrimSpace(*category)) {
	case "AUTO":
		delta.Auto++
	case "TRAVEL":
		delta.Travel++
	case "REAL_ESTATE":
		delta.RealEstate++
	case "SERVICES":
		delta.Services++
	}
	return delta
}

func CharacterFromScores(scores model.ActivityScores) (*model.PetCharacter, int) {
	character, score := leadingCharacter(scores)
	if score < CharacterUnlockTarget {
		return nil, score
	}
	return &character, score
}

func BuildCharacterProfile(pet model.Pet, scores model.ActivityScores) CharacterProfile {
	candidate, progress := leadingCharacter(scores)
	unlocked := pet.Character != nil
	if unlocked {
		candidate = *pet.Character
	}
	profile := CharacterProfile{
		Code: candidate, Progress: min(progress, CharacterUnlockTarget),
		Target: CharacterUnlockTarget, Unlocked: unlocked,
	}
	switch candidate {
	case model.PetCharacterEntrepreneur:
		profile.Name = "Предприниматель"
		profile.Description = "Любит создавать объявления и находить вещам новых хозяев"
		profile.IconKey = "character.entrepreneur"
		profile.VisualDetail = "notebook"
	case model.PetCharacterMechanic:
		profile.Name = "Механик"
		profile.Description = "Интересуется автомобилями и всем, что движется"
		profile.IconKey = "character.mechanic"
		profile.VisualDetail = "toy-wrench"
	case model.PetCharacterTraveler:
		profile.Name = "Путешественник"
		profile.Description = "Собирает идеи для новых поездок"
		profile.IconKey = "character.traveler"
		profile.VisualDetail = "suitcase-badge"
	case model.PetCharacterArchitect:
		profile.Name = "Архитектор"
		profile.Description = "Продумывает пространство и будущий дом"
		profile.IconKey = "character.architect"
		profile.VisualDetail = "blueprint"
	case model.PetCharacterCraftsperson:
		profile.Name = "Мастер"
		profile.Description = "Ценит полезные услуги и умелые руки"
		profile.IconKey = "character.craftsperson"
		profile.VisualDetail = "tool-badge"
	default:
		profile.Name = "Исследователь"
		profile.Description = "Любит искать, сравнивать и сохранять интересные находки"
		profile.IconKey = "character.explorer"
		profile.VisualDetail = "magnifier"
	}
	return profile
}

func leadingCharacter(scores model.ActivityScores) (model.PetCharacter, int) {
	candidates := []struct {
		character model.PetCharacter
		score     int
	}{
		{model.PetCharacterExplorer, scores.BuyerScore},
		{model.PetCharacterEntrepreneur, scores.SellerScore},
		{model.PetCharacterMechanic, scores.AutoScore},
		{model.PetCharacterTraveler, scores.TravelScore},
		{model.PetCharacterArchitect, scores.RealEstateScore},
		{model.PetCharacterCraftsperson, scores.ServicesScore},
	}
	best := candidates[0]
	for _, item := range candidates[1:] {
		if item.score > best.score {
			best = item
		}
	}
	return best.character, best.score
}

func AchievementCodesForTask(taskCode string, unlockedItem, storyCompleted bool) []string {
	codes := []string{"FIRST_STEP"}
	if unlockedItem {
		codes = append(codes, "HOUSEWARMING")
	}
	switch taskCode {
	case "VIEW_FURNITURE_ADS":
		codes = append(codes, "EXPLORER")
	case "MESSAGE_SELLER":
		codes = append(codes, "IN_TOUCH")
	case "CREATE_FIRST_AD":
		codes = append(codes, "FIRST_AD")
	}
	if storyCompleted {
		codes = append(codes, "ROOM_COMPLETE")
	}
	return codes
}

func ValidActionType(actionType model.ActionType) bool {
	switch actionType {
	case model.ActionTypeAdViewed, model.ActionTypeAdFavorited, model.ActionTypeMessageSent,
		model.ActionTypeAdCreated, model.ActionTypeDeliveryUsed, model.ActionTypeReviewLeft,
		model.ActionTypeBookingCreated:
		return true
	default:
		return false
	}
}

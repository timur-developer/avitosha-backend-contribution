package usecase

import "errors"

var (
	ErrPetNotFound              = errors.New("pet not found")
	ErrInvalidGrowthXP          = errors.New("invalid growth XP")
	ErrTaskNotFound             = errors.New("task not found")
	ErrStoryNotFound            = errors.New("story not found")
	ErrActionNotFound           = errors.New("action not found")
	ErrDailyProgressNotFound    = errors.New("daily progress not found")
	ErrDailyQuestNotFound       = errors.New("daily quest not found")
	ErrLeaderboardEntryNotFound = errors.New("leaderboard entry not found")
	ErrEventIDConflict          = errors.New("event ID belongs to another action")
	ErrInvalidAction            = errors.New("invalid action")
	ErrInvalidPetName           = errors.New("invalid pet name")
	ErrForbiddenPetName         = errors.New("forbidden pet name")
	ErrOutOfOrderStoryStage     = errors.New("story stage is out of order")
)

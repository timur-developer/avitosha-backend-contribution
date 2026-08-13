package usecase

import "errors"

var (
	ErrPetNotFound               = errors.New("pet not found")
	ErrInvalidGrowthXP           = errors.New("invalid growth XP")
	ErrTaskNotFound              = errors.New("task not found")
	ErrStoryNotFound             = errors.New("story not found")
	ErrActionNotFound            = errors.New("action not found")
	ErrDailyProgressNotFound     = errors.New("daily progress not found")
	ErrDailyQuestNotFound        = errors.New("daily quest not found")
	ErrLeaderboardEntryNotFound  = errors.New("leaderboard entry not found")
	ErrEventIDConflict           = errors.New("event ID belongs to another action")
	ErrInvalidAction             = errors.New("invalid action")
	ErrInvalidPetName            = errors.New("invalid pet name")
	ErrForbiddenPetName          = errors.New("forbidden pet name")
	ErrOutOfOrderStoryStage      = errors.New("story stage is out of order")
	ErrListingNotFound           = errors.New("listing not found")
	ErrListingCategoryNotFound   = errors.New("listing category not found")
	ErrListingForbidden          = errors.New("listing access forbidden")
	ErrListingOwnAction          = errors.New("cannot interact with own listing")
	ErrListingInvalidTransition  = errors.New("invalid listing status transition")
	ErrDemoPurchaseCompleted     = errors.New("demo purchase already completed")
	ErrListingNotEligible        = errors.New("listing is not eligible for publication")
	ErrInvalidListingInput       = errors.New("invalid listing input")
	ErrProductActionRuleNotFound = errors.New("product action rule not found")
)

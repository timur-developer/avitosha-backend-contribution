package usecase

const (
	LevelTwoXPThreshold   = 100
	LevelThreeXPThreshold = 250
	LevelFourXPThreshold  = 450
	LevelFiveXPThreshold  = 700
	MaxPetLevel           = 5
)

func CalculateLevel(growthXP int) (int, error) {
	if growthXP < 0 {
		return 0, ErrInvalidGrowthXP
	}
	switch {
	case growthXP >= LevelFiveXPThreshold:
		return MaxPetLevel, nil
	case growthXP >= LevelFourXPThreshold:
		return 4, nil
	case growthXP >= LevelThreeXPThreshold:
		return 3, nil
	case growthXP >= LevelTwoXPThreshold:
		return 2, nil
	default:
		return 1, nil
	}
}

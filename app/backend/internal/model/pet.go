package model

import (
	"time"

	"github.com/google/uuid"
)

type PetMood string

const (
	PetMoodCalm     PetMood = "CALM"
	PetMoodCurious  PetMood = "CURIOUS"
	PetMoodHappy    PetMood = "HAPPY"
	PetMoodExcited  PetMood = "EXCITED"
	PetMoodProud    PetMood = "PROUD"
	PetMoodSleeping PetMood = "SLEEPING"
)

type PetCharacter string

const (
	PetCharacterExplorer     PetCharacter = "EXPLORER"
	PetCharacterEntrepreneur PetCharacter = "ENTREPRENEUR"
	PetCharacterMechanic     PetCharacter = "MECHANIC"
	PetCharacterTraveler     PetCharacter = "TRAVELER"
	PetCharacterArchitect    PetCharacter = "ARCHITECT"
	PetCharacterCraftsperson PetCharacter = "CRAFTSPERSON"
)

type Pet struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Name      string
	Level     int
	GrowthXP  int
	Mood      PetMood
	Character *PetCharacter
	CreatedAt time.Time
	UpdatedAt time.Time
}

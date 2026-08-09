package usecase

import (
	"errors"
	"testing"
)

func TestValidatePetName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		want      string
		wantError error
	}{
		{name: "simple Russian name", input: "мурзик", want: "Мурзик"},
		{name: "normalizes spaces and case", input: "  БЕЛЫЙ   БИМ  ", want: "Белый Бим"},
		{name: "allows a hyphen", input: "иван - царевич", want: "Иван-Царевич"},
		{name: "allows yo", input: "ёжик", want: "Ёжик"},
		{name: "rejects Latin letters", input: "Barsik", wantError: ErrInvalidPetName},
		{name: "rejects mixed alphabets", input: "Барсik", wantError: ErrInvalidPetName},
		{name: "rejects digits", input: "Барсик2", wantError: ErrInvalidPetName},
		{name: "rejects punctuation", input: "Барсик!", wantError: ErrInvalidPetName},
		{name: "rejects a one letter name", input: "Я", wantError: ErrInvalidPetName},
		{name: "rejects an overlong name", input: "Оченьоченьдлинноеимяпитомца", wantError: ErrInvalidPetName},
		{name: "rejects profanity", input: "Мудак", wantError: ErrForbiddenPetName},
		{name: "rejects insulting names", input: "Какашка", wantError: ErrForbiddenPetName},
		{name: "rejects separated profanity", input: "Х у й", wantError: ErrForbiddenPetName},
		{name: "rejects forbidden historical name", input: "Гитлер", wantError: ErrForbiddenPetName},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := ValidatePetName(test.input)
			if !errors.Is(err, test.wantError) {
				t.Fatalf("ValidatePetName(%q) error = %v, want %v", test.input, err, test.wantError)
			}
			if got != test.want {
				t.Fatalf("ValidatePetName(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

package usecase

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	MinPetNameLength = 2
	MaxPetNameLength = 20
)

var (
	petNamePattern            = regexp.MustCompile(`^[А-Яа-яЁё]+(?:[ -][А-Яа-яЁё]+)*$`)
	forbiddenPetNameFragments = []string{
		"бля", "гандон", "говн", "дебил", "дерьм", "долбо", "еба", "ебл", "ебу", "идиот",
		"какаш", "мраз", "мудак", "нацист", "педик", "пидор", "пизд", "сволоч", "скотин",
		"сука", "суч", "твар", "ублюд", "урод", "фашист", "хуе", "хуи", "хуй", "хуя", "шлюх",
	}
	forbiddenPetNames = map[string]struct{}{
		"геббельс": {},
		"гитлер":   {},
		"чекатило": {},
	}
)

// ValidatePetName returns a canonical display name or a validation error.
func ValidatePetName(value string) (string, error) {
	name := normalizePetName(value)
	length := utf8.RuneCountInString(name)
	if length < MinPetNameLength || length > MaxPetNameLength || !petNamePattern.MatchString(name) {
		return "", ErrInvalidPetName
	}
	if isForbiddenPetName(name) {
		return "", ErrForbiddenPetName
	}
	return name, nil
}

func normalizePetName(value string) string {
	value = strings.TrimSpace(value)
	for strings.Contains(value, "  ") {
		value = strings.ReplaceAll(value, "  ", " ")
	}
	value = strings.ReplaceAll(value, " -", "-")
	value = strings.ReplaceAll(value, "- ", "-")

	runes := []rune(strings.ToLower(value))
	capitalizeNext := true
	for index, current := range runes {
		if capitalizeNext && unicode.IsLetter(current) {
			runes[index] = unicode.ToUpper(current)
			capitalizeNext = false
			continue
		}
		capitalizeNext = current == ' ' || current == '-'
	}
	return string(runes)
}

func isForbiddenPetName(name string) bool {
	compact := strings.NewReplacer(" ", "", "-", "", "ё", "е").Replace(strings.ToLower(name))
	if _, forbidden := forbiddenPetNames[compact]; forbidden {
		return true
	}
	for _, fragment := range forbiddenPetNameFragments {
		if strings.Contains(compact, fragment) {
			return true
		}
	}
	return false
}

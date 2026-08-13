package usecase

import (
	"context"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
)

const maxAdviceLength = 220

func (service *GameService) GetTaskAdvice(
	ctx context.Context,
	userID uuid.UUID,
	taskID uuid.UUID,
	now time.Time,
) (TaskAdvice, error) {
	profile, err := service.EnsureProfile(ctx, userID, now)
	if err != nil {
		return TaskAdvice{}, err
	}
	task, err := service.repository.GetTaskProgress(ctx, userID, taskID)
	if err != nil {
		return TaskAdvice{}, err
	}

	fallback := TaskAdvice{TaskID: taskID, Text: fallbackTaskAdvice(task)}
	if service.advice == nil {
		return fallback, nil
	}

	input := AdviceGenerationInput{
		PetName: profile.Pet.Name, PetMood: profile.Pet.Mood,
		CharacterName: profile.Character.Name, TaskTitle: task.Task.Title,
		TaskDescription: task.Task.Description, ActionType: task.Task.ActionType,
		Progress: task.Progress.Progress, Target: task.Progress.TargetValue,
		Status: task.Progress.Status, XPReward: task.Task.XPReward,
		AvitoRewardAmount: task.Task.AvitoRewardAmount,
	}
	if task.Task.RoomItemCode != nil {
		input.RoomItemCode = *task.Task.RoomItemCode
	}
	if task.Task.AvitoRewardType != nil {
		input.AvitoRewardType = *task.Task.AvitoRewardType
	}

	text, err := service.advice.Generate(ctx, input)
	text = normalizeAdvice(text)
	if err != nil || !validAdvice(text) {
		return fallback, nil
	}

	return TaskAdvice{TaskID: taskID, Text: text, GeneratedByAI: true}, nil
}

func fallbackTaskAdvice(task model.TaskProgress) string {
	if task.Progress.Status != model.TaskStatusActive {
		return "Задание уже выполнено — загляни в список и выбери следующую цель."
	}

	remaining := max(task.Progress.TargetValue-task.Progress.Progress, 0)
	switch task.Task.Code {
	case "VIEW_FURNITURE_ADS":
		return "Посмотри ещё " + russianCount(remaining, "объявление", "объявления", "объявлений") + " с мебелью — сравни фото, состояние и условия доставки."
	case "FAVORITE_FURNITURE_AD":
		return "Добавь подходящее объявление в избранное, чтобы быстро вернуться к нему и сравнить варианты."
	case "MESSAGE_SELLER":
		return "Уточни у продавца состояние товара, размеры и удобное время встречи."
	case "CREATE_FIRST_AD":
		return "Добавь понятный заголовок, честное описание и несколько хорошо освещённых фотографий."
	case "USE_DELIVERY":
		return "Проверь адрес, сроки и условия доставки перед подтверждением заказа."
	default:
		return "Сделай следующий шаг по заданию — до цели осталось совсем немного."
	}
}

func russianCount(value int, one, few, many string) string {
	word := many
	if value%10 == 1 && value%100 != 11 {
		word = one
	} else if value%10 >= 2 && value%10 <= 4 && (value%100 < 12 || value%100 > 14) {
		word = few
	}
	return strconv.Itoa(value) + " " + word
}

func normalizeAdvice(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func validAdvice(value string) bool {
	if value == "" || utf8.RuneCountInString(value) > maxAdviceLength {
		return false
	}
	hasCyrillic := false
	for _, symbol := range value {
		if unicode.IsControl(symbol) || symbol == '<' || symbol == '>' {
			return false
		}
		if unicode.In(symbol, unicode.Cyrillic) {
			hasCyrillic = true
		}
	}
	return hasCyrillic
}

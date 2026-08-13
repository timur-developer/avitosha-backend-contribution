package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

type GameUseCase interface {
	EnsureProfile(context.Context, uuid.UUID, time.Time) (usecase.GameProfile, error)
	RenamePet(context.Context, uuid.UUID, string, time.Time) (usecase.GameProfile, error)
	ListTasks(context.Context, uuid.UUID, time.Time) ([]model.TaskProgress, error)
	GetTask(context.Context, uuid.UUID, uuid.UUID, time.Time) (model.TaskProgress, error)
	GetTaskAdvice(context.Context, uuid.UUID, uuid.UUID, time.Time) (usecase.TaskAdvice, error)
	GetRoom(context.Context, uuid.UUID, time.Time) ([]model.RoomItemProgress, error)
	GetStory(context.Context, uuid.UUID, time.Time) (model.StorySnapshot, error)
	GetDailySummary(context.Context, uuid.UUID, time.Time) (usecase.DailySummary, error)
	GetLeaderboard(context.Context, uuid.UUID, int, time.Time) (usecase.Leaderboard, error)
	GetAchievements(context.Context, uuid.UUID, time.Time) ([]model.AchievementProgress, error)
	GetRewardBalances(context.Context, uuid.UUID, time.Time) ([]model.RewardBalance, error)
	GetRewardWallet(context.Context, uuid.UUID, time.Time) (usecase.RewardWallet, error)
	ProcessAction(context.Context, usecase.ProcessActionCommand) (usecase.ProcessActionResult, error)
}

type GameHandler struct {
	logger  *slog.Logger
	service GameUseCase
	now     func() time.Time
}

func NewGameHandler(logger *slog.Logger, service GameUseCase, now func() time.Time) GameHandler {
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = time.Now
	}
	return GameHandler{logger: logger, service: service, now: now}
}

func (handler GameHandler) GetPet(w http.ResponseWriter, r *http.Request) {
	userID, ok := handler.requireUser(w, r)
	if !ok {
		return
	}
	profile, err := handler.service.EnsureProfile(r.Context(), userID, handler.now())
	if err != nil {
		handler.writeError(w, r, "get_pet", err)
		return
	}
	writeJSON(w, http.StatusOK, newGamePetDTO(profile))
}

func (handler GameHandler) RenamePet(w http.ResponseWriter, r *http.Request) {
	userID, ok := handler.requireUser(w, r)
	if !ok {
		return
	}
	request, err := decodeRenamePetRequest(r)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, invalidRequestCode, err.Error())
		return
	}
	profile, err := handler.service.RenamePet(r.Context(), userID, request.Name, handler.now())
	if err != nil {
		handler.writeError(w, r, "rename_pet", err)
		return
	}
	writeJSON(w, http.StatusOK, newGamePetDTO(profile))
}

func (handler GameHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	userID, ok := handler.requireUser(w, r)
	if !ok {
		return
	}
	tasks, err := handler.service.ListTasks(r.Context(), userID, handler.now())
	if err != nil {
		handler.writeError(w, r, "list_tasks", err)
		return
	}
	writeJSON(w, http.StatusOK, taskListDTO{Tasks: newTaskDTOs(tasks)})
}

func (handler GameHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := handler.requireUser(w, r)
	if !ok {
		return
	}
	taskID, err := uuid.Parse(chi.URLParam(r, "task_id"))
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, invalidRequestCode, "task_id must be a UUID")
		return
	}
	task, err := handler.service.GetTask(r.Context(), userID, taskID, handler.now())
	if err != nil {
		handler.writeError(w, r, "get_task", err)
		return
	}
	writeJSON(w, http.StatusOK, newTaskDTO(task))
}

func (handler GameHandler) GetTaskAdvice(w http.ResponseWriter, r *http.Request) {
	userID, ok := handler.requireUser(w, r)
	if !ok {
		return
	}
	taskID, err := uuid.Parse(chi.URLParam(r, "task_id"))
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, invalidRequestCode, "task_id must be a UUID")
		return
	}
	advice, err := handler.service.GetTaskAdvice(r.Context(), userID, taskID, handler.now())
	if err != nil {
		handler.writeError(w, r, "get_task_advice", err)
		return
	}
	writeJSON(w, http.StatusOK, newTaskAdviceDTO(advice))
}

func (handler GameHandler) ProcessAction(w http.ResponseWriter, r *http.Request) {
	userID, ok := handler.requireUser(w, r)
	if !ok {
		return
	}
	request, err := decodeActionRequest(r)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, invalidRequestCode, err.Error())
		return
	}
	result, err := handler.service.ProcessAction(r.Context(), usecase.ProcessActionCommand{
		UserID: userID, EventID: request.EventID, ActionType: request.Type,
		EntityID: request.EntityID, Category: request.Category, Metadata: request.Metadata,
		OccurredAt: request.OccurredAt, Now: handler.now(),
	})
	if err != nil {
		handler.writeError(w, r, "process_action", err)
		return
	}
	writeJSON(w, http.StatusOK, newActionResultDTO(result))
}

func (handler GameHandler) GetRoom(w http.ResponseWriter, r *http.Request) {
	userID, ok := handler.requireUser(w, r)
	if !ok {
		return
	}
	items, err := handler.service.GetRoom(r.Context(), userID, handler.now())
	if err != nil {
		handler.writeError(w, r, "get_room", err)
		return
	}
	writeJSON(w, http.StatusOK, newRoomDTO(items))
}

func (handler GameHandler) GetStory(w http.ResponseWriter, r *http.Request) {
	userID, ok := handler.requireUser(w, r)
	if !ok {
		return
	}
	story, err := handler.service.GetStory(r.Context(), userID, handler.now())
	if err != nil {
		handler.writeError(w, r, "get_story", err)
		return
	}
	writeJSON(w, http.StatusOK, newStoryDTO(story))
}

func (handler GameHandler) GetDailySummary(w http.ResponseWriter, r *http.Request) {
	userID, ok := handler.requireUser(w, r)
	if !ok {
		return
	}
	summary, err := handler.service.GetDailySummary(r.Context(), userID, handler.now())
	if err != nil {
		handler.writeError(w, r, "get_daily_summary", err)
		return
	}
	writeJSON(w, http.StatusOK, newDailySummaryDTO(summary))
}

func (handler GameHandler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	userID, ok := handler.requireUser(w, r)
	if !ok {
		return
	}
	if period := strings.TrimSpace(r.URL.Query().Get("period")); period != "" && period != "weekly" {
		writeErrorResponse(w, http.StatusBadRequest, invalidRequestCode, "period must be weekly")
		return
	}
	limit, err := parseLeaderboardLimit(r)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, invalidRequestCode, err.Error())
		return
	}
	leaderboard, err := handler.service.GetLeaderboard(r.Context(), userID, limit, handler.now())
	if err != nil {
		handler.writeError(w, r, "get_leaderboard", err)
		return
	}
	writeJSON(w, http.StatusOK, newLeaderboardDTO(leaderboard))
}

func (handler GameHandler) GetAchievements(w http.ResponseWriter, r *http.Request) {
	userID, ok := handler.requireUser(w, r)
	if !ok {
		return
	}
	achievements, err := handler.service.GetAchievements(r.Context(), userID, handler.now())
	if err != nil {
		handler.writeError(w, r, "get_achievements", err)
		return
	}
	writeJSON(w, http.StatusOK, newAchievementsDTO(achievements))
}

func (handler GameHandler) GetRewardBalances(w http.ResponseWriter, r *http.Request) {
	userID, ok := handler.requireUser(w, r)
	if !ok {
		return
	}
	balances, err := handler.service.GetRewardBalances(r.Context(), userID, handler.now())
	if err != nil {
		handler.writeError(w, r, "get_reward_balances", err)
		return
	}
	writeJSON(w, http.StatusOK, rewardBalanceListDTO{Balances: newRewardBalanceDTOs(balances)})
}

func (handler GameHandler) GetRewardWallet(w http.ResponseWriter, r *http.Request) {
	userID, ok := handler.requireUser(w, r)
	if !ok {
		return
	}
	wallet, err := handler.service.GetRewardWallet(r.Context(), userID, handler.now())
	if err != nil {
		handler.writeError(w, r, "get_reward_wallet", err)
		return
	}
	writeJSON(w, http.StatusOK, newRewardWalletDTO(wallet))
}

func (handler GameHandler) requireUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := gameUserID(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, unauthorizedCode, "Authentication is required")
		return uuid.Nil, false
	}
	return userID, true
}

func (handler GameHandler) writeError(w http.ResponseWriter, r *http.Request, operation string, err error) {
	status, code, message := mapGameUsecaseError(err)
	if status >= http.StatusInternalServerError {
		handler.logger.Error("game request failed", "request_id", chimiddleware.GetReqID(r.Context()),
			"operation", operation, "error", err.Error())
	}
	writeErrorResponse(w, status, code, message)
}

func mapGameUsecaseError(err error) (int, string, string) {
	switch {
	case errors.Is(err, usecase.ErrInvalidPetName):
		return http.StatusBadRequest, "invalid_pet_name", "Имя должно содержать от 2 до 20 символов: только русские буквы, пробел или дефис"
	case errors.Is(err, usecase.ErrForbiddenPetName):
		return http.StatusBadRequest, "forbidden_pet_name", "Это имя нельзя использовать. Выберите доброе имя без оскорблений"
	case errors.Is(err, usecase.ErrInvalidAction):
		return http.StatusBadRequest, invalidRequestCode, "Action is invalid"
	case errors.Is(err, usecase.ErrEventIDConflict):
		return http.StatusConflict, "event_id_conflict", "eventId was already used for another action"
	case errors.Is(err, usecase.ErrTaskNotFound):
		return http.StatusNotFound, "task_not_found", "Task not found"
	case errors.Is(err, usecase.ErrStoryNotFound):
		return http.StatusNotFound, "story_not_found", "Story not found"
	default:
		return http.StatusInternalServerError, internalErrorCode, "Internal server error"
	}
}

type actionRequest struct {
	EventID    uuid.UUID        `json:"eventId"`
	Type       model.ActionType `json:"type"`
	EntityID   *string          `json:"entityId"`
	Category   *string          `json:"category"`
	OccurredAt time.Time        `json:"occurredAt"`
	Metadata   json.RawMessage  `json:"metadata"`
}

type renamePetRequest struct {
	Name string `json:"name"`
}

func decodeRenamePetRequest(r *http.Request) (renamePetRequest, error) {
	var request renamePetRequest
	decoder := json.NewDecoder(io.LimitReader(r.Body, 4<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return renamePetRequest{}, fmt.Errorf("request body must be valid JSON")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return renamePetRequest{}, fmt.Errorf("request body must contain one JSON object")
	}
	return request, nil
}

func decodeActionRequest(r *http.Request) (actionRequest, error) {
	var request actionRequest
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return actionRequest{}, fmt.Errorf("request body must be valid JSON")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return actionRequest{}, fmt.Errorf("request body must contain one JSON object")
	}
	if request.EventID == uuid.Nil {
		return actionRequest{}, fmt.Errorf("eventId must be a UUID")
	}
	if strings.TrimSpace(string(request.Type)) == "" {
		return actionRequest{}, fmt.Errorf("type is required")
	}
	if request.OccurredAt.IsZero() {
		return actionRequest{}, fmt.Errorf("occurredAt must be RFC3339 date-time")
	}
	if len(request.Metadata) == 0 {
		request.Metadata = json.RawMessage(`{}`)
	}
	return request, nil
}

func parseLeaderboardLimit(r *http.Request) (int, error) {
	value := strings.TrimSpace(r.URL.Query().Get("limit"))
	if value == "" {
		return usecase.DefaultLeaderboardLimit, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit < 1 || limit > 100 {
		return 0, fmt.Errorf("limit must be between 1 and 100")
	}
	return limit, nil
}

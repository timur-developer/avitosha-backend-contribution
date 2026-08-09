package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

func (repository *GameRepository) InsertAction(
	ctx context.Context,
	candidate model.UserAction,
) (model.UserAction, bool, error) {
	executor := executorFromContext(ctx, repository.executor)
	row := executor.QueryRow(ctx, `
INSERT INTO user_actions (
    id, user_id, event_id, action_type, entity_id, category, metadata,
    occurred_at, processed_at, result_events, created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULL, '[]'::JSONB, $9)
ON CONFLICT (event_id) DO NOTHING
RETURNING id, user_id, event_id, action_type, entity_id, category, metadata,
          occurred_at, processed_at, result_events, created_at
`, candidate.ID, candidate.UserID, candidate.EventID, candidate.ActionType, candidate.EntityID,
		candidate.Category, candidate.Metadata, candidate.OccurredAt, candidate.CreatedAt)

	action, err := scanAction(row)
	if err == nil {
		return action, true, nil
	}
	if !errors.Is(err, usecase.ErrActionNotFound) {
		return model.UserAction{}, false, fmt.Errorf("insert action: %w", err)
	}

	existing, err := scanAction(executor.QueryRow(ctx, `
SELECT id, user_id, event_id, action_type, entity_id, category, metadata,
       occurred_at, processed_at, result_events, created_at
FROM user_actions
WHERE event_id = $1
`, candidate.EventID))
	if err != nil {
		return model.UserAction{}, false, fmt.Errorf("get idempotent action: %w", err)
	}
	return existing, false, nil
}

func (repository *GameRepository) CompleteAction(
	ctx context.Context,
	actionID uuid.UUID,
	processedAt time.Time,
	events []model.DomainEvent,
) error {
	result, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("marshal action result events: %w", err)
	}

	tag, err := executorFromContext(ctx, repository.executor).Exec(ctx, `
UPDATE user_actions
SET processed_at = $2, result_events = $3
WHERE id = $1 AND processed_at IS NULL
`, actionID, processedAt, result)
	if err != nil {
		return mapGameStorageError("complete action", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("complete action: %w", usecase.ErrActionNotFound)
	}
	return nil
}

func (repository *GameRepository) InsertDomainEvents(ctx context.Context, events []model.DomainEvent) error {
	executor := executorFromContext(ctx, repository.executor)
	for _, event := range events {
		_, err := executor.Exec(ctx, `
INSERT INTO domain_events (id, user_id, action_id, event_type, payload, occurred_at)
VALUES ($1, $2, $3, $4, $5, $6)
`, event.ID, event.UserID, event.ActionID, event.Type, event.Payload, event.OccurredAt)
		if err != nil {
			return mapGameStorageError("insert domain event", err)
		}
	}
	return nil
}

func scanAction(row pgx.Row) (model.UserAction, error) {
	var action model.UserAction
	if err := row.Scan(
		&action.ID, &action.UserID, &action.EventID, &action.ActionType, &action.EntityID,
		&action.Category, &action.Metadata, &action.OccurredAt, &action.ProcessedAt,
		&action.ResultEvents, &action.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.UserAction{}, usecase.ErrActionNotFound
		}
		return model.UserAction{}, mapGameStorageError("scan action", err)
	}
	return action, nil
}

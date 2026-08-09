package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

func (repository *GameRepository) AssignStoryTask(
	ctx context.Context,
	userID uuid.UUID,
	storyCode string,
	stage int,
	now time.Time,
) (model.TaskProgress, error) {
	executor := executorFromContext(ctx, repository.executor)
	_, err := executor.Exec(ctx, `
INSERT INTO user_tasks (id, user_id, task_id, progress, target_value, status, assigned_at, created_at, updated_at)
SELECT gen_random_uuid(), $1, id, 0, target_value, 'ACTIVE', $4, $4, $4
FROM tasks
WHERE story_code = $2 AND story_stage = $3 AND is_active
ON CONFLICT (user_id, task_id) DO NOTHING
`, userID, storyCode, stage, now)
	if err != nil {
		return model.TaskProgress{}, mapGameStorageError("assign story task", err)
	}

	return scanTaskProgress(executor.QueryRow(ctx, taskProgressSelect+`
WHERE ut.user_id = $1 AND t.story_code = $2 AND t.story_stage = $3
`, userID, storyCode, stage))
}

func (repository *GameRepository) ListTaskProgress(ctx context.Context, userID uuid.UUID) ([]model.TaskProgress, error) {
	rows, err := executorFromContext(ctx, repository.executor).Query(ctx, taskProgressSelect+`
WHERE ut.user_id = $1
ORDER BY CASE ut.status WHEN 'ACTIVE' THEN 0 WHEN 'COMPLETED' THEN 1 WHEN 'REWARDED' THEN 2 ELSE 3 END,
         t.story_stage NULLS LAST, ut.assigned_at, t.code
`, userID)
	if err != nil {
		return nil, mapGameStorageError("list task progress", err)
	}
	defer rows.Close()

	return collectTaskProgress(rows)
}

func (repository *GameRepository) GetTaskProgress(
	ctx context.Context,
	userID uuid.UUID,
	taskID uuid.UUID,
) (model.TaskProgress, error) {
	return scanTaskProgress(executorFromContext(ctx, repository.executor).QueryRow(ctx, taskProgressSelect+`
WHERE ut.user_id = $1 AND t.id = $2
`, userID, taskID))
}

func (repository *GameRepository) FindMatchingActiveTasksForUpdate(
	ctx context.Context,
	userID uuid.UUID,
	actionType model.ActionType,
	category *string,
) ([]model.TaskProgress, error) {
	rows, err := executorFromContext(ctx, repository.executor).Query(ctx, taskProgressSelect+`
WHERE ut.user_id = $1
  AND ut.status = 'ACTIVE'
  AND t.is_active
  AND t.action_type = $2
  AND (t.category IS NULL OR t.category = $3)
ORDER BY t.story_stage NULLS LAST, t.code
FOR UPDATE OF ut
`, userID, actionType, category)
	if err != nil {
		return nil, mapGameStorageError("find matching tasks", err)
	}
	defer rows.Close()

	return collectTaskProgress(rows)
}

func (repository *GameRepository) UpdateTaskProgress(ctx context.Context, progress model.UserTask) error {
	tag, err := executorFromContext(ctx, repository.executor).Exec(ctx, `
UPDATE user_tasks
SET progress = $2, status = $3, completed_at = $4, rewarded_at = $5, updated_at = $6
WHERE id = $1
`, progress.ID, progress.Progress, progress.Status, progress.CompletedAt, progress.RewardedAt, progress.UpdatedAt)
	if err != nil {
		return mapGameStorageError("update task progress", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("update task progress: %w", usecase.ErrTaskNotFound)
	}
	return nil
}

const taskProgressSelect = `
SELECT t.id, t.code, t.title, t.description, t.pet_phrase, t.action_type, t.category,
       t.target_value, t.xp_reward, t.avito_reward_type, t.avito_reward_amount,
       t.room_item_code, t.story_code, t.story_stage, t.is_active, t.created_at, t.updated_at,
       ut.id, ut.user_id, ut.task_id, ut.progress, ut.target_value, ut.status,
       ut.assigned_at, ut.completed_at, ut.rewarded_at, ut.created_at, ut.updated_at
FROM user_tasks ut
JOIN tasks t ON t.id = ut.task_id
`

func scanTaskProgress(row pgx.Row) (model.TaskProgress, error) {
	var progress model.TaskProgress
	if err := row.Scan(taskProgressDestinations(&progress)...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.TaskProgress{}, usecase.ErrTaskNotFound
		}
		return model.TaskProgress{}, mapGameStorageError("scan task progress", err)
	}
	return progress, nil
}

func collectTaskProgress(rows pgx.Rows) ([]model.TaskProgress, error) {
	items := make([]model.TaskProgress, 0)
	for rows.Next() {
		var item model.TaskProgress
		if err := rows.Scan(taskProgressDestinations(&item)...); err != nil {
			return nil, mapGameStorageError("scan task progress list", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapGameStorageError("iterate task progress", err)
	}
	return items, nil
}

func taskProgressDestinations(progress *model.TaskProgress) []any {
	return append(taskDestinations(&progress.Task),
		&progress.Progress.ID, &progress.Progress.UserID, &progress.Progress.TaskID,
		&progress.Progress.Progress, &progress.Progress.TargetValue, &progress.Progress.Status,
		&progress.Progress.AssignedAt, &progress.Progress.CompletedAt, &progress.Progress.RewardedAt,
		&progress.Progress.CreatedAt, &progress.Progress.UpdatedAt,
	)
}

func scanTask(row pgx.Row) (model.Task, error) {
	var task model.Task
	if err := row.Scan(taskDestinations(&task)...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Task{}, usecase.ErrTaskNotFound
		}
		return model.Task{}, mapGameStorageError("scan task", err)
	}
	return task, nil
}

func taskDestinations(task *model.Task) []any {
	return []any{
		&task.ID, &task.Code, &task.Title, &task.Description, &task.PetPhrase, &task.ActionType,
		&task.Category, &task.TargetValue, &task.XPReward, &task.AvitoRewardType,
		&task.AvitoRewardAmount, &task.RoomItemCode, &task.StoryCode, &task.StoryStage,
		&task.IsActive, &task.CreatedAt, &task.UpdatedAt,
	}
}

func (repository *GameRepository) EnsureInitialRoomItem(ctx context.Context, item model.UserRoomItem) error {
	_, err := executorFromContext(ctx, repository.executor).Exec(ctx, `
INSERT INTO user_room_items (
    id, user_id, item_code, status, source_task_id, unlocked_at, placed_at, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (user_id, item_code) DO NOTHING
`, item.ID, item.UserID, item.ItemCode, item.Status, item.SourceTaskID, item.UnlockedAt,
		item.PlacedAt, item.CreatedAt, item.UpdatedAt)
	if err != nil {
		return mapGameStorageError("ensure initial room item", err)
	}
	return nil
}

func (repository *GameRepository) UnlockRoomItem(ctx context.Context, item model.UserRoomItem) (bool, error) {
	tag, err := executorFromContext(ctx, repository.executor).Exec(ctx, `
INSERT INTO user_room_items (
    id, user_id, item_code, status, source_task_id, unlocked_at, placed_at, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (user_id, item_code) DO NOTHING
`, item.ID, item.UserID, item.ItemCode, item.Status, item.SourceTaskID, item.UnlockedAt,
		item.PlacedAt, item.CreatedAt, item.UpdatedAt)
	if err != nil {
		return false, mapGameStorageError("unlock room item", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (repository *GameRepository) ListRoomItems(ctx context.Context, userID uuid.UUID) ([]model.RoomItemProgress, error) {
	rows, err := executorFromContext(ctx, repository.executor).Query(ctx, `
SELECT ri.code, ri.name, ri.description, ri.asset_key, ri.position_key, ri.unlock_level,
       ri.sort_order, ri.created_at, ri.updated_at,
       COALESCE(uri.status, 'LOCKED'), unlock_task.code, uri.unlocked_at
FROM room_items ri
LEFT JOIN user_room_items uri ON uri.item_code = ri.code AND uri.user_id = $1
LEFT JOIN tasks unlock_task ON unlock_task.room_item_code = ri.code AND unlock_task.is_active
ORDER BY ri.sort_order, ri.code
`, userID)
	if err != nil {
		return nil, mapGameStorageError("list room items", err)
	}
	defer rows.Close()

	items := make([]model.RoomItemProgress, 0)
	for rows.Next() {
		var item model.RoomItemProgress
		if err := rows.Scan(
			&item.Item.Code, &item.Item.Name, &item.Item.Description, &item.Item.AssetKey,
			&item.Item.PositionKey, &item.Item.UnlockLevel, &item.Item.SortOrder,
			&item.Item.CreatedAt, &item.Item.UpdatedAt, &item.Status,
			&item.SourceTaskCode, &item.UnlockedAt,
		); err != nil {
			return nil, mapGameStorageError("scan room item", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapGameStorageError("iterate room items", err)
	}
	return items, nil
}

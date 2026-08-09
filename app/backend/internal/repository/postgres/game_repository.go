package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

type GameRepository struct {
	executor QueryExecutor
}

func NewGameRepository(pool *pgxpool.Pool) *GameRepository {
	return &GameRepository{executor: pool}
}

func (repository *GameRepository) GetOrCreateGamePet(ctx context.Context, candidate model.Pet) (model.Pet, error) {
	executor := executorFromContext(ctx, repository.executor)
	_, err := executor.Exec(ctx, `
INSERT INTO pets (id, user_id, name, level, growth_xp, mood, character, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (user_id) DO NOTHING
`, candidate.ID, candidate.UserID, candidate.Name, candidate.Level, candidate.GrowthXP, candidate.Mood,
		candidate.Character, candidate.CreatedAt, candidate.UpdatedAt)
	if err != nil {
		return model.Pet{}, mapGameStorageError("create game pet", err)
	}

	return repository.GetGamePetByUserIDForUpdate(ctx, candidate.UserID)
}

func (repository *GameRepository) GetGamePetByUserID(ctx context.Context, userID uuid.UUID) (model.Pet, error) {
	return scanGamePet(executorFromContext(ctx, repository.executor).QueryRow(ctx, `
SELECT id, user_id, name, level, growth_xp, mood, character, created_at, updated_at
FROM pets
WHERE user_id = $1
`, userID))
}

func (repository *GameRepository) GetGamePetByUserIDForUpdate(ctx context.Context, userID uuid.UUID) (model.Pet, error) {
	return scanGamePet(executorFromContext(ctx, repository.executor).QueryRow(ctx, `
SELECT id, user_id, name, level, growth_xp, mood, character, created_at, updated_at
FROM pets
WHERE user_id = $1
FOR UPDATE
`, userID))
}

func (repository *GameRepository) UpdateGamePet(ctx context.Context, pet model.Pet) error {
	tag, err := executorFromContext(ctx, repository.executor).Exec(ctx, `
UPDATE pets
SET name = $2, level = $3, growth_xp = $4, mood = $5, character = $6, updated_at = $7
WHERE id = $1
`, pet.ID, pet.Name, pet.Level, pet.GrowthXP, pet.Mood, pet.Character, pet.UpdatedAt)
	if err != nil {
		return mapGameStorageError("update game pet", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("update game pet: %w", usecase.ErrPetNotFound)
	}
	return nil
}

func scanGamePet(row pgx.Row) (model.Pet, error) {
	var pet model.Pet
	if err := row.Scan(
		&pet.ID, &pet.UserID, &pet.Name, &pet.Level, &pet.GrowthXP, &pet.Mood,
		&pet.Character, &pet.CreatedAt, &pet.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Pet{}, usecase.ErrPetNotFound
		}
		return model.Pet{}, mapGameStorageError("scan game pet", err)
	}
	return pet, nil
}

func (repository *GameRepository) GetOrCreateStoryProgress(
	ctx context.Context,
	candidate model.UserStoryProgress,
) (model.UserStoryProgress, error) {
	executor := executorFromContext(ctx, repository.executor)
	_, err := executor.Exec(ctx, `
INSERT INTO user_story_progress (
    id, user_id, story_code, current_stage, status, started_at, completed_at, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (user_id, story_code) DO NOTHING
`, candidate.ID, candidate.UserID, candidate.StoryCode, candidate.CurrentStage, candidate.Status,
		candidate.StartedAt, candidate.CompletedAt, candidate.CreatedAt, candidate.UpdatedAt)
	if err != nil {
		return model.UserStoryProgress{}, mapGameStorageError("create story progress", err)
	}

	return repository.GetStoryProgressForUpdate(ctx, candidate.UserID, candidate.StoryCode)
}

func (repository *GameRepository) GetStoryProgressForUpdate(
	ctx context.Context,
	userID uuid.UUID,
	storyCode string,
) (model.UserStoryProgress, error) {
	return scanStoryProgress(executorFromContext(ctx, repository.executor).QueryRow(ctx, `
SELECT id, user_id, story_code, current_stage, status, started_at, completed_at, created_at, updated_at
FROM user_story_progress
WHERE user_id = $1 AND story_code = $2
FOR UPDATE
`, userID, storyCode))
}

func (repository *GameRepository) UpdateStoryProgress(ctx context.Context, progress model.UserStoryProgress) error {
	tag, err := executorFromContext(ctx, repository.executor).Exec(ctx, `
UPDATE user_story_progress
SET current_stage = $2, status = $3, completed_at = $4, updated_at = $5
WHERE id = $1
`, progress.ID, progress.CurrentStage, progress.Status, progress.CompletedAt, progress.UpdatedAt)
	if err != nil {
		return mapGameStorageError("update story progress", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("update story progress: %w", usecase.ErrStoryNotFound)
	}
	return nil
}

func scanStoryProgress(row pgx.Row) (model.UserStoryProgress, error) {
	var progress model.UserStoryProgress
	if err := row.Scan(
		&progress.ID, &progress.UserID, &progress.StoryCode, &progress.CurrentStage, &progress.Status,
		&progress.StartedAt, &progress.CompletedAt, &progress.CreatedAt, &progress.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.UserStoryProgress{}, usecase.ErrStoryNotFound
		}
		return model.UserStoryProgress{}, mapGameStorageError("scan story progress", err)
	}
	return progress, nil
}

func (repository *GameRepository) GetStorySnapshot(
	ctx context.Context,
	userID uuid.UUID,
	storyCode string,
) (model.StorySnapshot, error) {
	var snapshot model.StorySnapshot
	err := executorFromContext(ctx, repository.executor).QueryRow(ctx, `
SELECT s.code, s.title, s.description, s.total_stages, s.is_active, s.created_at, s.updated_at,
       usp.id, usp.user_id, usp.story_code, usp.current_stage, usp.status,
       usp.started_at, usp.completed_at, usp.created_at, usp.updated_at
FROM stories s
JOIN user_story_progress usp ON usp.story_code = s.code AND usp.user_id = $1
WHERE s.code = $2
`, userID, storyCode).Scan(
		&snapshot.Story.Code, &snapshot.Story.Title, &snapshot.Story.Description, &snapshot.Story.TotalStages,
		&snapshot.Story.IsActive, &snapshot.Story.CreatedAt, &snapshot.Story.UpdatedAt,
		&snapshot.Progress.ID, &snapshot.Progress.UserID, &snapshot.Progress.StoryCode,
		&snapshot.Progress.CurrentStage, &snapshot.Progress.Status, &snapshot.Progress.StartedAt,
		&snapshot.Progress.CompletedAt, &snapshot.Progress.CreatedAt, &snapshot.Progress.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.StorySnapshot{}, usecase.ErrStoryNotFound
		}
		return model.StorySnapshot{}, mapGameStorageError("get story snapshot", err)
	}
	if snapshot.Progress.Status == model.StoryStatusActive {
		nextTask, taskErr := scanTask(executorFromContext(ctx, repository.executor).QueryRow(ctx, `
SELECT id, code, title, description, pet_phrase, action_type, category, target_value, xp_reward,
       avito_reward_type, avito_reward_amount, room_item_code, story_code, story_stage,
       is_active, created_at, updated_at
FROM tasks
WHERE story_code = $1 AND story_stage = $2 AND is_active
`, storyCode, snapshot.Progress.CurrentStage+1))
		if taskErr != nil && !errors.Is(taskErr, usecase.ErrTaskNotFound) {
			return model.StorySnapshot{}, fmt.Errorf("get next story task: %w", taskErr)
		}
		if taskErr == nil {
			snapshot.NextTask = &nextTask
		}
	}
	return snapshot, nil
}

func mapGameStorageError(operation string, err error) error {
	return fmt.Errorf("%s: %w: %v", operation, usecase.ErrUnexpectedStorage, err)
}

var _ usecase.GameRepository = (*GameRepository)(nil)

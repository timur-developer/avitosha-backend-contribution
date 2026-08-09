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

func (repository *GameRepository) GetOrCreateUserStreak(
	ctx context.Context,
	candidate model.UserStreak,
) (model.UserStreak, error) {
	executor := executorFromContext(ctx, repository.executor)
	_, err := executor.Exec(ctx, `
INSERT INTO user_streaks (
    user_id, current_streak, longest_streak, last_active_date, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id) DO NOTHING
`, candidate.UserID, candidate.CurrentStreak, candidate.LongestStreak, candidate.LastActiveDate,
		candidate.CreatedAt, candidate.UpdatedAt)
	if err != nil {
		return model.UserStreak{}, mapGameStorageError("create user streak", err)
	}

	return scanUserStreak(executor.QueryRow(ctx, `
SELECT user_id, current_streak, longest_streak, last_active_date, created_at, updated_at
FROM user_streaks
WHERE user_id = $1
FOR UPDATE
`, candidate.UserID))
}

func (repository *GameRepository) UpdateUserStreak(ctx context.Context, streak model.UserStreak) error {
	tag, err := executorFromContext(ctx, repository.executor).Exec(ctx, `
UPDATE user_streaks
SET current_streak = $2, longest_streak = $3, last_active_date = $4, updated_at = $5
WHERE user_id = $1
`, streak.UserID, streak.CurrentStreak, streak.LongestStreak, streak.LastActiveDate, streak.UpdatedAt)
	if err != nil {
		return mapGameStorageError("update user streak", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("update user streak: %w", usecase.ErrUnexpectedStorage)
	}
	return nil
}

func (repository *GameRepository) ListActiveDailyQuestTemplates(ctx context.Context) ([]model.DailyQuestTemplate, error) {
	rows, err := executorFromContext(ctx, repository.executor).Query(ctx, `
SELECT code, title, description, action_type, category, target_value,
       reward_type, reward_amount, sort_order, is_active, created_at, updated_at
FROM daily_quest_templates
WHERE is_active
ORDER BY sort_order, code
`)
	if err != nil {
		return nil, mapGameStorageError("list daily quest templates", err)
	}
	defer rows.Close()

	items := make([]model.DailyQuestTemplate, 0)
	for rows.Next() {
		var item model.DailyQuestTemplate
		if err := rows.Scan(
			&item.Code, &item.Title, &item.Description, &item.ActionType, &item.Category,
			&item.TargetValue, &item.RewardType, &item.RewardAmount, &item.SortOrder,
			&item.IsActive, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, mapGameStorageError("scan daily quest template", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapGameStorageError("iterate daily quest templates", err)
	}
	return items, nil
}

func (repository *GameRepository) ExpireDailyQuestsBefore(
	ctx context.Context,
	userID uuid.UUID,
	date time.Time,
	now time.Time,
) error {
	_, err := executorFromContext(ctx, repository.executor).Exec(ctx, `
UPDATE user_daily_quests
SET status = 'EXPIRED', updated_at = $3
WHERE user_id = $1
  AND quest_date < $2
  AND status IN ('ACTIVE', 'COMPLETED')
`, userID, date, now)
	if err != nil {
		return mapGameStorageError("expire daily quests", err)
	}
	return nil
}

func (repository *GameRepository) AssignDailyQuest(
	ctx context.Context,
	candidate model.UserDailyQuest,
) (model.UserDailyQuest, error) {
	executor := executorFromContext(ctx, repository.executor)
	_, err := executor.Exec(ctx, `
INSERT INTO user_daily_quests (
    id, user_id, quest_date, template_code, progress, target_value, status,
    reward_type, reward_amount, assigned_at, completed_at, rewarded_at, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
ON CONFLICT (user_id, quest_date) DO NOTHING
`, candidate.ID, candidate.UserID, candidate.QuestDate, candidate.TemplateCode, candidate.Progress,
		candidate.TargetValue, candidate.Status, candidate.RewardType, candidate.RewardAmount,
		candidate.AssignedAt, candidate.CompletedAt, candidate.RewardedAt, candidate.CreatedAt, candidate.UpdatedAt)
	if err != nil {
		return model.UserDailyQuest{}, mapGameStorageError("assign daily quest", err)
	}

	progress, err := repository.GetDailyQuestProgress(ctx, candidate.UserID, candidate.QuestDate)
	if err != nil {
		return model.UserDailyQuest{}, err
	}
	return progress.Quest, nil
}

func (repository *GameRepository) GetDailyQuestProgress(
	ctx context.Context,
	userID uuid.UUID,
	date time.Time,
) (model.DailyQuestProgress, error) {
	return repository.getDailyQuestProgress(ctx, userID, date, false)
}

func (repository *GameRepository) GetDailyQuestProgressForUpdate(
	ctx context.Context,
	userID uuid.UUID,
	date time.Time,
) (model.DailyQuestProgress, error) {
	return repository.getDailyQuestProgress(ctx, userID, date, true)
}

func (repository *GameRepository) UpdateDailyQuest(ctx context.Context, quest model.UserDailyQuest) error {
	tag, err := executorFromContext(ctx, repository.executor).Exec(ctx, `
UPDATE user_daily_quests
SET progress = $2, status = $3, completed_at = $4, rewarded_at = $5, updated_at = $6
WHERE id = $1
`, quest.ID, quest.Progress, quest.Status, quest.CompletedAt, quest.RewardedAt, quest.UpdatedAt)
	if err != nil {
		return mapGameStorageError("update daily quest", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("update daily quest: %w", usecase.ErrDailyQuestNotFound)
	}
	return nil
}

func (repository *GameRepository) getDailyQuestProgress(
	ctx context.Context,
	userID uuid.UUID,
	date time.Time,
	forUpdate bool,
) (model.DailyQuestProgress, error) {
	query := dailyQuestProgressSelect + `
WHERE udq.user_id = $1 AND udq.quest_date = $2
`
	if forUpdate {
		query += "FOR UPDATE"
	}
	return scanDailyQuestProgress(executorFromContext(ctx, repository.executor).QueryRow(ctx, query, userID, date))
}

func scanUserStreak(row pgx.Row) (model.UserStreak, error) {
	var streak model.UserStreak
	if err := row.Scan(
		&streak.UserID, &streak.CurrentStreak, &streak.LongestStreak, &streak.LastActiveDate,
		&streak.CreatedAt, &streak.UpdatedAt,
	); err != nil {
		return model.UserStreak{}, mapGameStorageError("scan user streak", err)
	}
	return streak, nil
}

const dailyQuestProgressSelect = `
SELECT dqt.code, dqt.title, dqt.description, dqt.action_type, dqt.category, dqt.target_value,
       dqt.reward_type, dqt.reward_amount, dqt.sort_order, dqt.is_active, dqt.created_at, dqt.updated_at,
       udq.id, udq.user_id, udq.quest_date, udq.template_code, udq.progress, udq.target_value, udq.status,
       udq.reward_type, udq.reward_amount, udq.assigned_at, udq.completed_at, udq.rewarded_at, udq.created_at, udq.updated_at
FROM user_daily_quests udq
JOIN daily_quest_templates dqt ON dqt.code = udq.template_code
`

func scanDailyQuestProgress(row pgx.Row) (model.DailyQuestProgress, error) {
	var progress model.DailyQuestProgress
	if err := row.Scan(
		&progress.Template.Code, &progress.Template.Title, &progress.Template.Description,
		&progress.Template.ActionType, &progress.Template.Category, &progress.Template.TargetValue,
		&progress.Template.RewardType, &progress.Template.RewardAmount, &progress.Template.SortOrder,
		&progress.Template.IsActive, &progress.Template.CreatedAt, &progress.Template.UpdatedAt,
		&progress.Quest.ID, &progress.Quest.UserID, &progress.Quest.QuestDate, &progress.Quest.TemplateCode,
		&progress.Quest.Progress, &progress.Quest.TargetValue, &progress.Quest.Status,
		&progress.Quest.RewardType, &progress.Quest.RewardAmount, &progress.Quest.AssignedAt,
		&progress.Quest.CompletedAt, &progress.Quest.RewardedAt, &progress.Quest.CreatedAt, &progress.Quest.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.DailyQuestProgress{}, usecase.ErrDailyQuestNotFound
		}
		return model.DailyQuestProgress{}, mapGameStorageError("scan daily quest progress", err)
	}
	return progress, nil
}

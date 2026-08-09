package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/usecase"
)

func (repository *GameRepository) UpdateWeeklyProgress(
	ctx context.Context,
	userID uuid.UUID,
	weekStart time.Time,
	delta usecase.WeeklyProgressDelta,
	now time.Time,
) (model.WeeklyProgress, error) {
	var progress model.WeeklyProgress
	err := executorFromContext(ctx, repository.executor).QueryRow(ctx, `
INSERT INTO weekly_progress (
    id, user_id, week_start, earned_xp, completed_tasks, completed_stages, score, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $4 + $5 * 20 + $6 * 50, $7, $7)
ON CONFLICT (user_id, week_start) DO UPDATE
SET earned_xp = weekly_progress.earned_xp + EXCLUDED.earned_xp,
    completed_tasks = weekly_progress.completed_tasks + EXCLUDED.completed_tasks,
    completed_stages = weekly_progress.completed_stages + EXCLUDED.completed_stages,
    score = weekly_progress.score + EXCLUDED.earned_xp + EXCLUDED.completed_tasks * 20 + EXCLUDED.completed_stages * 50,
    updated_at = EXCLUDED.updated_at
RETURNING id, user_id, week_start, earned_xp, completed_tasks, completed_stages, score, created_at, updated_at
`, uuid.New(), userID, weekStart, delta.EarnedXP, delta.CompletedTasks, delta.CompletedStages, now).Scan(
		&progress.ID, &progress.UserID, &progress.WeekStart, &progress.EarnedXP,
		&progress.CompletedTasks, &progress.CompletedStages, &progress.Score,
		&progress.CreatedAt, &progress.UpdatedAt,
	)
	if err != nil {
		return model.WeeklyProgress{}, mapGameStorageError("update weekly progress", err)
	}
	return progress, nil
}

func (repository *GameRepository) UpdateDailyProgress(
	ctx context.Context,
	userID uuid.UUID,
	date time.Time,
	delta usecase.DailyProgressDelta,
	now time.Time,
) (model.DailyProgress, error) {
	var progress model.DailyProgress
	err := executorFromContext(ctx, repository.executor).QueryRow(ctx, `
INSERT INTO daily_progress (
    id, user_id, date, actions_count, completed_tasks, earned_xp,
    level_before, level_after, unlocked_room_items, story_stage_before,
    story_stage_after, weekly_score_delta, pet_mood, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $14)
ON CONFLICT (user_id, date) DO UPDATE
SET actions_count = daily_progress.actions_count + EXCLUDED.actions_count,
    completed_tasks = daily_progress.completed_tasks + EXCLUDED.completed_tasks,
    earned_xp = daily_progress.earned_xp + EXCLUDED.earned_xp,
    level_after = EXCLUDED.level_after,
    unlocked_room_items = ARRAY(
        SELECT DISTINCT item
        FROM unnest(daily_progress.unlocked_room_items || EXCLUDED.unlocked_room_items) AS item
        ORDER BY item
    ),
    story_stage_after = GREATEST(daily_progress.story_stage_after, EXCLUDED.story_stage_after),
    weekly_score_delta = daily_progress.weekly_score_delta + EXCLUDED.weekly_score_delta,
    pet_mood = EXCLUDED.pet_mood,
    updated_at = EXCLUDED.updated_at
RETURNING date, user_id, actions_count, completed_tasks, earned_xp,
          level_before, level_after, unlocked_room_items, story_stage_before,
          story_stage_after, weekly_score_delta, pet_mood
`, uuid.New(), userID, date, delta.ActionsCount, delta.CompletedTasks, delta.EarnedXP,
		delta.LevelBefore, delta.LevelAfter, delta.UnlockedRoomItems, delta.StoryStageBefore,
		delta.StoryStageAfter, delta.WeeklyScoreDelta, delta.PetMood, now).Scan(
		&progress.Date, &progress.UserID, &progress.ActionsCount, &progress.CompletedTasks,
		&progress.EarnedXP, &progress.LevelBefore, &progress.LevelAfter,
		&progress.UnlockedRoomItems, &progress.StoryStageBefore, &progress.StoryStageAfter,
		&progress.WeeklyScoreDelta, &progress.PetMood,
	)
	if err != nil {
		return model.DailyProgress{}, mapGameStorageError("update daily progress", err)
	}
	return progress, nil
}

func (repository *GameRepository) GetDailyProgress(
	ctx context.Context,
	userID uuid.UUID,
	date time.Time,
) (model.DailyProgress, error) {
	var progress model.DailyProgress
	err := executorFromContext(ctx, repository.executor).QueryRow(ctx, `
SELECT date, user_id, actions_count, completed_tasks, earned_xp,
       level_before, level_after, unlocked_room_items, story_stage_before,
       story_stage_after, weekly_score_delta, pet_mood
FROM daily_progress
WHERE user_id = $1 AND date = $2
`, userID, date).Scan(
		&progress.Date, &progress.UserID, &progress.ActionsCount, &progress.CompletedTasks,
		&progress.EarnedXP, &progress.LevelBefore, &progress.LevelAfter,
		&progress.UnlockedRoomItems, &progress.StoryStageBefore, &progress.StoryStageAfter,
		&progress.WeeklyScoreDelta, &progress.PetMood,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.DailyProgress{}, usecase.ErrDailyProgressNotFound
		}
		return model.DailyProgress{}, mapGameStorageError("get daily progress", err)
	}
	return progress, nil
}

func (repository *GameRepository) ListWeeklyLeaders(
	ctx context.Context,
	weekStart time.Time,
	limit int,
) ([]model.LeaderboardEntry, error) {
	rows, err := executorFromContext(ctx, repository.executor).Query(ctx, `
SELECT position, user_id, pet_name, level, score, completed_tasks
FROM (
    SELECT ROW_NUMBER() OVER (ORDER BY wp.score DESC, wp.user_id) AS position,
           wp.user_id, p.name AS pet_name, p.level, wp.score, wp.completed_tasks
    FROM weekly_progress wp
    JOIN pets p ON p.user_id = wp.user_id
    WHERE wp.week_start = $1
) ranked
ORDER BY position
LIMIT $2
`, weekStart, limit)
	if err != nil {
		return nil, mapGameStorageError("list weekly leaders", err)
	}
	defer rows.Close()

	leaders := make([]model.LeaderboardEntry, 0, limit)
	for rows.Next() {
		var entry model.LeaderboardEntry
		if err := rows.Scan(
			&entry.Position, &entry.UserID, &entry.PetName, &entry.Level,
			&entry.Score, &entry.CompletedTasks,
		); err != nil {
			return nil, mapGameStorageError("scan weekly leader", err)
		}
		leaders = append(leaders, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, mapGameStorageError("iterate weekly leaders", err)
	}
	return leaders, nil
}

func (repository *GameRepository) GetWeeklyPosition(
	ctx context.Context,
	userID uuid.UUID,
	weekStart time.Time,
) (model.LeaderboardEntry, error) {
	var entry model.LeaderboardEntry
	err := executorFromContext(ctx, repository.executor).QueryRow(ctx, `
SELECT position, user_id, pet_name, level, score, completed_tasks
FROM (
    SELECT ROW_NUMBER() OVER (ORDER BY wp.score DESC, wp.user_id) AS position,
           wp.user_id, p.name AS pet_name, p.level, wp.score, wp.completed_tasks
    FROM weekly_progress wp
    JOIN pets p ON p.user_id = wp.user_id
    WHERE wp.week_start = $2
) ranked
WHERE user_id = $1
`, userID, weekStart).Scan(
		&entry.Position, &entry.UserID, &entry.PetName, &entry.Level,
		&entry.Score, &entry.CompletedTasks,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.LeaderboardEntry{}, usecase.ErrLeaderboardEntryNotFound
		}
		return model.LeaderboardEntry{}, mapGameStorageError("get weekly position", err)
	}
	return entry, nil
}

func (repository *GameRepository) UpdateActivityScores(
	ctx context.Context,
	userID uuid.UUID,
	delta usecase.ActivityScoreDelta,
	now time.Time,
) (model.ActivityScores, error) {
	var scores model.ActivityScores
	err := executorFromContext(ctx, repository.executor).QueryRow(ctx, `
INSERT INTO pet_activity_scores (
    user_id, buyer_score, seller_score, auto_score, travel_score,
    real_estate_score, services_score, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (user_id) DO UPDATE
SET buyer_score = pet_activity_scores.buyer_score + EXCLUDED.buyer_score,
    seller_score = pet_activity_scores.seller_score + EXCLUDED.seller_score,
    auto_score = pet_activity_scores.auto_score + EXCLUDED.auto_score,
    travel_score = pet_activity_scores.travel_score + EXCLUDED.travel_score,
    real_estate_score = pet_activity_scores.real_estate_score + EXCLUDED.real_estate_score,
    services_score = pet_activity_scores.services_score + EXCLUDED.services_score,
    updated_at = EXCLUDED.updated_at
RETURNING user_id, buyer_score, seller_score, auto_score, travel_score,
          real_estate_score, services_score, updated_at
`, userID, delta.Buyer, delta.Seller, delta.Auto, delta.Travel,
		delta.RealEstate, delta.Services, now).Scan(
		&scores.UserID, &scores.BuyerScore, &scores.SellerScore, &scores.AutoScore,
		&scores.TravelScore, &scores.RealEstateScore, &scores.ServicesScore, &scores.UpdatedAt,
	)
	if err != nil {
		return model.ActivityScores{}, mapGameStorageError("update activity scores", err)
	}
	return scores, nil
}

func (repository *GameRepository) GetActivityScores(
	ctx context.Context,
	userID uuid.UUID,
) (model.ActivityScores, error) {
	var scores model.ActivityScores
	err := executorFromContext(ctx, repository.executor).QueryRow(ctx, `
SELECT user_id, buyer_score, seller_score, auto_score, travel_score,
       real_estate_score, services_score, updated_at
FROM pet_activity_scores
WHERE user_id = $1
`, userID).Scan(
		&scores.UserID, &scores.BuyerScore, &scores.SellerScore, &scores.AutoScore,
		&scores.TravelScore, &scores.RealEstateScore, &scores.ServicesScore, &scores.UpdatedAt,
	)
	if err != nil {
		return model.ActivityScores{}, mapGameStorageError("get activity scores", err)
	}
	return scores, nil
}

func (repository *GameRepository) UnlockAchievements(
	ctx context.Context,
	userID uuid.UUID,
	codes []string,
	now time.Time,
) ([]model.UserAchievement, error) {
	if len(codes) == 0 {
		return []model.UserAchievement{}, nil
	}

	rows, err := executorFromContext(ctx, repository.executor).Query(ctx, `
INSERT INTO user_achievements (id, user_id, achievement_code, unlocked_at)
SELECT gen_random_uuid(), $1, code, $3
FROM unnest($2::TEXT[]) AS code
ON CONFLICT (user_id, achievement_code) DO NOTHING
RETURNING id, user_id, achievement_code, unlocked_at
`, userID, codes, now)
	if err != nil {
		return nil, mapGameStorageError("unlock achievements", err)
	}
	defer rows.Close()

	unlocked := make([]model.UserAchievement, 0, len(codes))
	for rows.Next() {
		var achievement model.UserAchievement
		if err := rows.Scan(
			&achievement.ID, &achievement.UserID, &achievement.AchievementCode, &achievement.UnlockedAt,
		); err != nil {
			return nil, mapGameStorageError("scan unlocked achievement", err)
		}
		unlocked = append(unlocked, achievement)
	}
	if err := rows.Err(); err != nil {
		return nil, mapGameStorageError("iterate unlocked achievements", err)
	}
	return unlocked, nil
}

func (repository *GameRepository) ListAchievements(
	ctx context.Context,
	userID uuid.UUID,
) ([]model.AchievementProgress, error) {
	rows, err := executorFromContext(ctx, repository.executor).Query(ctx, `
SELECT a.code, a.title, a.description, a.icon_key, a.sort_order, ua.unlocked_at
FROM achievements a
LEFT JOIN user_achievements ua ON ua.achievement_code = a.code AND ua.user_id = $1
ORDER BY a.sort_order, a.code
`, userID)
	if err != nil {
		return nil, mapGameStorageError("list achievements", err)
	}
	defer rows.Close()

	items := make([]model.AchievementProgress, 0)
	for rows.Next() {
		var item model.AchievementProgress
		if err := rows.Scan(
			&item.Achievement.Code, &item.Achievement.Title, &item.Achievement.Description,
			&item.Achievement.IconKey, &item.Achievement.SortOrder, &item.UnlockedAt,
		); err != nil {
			return nil, mapGameStorageError("scan achievement", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapGameStorageError("iterate achievements", err)
	}
	return items, nil
}

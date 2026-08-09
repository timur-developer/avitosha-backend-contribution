package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
)

func (repository *GameRepository) EnsureRewardBalance(
	ctx context.Context,
	userID uuid.UUID,
	rewardType string,
	now time.Time,
) (model.RewardBalance, error) {
	var balance model.RewardBalance
	err := executorFromContext(ctx, repository.executor).QueryRow(ctx, `
INSERT INTO user_reward_balances (
    user_id, reward_type, balance, earned_total, created_at, updated_at
)
VALUES ($1, $2, 0, 0, $3, $3)
ON CONFLICT (user_id, reward_type) DO UPDATE
SET reward_type = EXCLUDED.reward_type
RETURNING user_id, reward_type, balance, earned_total, created_at, updated_at
`, userID, rewardType, now).Scan(
		&balance.UserID, &balance.RewardType, &balance.Balance, &balance.EarnedTotal,
		&balance.CreatedAt, &balance.UpdatedAt,
	)
	if err != nil {
		return model.RewardBalance{}, mapGameStorageError("ensure reward balance", err)
	}
	return balance, nil
}

func (repository *GameRepository) CreditReward(
	ctx context.Context,
	credit model.RewardCredit,
) (model.RewardBalance, bool, error) {
	executor := executorFromContext(ctx, repository.executor)
	initialBalanceAfter := int64(credit.Amount)
	tag, err := executor.Exec(ctx, `
INSERT INTO reward_transactions (
    id, user_id, action_id, task_id, reward_type, amount, balance_after,
    source_kind, source_ref, source_title, created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $11, $7, $8, $9, $10)
ON CONFLICT (action_id, reward_type, source_kind, source_ref) DO NOTHING
`, credit.ID, credit.UserID, credit.ActionID, credit.TaskID, credit.RewardType, credit.Amount,
		credit.SourceKind, credit.SourceRef, credit.SourceTitle, credit.CreatedAt, initialBalanceAfter)
	if err != nil {
		return model.RewardBalance{}, false, mapGameStorageError("insert reward transaction", err)
	}
	if tag.RowsAffected() == 0 {
		balance, getErr := repository.getRewardBalance(ctx, credit.UserID, credit.RewardType)
		return balance, false, getErr
	}

	var balance model.RewardBalance
	err = executor.QueryRow(ctx, `
INSERT INTO user_reward_balances (
    user_id, reward_type, balance, earned_total, created_at, updated_at
)
VALUES ($1, $2, $3, $3, $4, $4)
ON CONFLICT (user_id, reward_type) DO UPDATE
SET balance = user_reward_balances.balance + EXCLUDED.balance,
    earned_total = user_reward_balances.earned_total + EXCLUDED.earned_total,
    updated_at = EXCLUDED.updated_at
RETURNING user_id, reward_type, balance, earned_total, created_at, updated_at
`, credit.UserID, credit.RewardType, credit.Amount, credit.CreatedAt).Scan(
		&balance.UserID, &balance.RewardType, &balance.Balance, &balance.EarnedTotal,
		&balance.CreatedAt, &balance.UpdatedAt,
	)
	if err != nil {
		return model.RewardBalance{}, false, mapGameStorageError("credit reward balance", err)
	}

	tag, err = executor.Exec(ctx, `
UPDATE reward_transactions
SET balance_after = $2
WHERE id = $1
`, credit.ID, balance.Balance)
	if err != nil {
		return model.RewardBalance{}, false, mapGameStorageError("complete reward transaction", err)
	}
	if tag.RowsAffected() != 1 {
		return model.RewardBalance{}, false, fmt.Errorf("complete reward transaction: transaction not found")
	}
	return balance, true, nil
}

func (repository *GameRepository) ListRewardBalances(
	ctx context.Context,
	userID uuid.UUID,
) ([]model.RewardBalance, error) {
	rows, err := executorFromContext(ctx, repository.executor).Query(ctx, `
SELECT user_id, reward_type, balance, earned_total, created_at, updated_at
FROM user_reward_balances
WHERE user_id = $1
ORDER BY reward_type
`, userID)
	if err != nil {
		return nil, mapGameStorageError("list reward balances", err)
	}
	defer rows.Close()

	balances := make([]model.RewardBalance, 0, 1)
	for rows.Next() {
		var balance model.RewardBalance
		if err := rows.Scan(
			&balance.UserID, &balance.RewardType, &balance.Balance, &balance.EarnedTotal,
			&balance.CreatedAt, &balance.UpdatedAt,
		); err != nil {
			return nil, mapGameStorageError("scan reward balance", err)
		}
		balances = append(balances, balance)
	}
	if err := rows.Err(); err != nil {
		return nil, mapGameStorageError("iterate reward balances", err)
	}
	return balances, nil
}

func (repository *GameRepository) getRewardBalance(
	ctx context.Context,
	userID uuid.UUID,
	rewardType string,
) (model.RewardBalance, error) {
	var balance model.RewardBalance
	err := executorFromContext(ctx, repository.executor).QueryRow(ctx, `
SELECT user_id, reward_type, balance, earned_total, created_at, updated_at
FROM user_reward_balances
WHERE user_id = $1 AND reward_type = $2
`, userID, rewardType).Scan(
		&balance.UserID, &balance.RewardType, &balance.Balance, &balance.EarnedTotal,
		&balance.CreatedAt, &balance.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.RewardBalance{}, fmt.Errorf("get reward balance: balance not found")
	}
	if err != nil {
		return model.RewardBalance{}, mapGameStorageError("get reward balance", err)
	}
	return balance, nil
}

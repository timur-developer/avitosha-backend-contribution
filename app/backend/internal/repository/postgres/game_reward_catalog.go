package postgres

import (
	"context"

	"github.com/guitaramust-sudo/Avitosha/app/backend/internal/model"
)

func (repository *GameRepository) ListRewardCatalog(ctx context.Context) ([]model.RewardCatalogItem, error) {
	rows, err := executorFromContext(ctx, repository.executor).Query(ctx, `
SELECT code, title, description, reward_type, perk_type, threshold, sort_order, created_at, updated_at
FROM reward_catalog_items
ORDER BY sort_order, threshold, code
`)
	if err != nil {
		return nil, mapGameStorageError("list reward catalog", err)
	}
	defer rows.Close()

	items := make([]model.RewardCatalogItem, 0)
	for rows.Next() {
		var item model.RewardCatalogItem
		if err := rows.Scan(
			&item.Code, &item.Title, &item.Description, &item.RewardType, &item.PerkType,
			&item.Threshold, &item.SortOrder, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, mapGameStorageError("scan reward catalog item", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapGameStorageError("iterate reward catalog", err)
	}
	return items, nil
}

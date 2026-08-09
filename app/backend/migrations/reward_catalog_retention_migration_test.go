package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestRewardCatalogRetentionMigrationExtendsLedgerAndAddsRetentionTables(t *testing.T) {
	t.Parallel()

	up, err := os.ReadFile("000006_add_reward_catalog_and_retention.up.sql")
	if err != nil {
		t.Fatalf("read retention migration: %v", err)
	}
	for _, fragment := range []string{
		"ALTER TABLE reward_transactions",
		"ADD COLUMN source_kind TEXT NOT NULL DEFAULT 'TASK_COMPLETION'",
		"CREATE TABLE reward_catalog_items",
		"CREATE TABLE user_streaks",
		"CREATE TABLE daily_quest_templates",
		"CREATE TABLE user_daily_quests",
		"'DAILY_QUEST_UPDATED'",
		"'STREAK_UPDATED'",
	} {
		if !strings.Contains(string(up), fragment) {
			t.Errorf("retention migration missing %q", fragment)
		}
	}

	down, err := os.ReadFile("000006_add_reward_catalog_and_retention.down.sql")
	if err != nil {
		t.Fatalf("read retention down migration: %v", err)
	}
	for _, fragment := range []string{
		"DROP TABLE IF EXISTS user_daily_quests",
		"DROP TABLE IF EXISTS daily_quest_templates",
		"DROP TABLE IF EXISTS user_streaks",
		"DROP TABLE IF EXISTS reward_catalog_items",
		"DROP COLUMN IF EXISTS source_kind",
	} {
		if !strings.Contains(string(down), fragment) {
			t.Errorf("retention down migration missing %q", fragment)
		}
	}
}

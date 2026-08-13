package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestDailyQuestSetsMigration(t *testing.T) {
	up, err := os.ReadFile("000011_add_daily_quest_sets.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	for _, fragment := range []string{
		"ADD COLUMN role TEXT", "ADD COLUMN xp_reward", "CREATE TABLE user_daily_goals",
		"ADD COLUMN protection_count",
		"UNIQUE (user_id, quest_date, template_code)", "DAILY_GOAL", "BALANCED_DAY",
		"'BUYER'", "'SELLER'", "'UNIVERSAL'",
	} {
		if !strings.Contains(string(up), fragment) {
			t.Errorf("up migration missing %q", fragment)
		}
	}
	down, err := os.ReadFile("000011_add_daily_quest_sets.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}
	for _, fragment := range []string{"DROP TABLE IF EXISTS user_daily_goals", "DROP COLUMN IF EXISTS role", "user_daily_quests_user_id_quest_date_key"} {
		if !strings.Contains(string(down), fragment) {
			t.Errorf("down migration missing %q", fragment)
		}
	}
}

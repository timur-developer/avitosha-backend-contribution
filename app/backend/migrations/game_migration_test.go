package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestGameMigrationReplacesCareLoopAndSeedsFirstRoom(t *testing.T) {
	up, err := os.ReadFile("000004_rebuild_avitosha_game.up.sql")
	if err != nil {
		t.Fatalf("read game migration: %v", err)
	}

	sql := string(up)
	for _, fragment := range []string{
		"DROP TABLE IF EXISTS inventory_items",
		"DROP TABLE IF EXISTS pet_daily_states",
		"CREATE TABLE tasks",
		"CREATE TABLE user_tasks",
		"CREATE TABLE user_actions",
		"event_id UUID NOT NULL UNIQUE",
		"CREATE TABLE room_items",
		"UNIQUE (user_id, item_code)",
		"CREATE TABLE user_story_progress",
		"CREATE TABLE weekly_progress",
		"score = earned_xp + completed_tasks * 20 + completed_stages * 50",
		"CREATE TABLE daily_progress",
		"CREATE TABLE domain_events",
		"CREATE TABLE pet_activity_scores",
		"('FIRST_ROOM', 'Обустроить первую комнату'",
		"('VIEW_FURNITURE_ADS'",
		"('USE_DELIVERY'",
		"('ROOM_COMPLETE', 'Комната готова'",
	} {
		if !strings.Contains(sql, fragment) {
			t.Errorf("game migration missing %q", fragment)
		}
	}

	for _, removed := range []string{"CREATE TABLE inventory_items", "CREATE TABLE pet_daily_states"} {
		if strings.Contains(sql, removed) {
			t.Errorf("game schema unexpectedly contains legacy marker %q", removed)
		}
	}
}

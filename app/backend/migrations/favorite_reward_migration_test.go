package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestFavoriteRewardMigration(t *testing.T) {
	up, err := os.ReadFile("000012_prevent_favorite_reward_replays.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	for _, fragment := range []string{"CREATE TABLE listing_favorite_rewards", "PRIMARY KEY (user_id, listing_id)"} {
		if !strings.Contains(string(up), fragment) {
			t.Errorf("up migration missing %q", fragment)
		}
	}
	down, err := os.ReadFile("000012_prevent_favorite_reward_replays.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}
	if !strings.Contains(string(down), "DROP TABLE IF EXISTS listing_favorite_rewards") {
		t.Error("down migration does not remove favorite reward history")
	}
}

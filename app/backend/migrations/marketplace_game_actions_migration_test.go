package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestMarketplaceGameActionsMigrationCreatesRulesAndIdempotencyTables(t *testing.T) {
	up, err := os.ReadFile("000008_add_marketplace_game_actions.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	for _, fragment := range []string{"CREATE TABLE product_action_rules", "CREATE TABLE marketplace_game_requests", "CREATE TABLE listing_quality_awards", "LISTING_IMPROVED", "LISTING_SOLD", "quality_score"} {
		if !strings.Contains(string(up), fragment) {
			t.Errorf("migration missing %q", fragment)
		}
	}
	down, err := os.ReadFile("000008_add_marketplace_game_actions.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}
	if !strings.Contains(string(down), "DROP TABLE IF EXISTS marketplace_game_requests") {
		t.Error("down migration must remove request journal")
	}
}

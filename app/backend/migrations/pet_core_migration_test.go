package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestPetCoreMigrationDefinesIntegratedSchema(t *testing.T) {
	up, err := os.ReadFile("000003_create_pet_core.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	down, err := os.ReadFile("000003_create_pet_core.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}

	upSQL := string(up)
	for _, fragment := range []string{
		"CREATE TABLE pets",
		"user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE",
		"UNIQUE (user_id)",
		"CREATE TABLE pet_daily_states",
		"starting_growth_xp INTEGER NOT NULL",
		"UNIQUE (pet_id, date)",
		"CREATE TABLE inventory_items",
		"item_type IN ('FOOD', 'TOY', 'BOOK')",
		"status IN ('AVAILABLE', 'USED', 'EXPIRED')",
		"UNIQUE (user_id, idempotency_key)",
	} {
		if !strings.Contains(upSQL, fragment) {
			t.Errorf("up migration missing %q", fragment)
		}
	}

	downSQL := string(down)
	inventory := strings.Index(downSQL, "DROP TABLE IF EXISTS inventory_items")
	daily := strings.Index(downSQL, "DROP TABLE IF EXISTS pet_daily_states")
	pets := strings.Index(downSQL, "DROP TABLE IF EXISTS pets")
	if inventory < 0 || daily < 0 || pets < 0 || inventory >= daily || daily >= pets {
		t.Fatalf("down migration must drop inventory_items, pet_daily_states, then pets; got:\n%s", downSQL)
	}
}

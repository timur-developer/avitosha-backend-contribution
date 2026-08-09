package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestRewardBalanceMigrationCreatesAuditableIdempotentLedger(t *testing.T) {
	t.Parallel()

	up, err := os.ReadFile("000005_create_reward_balances.up.sql")
	if err != nil {
		t.Fatalf("read reward balance migration: %v", err)
	}
	for _, fragment := range []string{
		"CREATE TABLE user_reward_balances",
		"PRIMARY KEY (user_id, reward_type)",
		"CREATE TABLE reward_transactions",
		"UNIQUE (action_id, task_id, reward_type)",
		"CHECK (balance >= 0)",
		"CHECK (amount > 0)",
	} {
		if !strings.Contains(string(up), fragment) {
			t.Errorf("reward balance migration missing %q", fragment)
		}
	}

	down, err := os.ReadFile("000005_create_reward_balances.down.sql")
	if err != nil {
		t.Fatalf("read reward balance down migration: %v", err)
	}
	if !strings.Contains(string(down), "DROP TABLE IF EXISTS reward_transactions") ||
		!strings.Contains(string(down), "DROP TABLE IF EXISTS user_reward_balances") {
		t.Fatal("reward balance down migration does not remove both tables")
	}
}

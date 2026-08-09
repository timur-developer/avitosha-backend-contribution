package postgres_test

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const truncateTablesQuery = "TRUNCATE TABLE sessions, users RESTART IDENTITY CASCADE"

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	requireTestDatabaseName(t, databaseURL)

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}

	t.Cleanup(pool.Close)

	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("pool.Ping() error = %v", err)
	}

	resetTestDatabase(t, pool)
	return pool
}

func resetTestDatabase(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	if _, err := pool.Exec(context.Background(), truncateTablesQuery); err != nil {
		t.Fatalf("reset test database: %v", err)
	}
}

func requireTestDatabaseName(t *testing.T, databaseURL string) {
	t.Helper()

	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}

	databaseName := strings.TrimPrefix(parsedURL.Path, "/")
	if !strings.Contains(strings.ToLower(databaseName), "test") {
		t.Fatalf("TEST_DATABASE_URL must point to a test database, got %q", databaseName)
	}
}

func stringPointer(value string) *string {
	return &value
}

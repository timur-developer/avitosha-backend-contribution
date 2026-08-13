package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestMarketplaceMigrationCreatesCoreTablesAndSeeds(t *testing.T) {
	up, err := os.ReadFile("000007_add_marketplace.up.sql")
	if err != nil {
		t.Fatalf("read marketplace migration: %v", err)
	}
	for _, fragment := range []string{
		"CREATE TABLE listing_categories", "CREATE TABLE listings", "CREATE TABLE listing_photos",
		"CREATE TABLE listing_favorites", "CREATE TABLE listing_daily_views", "CREATE TABLE listing_messages",
		"CREATE TABLE listing_deals", "PRIMARY KEY (user_id, listing_id, viewed_on)",
		"UNIQUE INDEX idx_listing_messages_first_buyer_contact", "INSERT INTO listing_categories", "INSERT INTO listings",
	} {
		if !strings.Contains(string(up), fragment) {
			t.Errorf("migration missing %q", fragment)
		}
	}
	down, err := os.ReadFile("000007_add_marketplace.down.sql")
	if err != nil {
		t.Fatalf("read marketplace down migration: %v", err)
	}
	for _, fragment := range []string{"DROP TABLE IF EXISTS listing_deals", "DROP TABLE IF EXISTS listing_categories"} {
		if !strings.Contains(string(down), fragment) {
			t.Errorf("down migration missing %q", fragment)
		}
	}
}

func TestDemoDealMigrationScopesDealsToBuyer(t *testing.T) {
	up, err := os.ReadFile("000009_make_demo_deals_user_scoped.up.sql")
	if err != nil {
		t.Fatalf("read demo deal migration: %v", err)
	}
	for _, fragment := range []string{"DROP CONSTRAINT listing_deals_listing_id_key", "UNIQUE (listing_id, buyer_id)"} {
		if !strings.Contains(string(up), fragment) {
			t.Errorf("migration missing %q", fragment)
		}
	}
}

func TestFirstRoomFurnitureSeedContainsThreeAdditionalListings(t *testing.T) {
	up, err := os.ReadFile("000010_add_first_room_furniture_seed.up.sql")
	if err != nil {
		t.Fatalf("read first room seed migration: %v", err)
	}
	for _, id := range []string{"10000000-0000-0000-0000-000000000005", "10000000-0000-0000-0000-000000000006", "10000000-0000-0000-0000-000000000007"} {
		if !strings.Contains(string(up), id) {
			t.Errorf("first room seed missing listing %s", id)
		}
	}
	for _, fragment := range []string{"'FURNITURE'", "'PUBLISHED'", "TRUE", "INSERT INTO listing_photos"} {
		if !strings.Contains(string(up), fragment) {
			t.Errorf("first room seed missing %q", fragment)
		}
	}
}

func TestLegacyDemoPhotoMigrationRepairsEveryHistoricalHost(t *testing.T) {
	up, err := os.ReadFile("000014_repair_legacy_demo_photo_urls.up.sql")
	if err != nil {
		t.Fatalf("read legacy photo repair migration: %v", err)
	}
	for _, fragment := range []string{
		"https://images.example.test/demo/",
		"https://storage.yandexcloud.net/avitosha-demo-images/",
		"/storage/avitosha-photos/demo/",
		"service-1.webp",
	} {
		if !strings.Contains(string(up), fragment) {
			t.Errorf("legacy photo repair migration missing %q", fragment)
		}
	}
}

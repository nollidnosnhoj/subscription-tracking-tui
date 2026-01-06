package service_test

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"subscription-tracker/internal/db"
	"subscription-tracker/internal/service"
)

// testDB holds test database resources
type testDB struct {
	DB                  *sql.DB
	Queries             *db.Queries
	SubscriptionService *service.SubscriptionService
	SpendingService     *service.SpendingService
	ExportService       *service.ExportService
	ConfigService       *service.ConfigService
	SyncService         *service.SyncService
}

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *testDB {
	t.Helper()

	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// Create schema
	schema := `
	CREATE TABLE IF NOT EXISTS subscriptions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		amount REAL NOT NULL,
		currency TEXT NOT NULL DEFAULT 'USD',
		billing_cycle TEXT NOT NULL CHECK (billing_cycle IN ('monthly', 'yearly')),
		next_renewal_date TEXT,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	);
	CREATE INDEX IF NOT EXISTS idx_subscriptions_billing_cycle ON subscriptions(billing_cycle);
	CREATE INDEX IF NOT EXISTS idx_subscriptions_next_renewal ON subscriptions(next_renewal_date);
	
	CREATE TABLE IF NOT EXISTS config (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	INSERT OR IGNORE INTO config (key, value) VALUES ('month_cutoff_day', '1');
	`
	if _, err := database.Exec(schema); err != nil {
		database.Close()
		t.Fatalf("failed to create schema: %v", err)
	}

	queries := db.New(database)
	configService := service.NewConfigService(queries)

	tdb := &testDB{
		DB:                  database,
		Queries:             queries,
		SubscriptionService: service.NewSubscriptionService(queries),
		SpendingService:     service.NewSpendingService(queries, configService),
		ExportService:       service.NewExportService(queries),
		ConfigService:       configService,
		SyncService:         service.NewSyncService(queries, configService),
	}

	t.Cleanup(func() {
		database.Close()
	})

	return tdb
}

// almostEqual checks if two floats are approximately equal
func almostEqual(a, b float64) bool {
	const epsilon = 0.01
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < epsilon
}

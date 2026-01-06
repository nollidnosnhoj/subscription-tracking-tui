package app

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/mattn/go-sqlite3"

	"subscription-tracker/db/migrations"
	"subscription-tracker/internal/db"
	"subscription-tracker/internal/service"
)

type App struct {
	DB                  *sql.DB
	Queries             *db.Queries
	SubscriptionService *service.SubscriptionService
	SpendingService     *service.SpendingService
	ExportService       *service.ExportService
	ConfigService       *service.ConfigService
	SyncService         *service.SyncService
}

func New() (*App, error) {
	dbPath, err := getDBPath()
	if err != nil {
		return nil, err
	}

	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := runMigrations(database); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	queries := db.New(database)
	configService := service.NewConfigService(queries)

	return &App{
		DB:                  database,
		Queries:             queries,
		SubscriptionService: service.NewSubscriptionService(queries),
		SpendingService:     service.NewSpendingService(queries, configService),
		ExportService:       service.NewExportService(queries),
		ConfigService:       configService,
		SyncService:         service.NewSyncService(queries, configService),
	}, nil
}

func (a *App) Close() error {
	return a.DB.Close()
}

func getDBPath() (string, error) {
	// Use XDG data home or fallback to ~/.local/share
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dataHome = filepath.Join(home, ".local", "share")
	}

	appDir := filepath.Join(dataHome, "subscription-tracker")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(appDir, "subscriptions.db"), nil
}

func runMigrations(database *sql.DB) error {
	sourceDriver, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	dbDriver, err := sqlite3.WithInstance(database, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("failed to create database driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "sqlite3", dbDriver)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}

package database

import (
	"database/sql"
	"fmt"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Config struct {
	DatabaseURL string
}

type Migrator interface {
	MigratePostgres(sqlDB *sql.DB) error
}

type defaultMigrator struct{}

func (m defaultMigrator) MigratePostgres(sqlDB *sql.DB) error {
	return RunMigrations(sqlDB, "postgres")
}

func DefaultMigrator() Migrator {
	return defaultMigrator{}
}

func Open(cfg Config, migrator Migrator) (*gorm.DB, error) {
	if migrator == nil {
		migrator = DefaultMigrator()
	}

	databaseURL := strings.TrimSpace(cfg.DatabaseURL)
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required; start Postgres with make test-db-up")
	}

	return openPostgres(databaseURL, migrator)
}

func openPostgres(databaseURL string, migrator Migrator) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, fmt.Errorf("open postgres database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get postgres sql db: %w", err)
	}

	if err = migrator.MigratePostgres(sqlDB); err != nil {
		return nil, err
	}

	return db, nil
}

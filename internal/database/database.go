package database

import (
	"fmt"
	"strings"

	"github.com/svetlyopet/heimdallr/internal/application"
	"github.com/svetlyopet/heimdallr/internal/auth"
	"github.com/svetlyopet/heimdallr/internal/automation"
	"github.com/svetlyopet/heimdallr/internal/job"
	"github.com/svetlyopet/heimdallr/internal/provider"
	"github.com/svetlyopet/heimdallr/internal/release"
	"github.com/svetlyopet/heimdallr/internal/report"
	"github.com/svetlyopet/heimdallr/internal/token"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Config struct {
	DatabaseURL  string
	DatabasePath string
}

func Open(cfg Config) (*gorm.DB, error) {
	if strings.TrimSpace(cfg.DatabaseURL) != "" {
		return openPostgres(cfg.DatabaseURL)
	}

	path := strings.TrimSpace(cfg.DatabasePath)
	if path == "" {
		path = "heimdallr.db"
	}

	return openSQLite(path)
}

func openPostgres(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open postgres database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get postgres sql db: %w", err)
	}

	if err = RunMigrations(sqlDB, "postgres"); err != nil {
		return nil, err
	}

	return db, nil
}

func openSQLite(databasePath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	if err = autoMigrateSQLite(db); err != nil {
		return nil, err
	}

	return db, nil
}

func autoMigrateSQLite(db *gorm.DB) error {
	return db.AutoMigrate(
		&auth.User{},
		&provider.Provider{},
		&automation.Automation{},
		&job.Job{},
		&application.Application{},
		&release.Release{},
		&report.Report{},
		&token.APIToken{},
	)
}

// NewSQLiteDatabase keeps backward compatibility for tests.
func NewSQLiteDatabase(databasePath string) (*gorm.DB, error) {
	return Open(Config{DatabasePath: databasePath})
}

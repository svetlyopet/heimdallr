package database

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/svetlyopet/heimdallr/internal/application"
	"github.com/svetlyopet/heimdallr/internal/auth"
	"github.com/svetlyopet/heimdallr/internal/automation"
	"github.com/svetlyopet/heimdallr/internal/agent"
	"github.com/svetlyopet/heimdallr/internal/job"
	"github.com/svetlyopet/heimdallr/internal/provider"
	"github.com/svetlyopet/heimdallr/internal/release"
	"github.com/svetlyopet/heimdallr/internal/report"
	"github.com/svetlyopet/heimdallr/internal/server"
	"github.com/svetlyopet/heimdallr/internal/token"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Config struct {
	DatabaseURL  string
	DatabasePath string
}

type Migrator interface {
	MigratePostgres(sqlDB *sql.DB) error
	MigrateSQLite(db *gorm.DB) error
}

type defaultMigrator struct{}

func (defaultMigrator) MigratePostgres(sqlDB *sql.DB) error {
	return RunMigrations(sqlDB, "postgres")
}

func (defaultMigrator) MigrateSQLite(db *gorm.DB) error {
	return autoMigrateSQLite(db)
}

func DefaultMigrator() Migrator {
	return defaultMigrator{}
}

func Open(cfg Config, migrator Migrator) (*gorm.DB, error) {
	if migrator == nil {
		migrator = DefaultMigrator()
	}

	if strings.TrimSpace(cfg.DatabaseURL) != "" {
		return openPostgres(cfg.DatabaseURL, migrator)
	}

	path := strings.TrimSpace(cfg.DatabasePath)
	if path == "" {
		path = "heimdallr.db"
	}

	return openSQLite(path, migrator)
}

func openPostgres(databaseURL string, migrator Migrator) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
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

func openSQLite(databasePath string, migrator Migrator) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	if err = migrator.MigrateSQLite(db); err != nil {
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
		&server.Server{},
		&agent.Agent{},
		&server.ServerJob{},
		&server.ServerRelease{},
	)
}

// NewSQLiteDatabase keeps backward compatibility for tests.
func NewSQLiteDatabase(databasePath string) (*gorm.DB, error) {
	return Open(Config{DatabasePath: databasePath}, nil)
}

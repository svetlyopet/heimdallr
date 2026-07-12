package database

import (
	"database/sql"
	"fmt"
	"strings"

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

type defaultMigrator struct {
	sqliteMigrator func(*gorm.DB) error
}

func (m defaultMigrator) MigratePostgres(sqlDB *sql.DB) error {
	return RunMigrations(sqlDB, "postgres")
}

func (m defaultMigrator) MigrateSQLite(db *gorm.DB) error {
	if m.sqliteMigrator == nil {
		return fmt.Errorf("sqlite migration is not configured")
	}

	return m.sqliteMigrator(db)
}

func DefaultMigrator() Migrator {
	return defaultMigrator{}
}

func NewMigrator(sqliteMigrate func(*gorm.DB) error) Migrator {
	return defaultMigrator{sqliteMigrator: sqliteMigrate}
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

func openSQLite(databasePath string, migrator Migrator) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	if err = migrator.MigrateSQLite(db); err != nil {
		return nil, err
	}

	return db, nil
}

// NewSQLiteDatabase keeps backward compatibility for tests.
func NewSQLiteDatabase(databasePath string) (*gorm.DB, error) {
	return Open(Config{DatabasePath: databasePath}, nil)
}

package database

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func RunMigrations(sqlDB *sql.DB, driverName string) error {
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("create migration source: %w", err)
	}

	var databaseDriver database.Driver
	switch driverName {
	case "postgres":
		databaseDriver, err = postgres.WithInstance(sqlDB, &postgres.Config{})
	case "sqlite":
		databaseDriver, err = sqlite.WithInstance(sqlDB, &sqlite.Config{})
	default:
		return fmt.Errorf("unsupported migration driver: %s", driverName)
	}
	if err != nil {
		return fmt.Errorf("create migration database driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, driverName, databaseDriver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

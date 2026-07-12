package database

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	pgx5 "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

const migrationDriverName = "pgx5"

func RunMigrations(sqlDB *sql.DB, driverName string) error {
	if driverName != "postgres" && driverName != migrationDriverName {
		return fmt.Errorf("unsupported migration driver: %s", driverName)
	}

	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("create migration source: %w", err)
	}

	databaseDriver, err := pgx5.WithInstance(sqlDB, &pgx5.Config{})
	if err != nil {
		return fmt.Errorf("create migration database driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, migrationDriverName, databaseDriver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

func RunMigrationsTo(sqlDB *sql.DB, version uint) error {
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("create migration source: %w", err)
	}

	databaseDriver, err := pgx5.WithInstance(sqlDB, &pgx5.Config{})
	if err != nil {
		return fmt.Errorf("create migration database driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, migrationDriverName, databaseDriver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	if err = m.Migrate(version); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations to version %d: %w", version, err)
	}

	return nil
}

package database_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/database"
	"github.com/svetlyopet/heimdallr/internal/testutil"
)

type recordingMigrator struct {
	postgresCalled bool
}

func (m *recordingMigrator) MigratePostgres(sqlDB *sql.DB) error {
	m.postgresCalled = true
	return nil
}

func TestOpenRequiresDatabaseURL(t *testing.T) {
	_, err := database.Open(database.Config{}, database.DefaultMigrator())
	require.Error(t, err)
	require.Contains(t, err.Error(), "DATABASE_URL is required")
}

func TestOpenPostgresUsesMigrator(t *testing.T) {
	migrator := &recordingMigrator{}

	db, err := database.Open(
		database.Config{DatabaseURL: testutil.PostgresURL(t)},
		migrator,
	)
	require.NoError(t, err)
	require.NotNil(t, db)
	require.True(t, migrator.postgresCalled)
}

func TestOpenGormDBImplementsShutdown(t *testing.T) {
	gormDB, err := database.OpenGormDB(
		database.Config{DatabaseURL: testutil.PostgresDatabaseURL(t)},
		database.DefaultMigrator(),
	)
	require.NoError(t, err)

	require.NoError(t, gormDB.Shutdown(t.Context()))
}

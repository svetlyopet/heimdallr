package database_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/database"
	"gorm.io/gorm"
)

type recordingMigrator struct {
	sqliteCalled   bool
	postgresCalled bool
}

func (m *recordingMigrator) MigratePostgres(sqlDB *sql.DB) error {
	m.postgresCalled = true
	return nil
}

func (m *recordingMigrator) MigrateSQLite(db *gorm.DB) error {
	m.sqliteCalled = true
	return nil
}

func TestOpenSQLiteUsesMigrator(t *testing.T) {
	migrator := &recordingMigrator{}

	db, err := database.Open(database.Config{DatabasePath: ":memory:"}, migrator)
	require.NoError(t, err)
	require.NotNil(t, db)
	require.True(t, migrator.sqliteCalled)
	require.False(t, migrator.postgresCalled)
}

func TestOpenGormDBImplementsShutdown(t *testing.T) {
	gormDB, err := database.OpenGormDB(database.Config{DatabasePath: ":memory:"}, database.DefaultMigrator())
	require.NoError(t, err)

	require.NoError(t, gormDB.Shutdown(t.Context()))
}

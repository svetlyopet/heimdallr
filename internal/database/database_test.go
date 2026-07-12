package database_test

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/database"
	"github.com/svetlyopet/heimdallr/internal/token"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type recordingMigrator struct {
	sqliteCalled   bool
	postgresCalled bool
}

type legacyAPIToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key"`
	Name      string    `gorm:"type:varchar(255);not null;check:name <> ''"`
	TokenHash string    `gorm:"type:char(64);not null;uniqueIndex"`
	Scopes    []string  `gorm:"serializer:json;type:text;not null"`
	Kind      string    `gorm:"type:varchar(32);not null;default:api"`
	ExpiresAt *time.Time
	CreatedBy *uuid.UUID `gorm:"type:uuid"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (legacyAPIToken) TableName() string {
	return "api_tokens"
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

func TestSQLiteMigrationBackfillsTokenExpiration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	legacyDB, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, legacyDB.AutoMigrate(&legacyAPIToken{}))
	require.NoError(t, legacyDB.Create(&legacyAPIToken{
		ID:        uuid.New(),
		Name:      "legacy-token",
		TokenHash: string(make([]byte, 64)),
		Scopes:    []string{"read"},
		Kind:      token.TokenKindAPI,
	}).Error)
	legacySQLDB, err := legacyDB.DB()
	require.NoError(t, err)
	require.NoError(t, legacySQLDB.Close())

	migratedDB, err := database.Open(
		database.Config{DatabasePath: path},
		database.DefaultMigrator(),
	)
	require.NoError(t, err)

	var migrated token.APIToken
	require.NoError(t, migratedDB.First(&migrated).Error)
	require.NotNil(t, migrated.ExpiresAt)
	require.WithinDuration(t, time.Now().UTC().Add(90*24*time.Hour), *migrated.ExpiresAt, time.Minute)
}

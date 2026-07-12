//go:build integration

package integration

import (
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/database"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestPostgresTokenExpirationMigration(t *testing.T) {
	databaseURL := os.Getenv("TEST_POSTGRES_URL")
	if databaseURL == "" {
		t.Skip("TEST_POSTGRES_URL is not configured")
	}

	adminDB, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	require.NoError(t, err)
	adminSQLDB, err := adminDB.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, adminSQLDB.Close())
	})

	const testSchema = "heimdallr_token_migration_test"
	require.NoError(t, adminDB.Exec("DROP SCHEMA IF EXISTS heimdallr_token_migration_test CASCADE").Error)
	require.NoError(t, adminDB.Exec("CREATE SCHEMA heimdallr_token_migration_test").Error)
	t.Cleanup(func() {
		require.NoError(t, adminDB.Exec("DROP SCHEMA IF EXISTS heimdallr_token_migration_test CASCADE").Error)
	})

	parsedURL, err := url.Parse(databaseURL)
	require.NoError(t, err)
	query := parsedURL.Query()
	query.Set("search_path", testSchema)
	parsedURL.RawQuery = query.Encode()

	migrationDB, err := gorm.Open(postgres.Open(parsedURL.String()), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := migrationDB.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	require.NoError(t, migrationDB.Exec(`
		CREATE TABLE api_tokens (
			id UUID PRIMARY KEY,
			kind VARCHAR(32) NOT NULL,
			expires_at TIMESTAMPTZ NULL
		)
	`).Error)
	require.NoError(t, migrationDB.Exec(`
		CREATE TABLE schema_migrations (
			version BIGINT NOT NULL PRIMARY KEY,
			dirty BOOLEAN NOT NULL
		)
	`).Error)
	require.NoError(t, migrationDB.Exec("INSERT INTO schema_migrations (version, dirty) VALUES (8, FALSE)").Error)

	sessionID := uuid.New()
	apiTokenID := uuid.New()
	require.NoError(t, migrationDB.Exec(
		"INSERT INTO api_tokens (id, kind, expires_at) VALUES (?, 'session', NULL), (?, 'api', NULL)",
		sessionID,
		apiTokenID,
	).Error)

	migrationStartedAt := time.Now().UTC()
	require.NoError(t, database.RunMigrations(sqlDB, "postgres"))

	var sessionExpiry time.Time
	require.NoError(t, migrationDB.Raw(
		"SELECT expires_at FROM api_tokens WHERE id = ?",
		sessionID,
	).Scan(&sessionExpiry).Error)
	require.WithinDuration(t, migrationStartedAt.Add(24*time.Hour), sessionExpiry, time.Minute)

	var apiTokenExpiry time.Time
	require.NoError(t, migrationDB.Raw(
		"SELECT expires_at FROM api_tokens WHERE id = ?",
		apiTokenID,
	).Scan(&apiTokenExpiry).Error)
	require.WithinDuration(t, migrationStartedAt.Add(90*24*time.Hour), apiTokenExpiry, time.Minute)

	insertWithoutExpiry := migrationDB.Exec(
		"INSERT INTO api_tokens (id, kind, expires_at) VALUES (?, 'api', NULL)",
		uuid.New(),
	).Error
	require.Error(t, insertWithoutExpiry)
}

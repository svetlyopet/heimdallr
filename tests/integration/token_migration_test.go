//go:build integration

package integration

import (
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/database"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestPostgresTokenExpirationMigration(t *testing.T) {
	adminURL := testutil.PostgresURL(t)
	schemaName := "heimdallr_token_migration_test"

	adminDB, err := gorm.Open(postgres.Open(adminURL), &gorm.Config{})
	require.NoError(t, err)
	adminSQLDB, err := adminDB.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = adminSQLDB.Close()
	})

	require.NoError(t, adminDB.Exec("DROP SCHEMA IF EXISTS heimdallr_token_migration_test CASCADE").Error)
	require.NoError(t, adminDB.Exec("CREATE SCHEMA heimdallr_token_migration_test").Error)
	t.Cleanup(func() {
		require.NoError(t, adminDB.Exec("DROP SCHEMA IF EXISTS heimdallr_token_migration_test CASCADE").Error)
	})

	parsedURL, err := url.Parse(adminURL)
	require.NoError(t, err)
	query := parsedURL.Query()
	query.Set("search_path", schemaName)
	parsedURL.RawQuery = query.Encode()

	migrationDB, err := gorm.Open(postgres.Open(parsedURL.String()), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := migrationDB.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	require.NoError(t, database.RunMigrationsTo(sqlDB, 8))

	sessionID := uuid.New()
	apiTokenID := uuid.New()
	tokenHash := "0000000000000000000000000000000000000000000000000000000000000000"
	apiTokenHash := "1111111111111111111111111111111111111111111111111111111111111111"
	require.NoError(t, migrationDB.Exec(
		`INSERT INTO api_tokens (id, name, token_hash, scopes, kind, expires_at)
		 VALUES (?, 'session-token', ?, '[]', 'session', NULL), (?, 'api-token', ?, '["read"]', 'api', NULL)`,
		sessionID,
		tokenHash,
		apiTokenID,
		apiTokenHash,
	).Error)

	migrationStartedAt := time.Now().UTC()
	require.NoError(t, database.RunMigrationsTo(sqlDB, 9))

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
		`INSERT INTO api_tokens (id, name, token_hash, scopes, kind, expires_at)
		 VALUES (?, 'legacy', ?, '["read"]', 'api', NULL)`,
		uuid.New(),
		"2222222222222222222222222222222222222222222222222222222222222222",
	).Error
	require.Error(t, insertWithoutExpiry)
}

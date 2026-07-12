package testutil

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/database"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func PostgresURL(t *testing.T) string {
	t.Helper()

	databaseURL := strings.TrimSpace(os.Getenv("TEST_POSTGRES_URL"))
	if databaseURL == "" {
		t.Fatal("TEST_POSTGRES_URL is not configured; start Postgres with make test-db-up")
	}

	return databaseURL
}

func testSchemaName(t *testing.T) string {
	t.Helper()

	hash := sha256.Sum256([]byte(t.Name()))
	return "t_" + hex.EncodeToString(hash[:16])
}

func NewPostgresDB(t *testing.T) *gorm.DB {
	t.Helper()

	adminURL := PostgresURL(t)
	schemaName := testSchemaName(t)

	adminDB, err := gorm.Open(postgres.Open(adminURL), &gorm.Config{TranslateError: true})
	require.NoError(t, err)
	adminSQLDB, err := adminDB.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = adminSQLDB.Close()
	})

	require.NoError(t, adminDB.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schemaName)).Error)
	require.NoError(t, adminDB.Exec(fmt.Sprintf("CREATE SCHEMA %s", schemaName)).Error)
	t.Cleanup(func() {
		require.NoError(t, adminDB.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schemaName)).Error)
	})

	parsedURL, err := url.Parse(adminURL)
	require.NoError(t, err)
	query := parsedURL.Query()
	query.Set("search_path", schemaName)
	parsedURL.RawQuery = query.Encode()

	testDB, err := gorm.Open(postgres.Open(parsedURL.String()), &gorm.Config{TranslateError: true})
	require.NoError(t, err)
	sqlDB, err := testDB.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	require.NoError(t, database.RunMigrations(sqlDB, "postgres"))

	return testDB
}

func PostgresDatabaseURL(t *testing.T) string {
	t.Helper()

	adminURL := PostgresURL(t)
	schemaName := testSchemaName(t)

	adminDB, err := gorm.Open(postgres.Open(adminURL), &gorm.Config{TranslateError: true})
	require.NoError(t, err)
	adminSQLDB, err := adminDB.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = adminSQLDB.Close()
	})

	require.NoError(t, adminDB.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schemaName)).Error)
	require.NoError(t, adminDB.Exec(fmt.Sprintf("CREATE SCHEMA %s", schemaName)).Error)
	t.Cleanup(func() {
		require.NoError(t, adminDB.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schemaName)).Error)
	})

	parsedURL, err := url.Parse(adminURL)
	require.NoError(t, err)
	query := parsedURL.Query()
	query.Set("search_path", schemaName)
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String()
}

package testutil

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"runtime"
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

func schemaNameFromQualified(qualified string) string {
	hash := sha256.Sum256([]byte(qualified))
	return "t_" + hex.EncodeToString(hash[:16])
}

func testSchemaName(t *testing.T) string {
	t.Helper()

	var pcs [32]uintptr
	n := runtime.Callers(2, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if isTestFunction(frame.Function) {
			return schemaNameFromQualified(frame.Function)
		}
		if !more {
			break
		}
	}

	return schemaNameFromQualified(t.Name())
}

func isTestFunction(name string) bool {
	i := strings.LastIndexByte(name, '.')
	if i < 0 {
		return false
	}

	return strings.HasPrefix(name[i+1:], "Test")
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

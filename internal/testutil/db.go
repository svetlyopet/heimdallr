package testutil

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewSQLiteDB(t *testing.T, models ...any) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{TranslateError: true})
	require.NoError(t, err)

	if len(models) > 0 {
		require.NoError(t, db.AutoMigrate(models...))
	}

	return db
}

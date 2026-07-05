package automation_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/automation"
	"github.com/svetlyopet/heimdallr/internal/provider"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	"github.com/svetlyopet/heimdallr/tests/testfixtures"
	"gorm.io/gorm"
)

func newAutomationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testutil.NewSQLiteDB(t, &provider.Provider{}, &automation.Automation{})
}

func TestRepositoryFindAllReturnsAutomations(t *testing.T) {
	db := newAutomationTestDB(t)
	repo := automation.NewRepository(db)

	fixture := testfixtures.SeedProviderAutomation(t, db, "awx", "deploy")

	automations, total, err := repo.FindAll(context.Background(), 10, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, fixture.Automation.ID, automations[0].ID)
}

func TestRepositoryFindByIdReturnsNotFound(t *testing.T) {
	db := newAutomationTestDB(t)
	repo := automation.NewRepository(db)

	_, err := repo.FindById(context.Background(), "00000000-0000-0000-0000-000000000001")
	require.Error(t, err)
}

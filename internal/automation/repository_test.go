package automation_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/automation"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	"github.com/svetlyopet/heimdallr/tests/testfixtures"
	"gorm.io/gorm"
)

func newAutomationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testutil.NewPostgresDB(t)
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

func TestRepositoryDeleteReturnsNotFoundForUnknownID(t *testing.T) {
	db := newAutomationTestDB(t)
	repo := automation.NewRepository(db)

	err := repo.Delete(context.Background(), "00000000-0000-0000-0000-000000000001")
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestRepositoryFindAllExcludesDeletedAutomations(t *testing.T) {
	db := newAutomationTestDB(t)
	repo := automation.NewRepository(db)

	fixture := testfixtures.SeedProviderAutomation(t, db, "awx", "deploy")
	require.NoError(t, repo.Delete(context.Background(), fixture.Automation.ID.String()))

	automations, total, err := repo.FindAll(context.Background(), 10, 0)
	require.NoError(t, err)
	require.Equal(t, int64(0), total)
	require.Empty(t, automations)
}

package job_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/modules/automation"
	"github.com/svetlyopet/heimdallr/internal/modules/job"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	"github.com/svetlyopet/heimdallr/tests/testfixtures"
	"gorm.io/gorm"
)

func newJobTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testutil.NewPostgresDB(t)
}

func TestRepositoryCreateAndFindById(t *testing.T) {
	db := newJobTestDB(t)
	repo := job.NewRepository(db)

	fixture := testfixtures.SeedProviderAutomation(t, db, "awx", "deploy")

	created, err := repo.Create(context.Background(), job.Job{
		ID:           "1000",
		AutomationID: fixture.Automation.ID,
		Automation:   fixture.Automation.Name,
		Provider:     fixture.Provider.Name,
		ProviderID:   fixture.Provider.ID,
		Status:       "success",
		Location:     "global",
		Url:          "https://example.com/#/jobs/playbook/200",
		Metadata:     []byte(`{"inventory":"true"}`),
		Output:       "dGVzdA==",
	})
	require.NoError(t, err)

	found, err := repo.FindById(context.Background(), created.ID, fixture.Automation.ID.String())
	require.NoError(t, err)
	require.Equal(t, "success", found.Status)
	require.Equal(t, "dGVzdA==", found.Output)
}

func TestRepositoryFindByIdReturnsNotFoundForDeletedAutomation(t *testing.T) {
	db := newJobTestDB(t)
	repo := job.NewRepository(db)
	automationRepo := automation.NewRepository(db)

	fixture := testfixtures.SeedProviderAutomation(t, db, "awx", "deploy")
	created, err := repo.Create(context.Background(), job.Job{
		ID:           "1002",
		AutomationID: fixture.Automation.ID,
		Automation:   fixture.Automation.Name,
		Provider:     fixture.Provider.Name,
		ProviderID:   fixture.Provider.ID,
		Status:       "success",
		Location:     "global",
		Url:          "https://example.com/#/jobs/playbook/202",
		Metadata:     []byte(`{}`),
		Output:       "dGVzdA==",
	})
	require.NoError(t, err)

	require.NoError(t, automationRepo.Delete(context.Background(), fixture.Automation.ID.String()))

	_, err = repo.FindById(context.Background(), created.ID, fixture.Automation.ID.String())
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	var rawCount int64
	require.NoError(t, db.Unscoped().Model(&job.Job{}).Where("id = ?", created.ID).Count(&rawCount).Error)
	require.Equal(t, int64(1), rawCount)
}

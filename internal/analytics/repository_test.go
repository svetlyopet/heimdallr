package analytics

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/automation"
	"github.com/svetlyopet/heimdallr/internal/job"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAnalyticsTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(&automation.Automation{}, &job.Job{}))

	return db
}

func createAutomation(t *testing.T, db *gorm.DB, name string) automation.Automation {
	t.Helper()

	a := automation.Automation{
		Name:       name,
		Url:        "https://example.com/automation",
		Provider:   "awx",
		ProviderID: uuidMustParse(t, "e9a411c9-f862-4f7d-9e8e-e54f46fc06f1"),
	}

	require.NoError(t, db.Create(&a).Error)

	return a
}

func createJob(t *testing.T, db *gorm.DB, id string, automationModel automation.Automation, status string, location string) {
	t.Helper()

	j := job.Job{
		ID:           id,
		Automation:   automationModel.Name,
		AutomationID: automationModel.ID,
		Provider:     automationModel.Provider,
		ProviderID:   automationModel.ProviderID,
		Status:       status,
		Location:     location,
		Url:          "https://example.com/jobs/" + id,
		Metadata:     []byte(`{"ok":true}`),
		Output:       "output",
	}

	require.NoError(t, db.Create(&j).Error)
}

func uuidMustParse(t *testing.T, raw string) uuid.UUID {
	t.Helper()

	u, err := uuid.Parse(raw)
	require.NoError(t, err)

	return u
}

func TestRepositoryGetAutomationOverviewAggregatesData(t *testing.T) {
	db := newAnalyticsTestDB(t)
	repo := NewRepository(db)

	a1 := createAutomation(t, db, "Deploy app")
	a2 := createAutomation(t, db, "Restart service")

	createJob(t, db, "job-1", a1, "success", "eu")
	createJob(t, db, "job-2", a1, "failed", "eu")
	createJob(t, db, "job-3", a2, "started", "us")

	response, err := repo.GetAutomationOverview(t.Context())
	require.NoError(t, err)
	require.Equal(t, 2, response.TotalAutomations)
	require.Equal(t, 3, response.TotalJobs)
	require.Equal(t, 1, response.SuccessfulJobs)
	require.Equal(t, 1, response.FailedJobs)
	require.Equal(t, 1, response.StartedJobs)
	require.Equal(t, 33.33333333333333, response.SuccessRate)
	require.Len(t, response.ByLocation, 2)
	require.Len(t, response.ByAutomation, 2)
}

func TestRepositoryGetAutomationOverviewByIDNotFound(t *testing.T) {
	db := newAnalyticsTestDB(t)
	repo := NewRepository(db)

	_, err := repo.GetAutomationOverviewByID(t.Context(), "5d8dd803-fca6-4f7c-9dd2-24417622d630")
	require.ErrorIs(t, err, ErrAutomationNotFound)
}

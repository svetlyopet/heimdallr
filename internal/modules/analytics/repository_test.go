package analytics

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/modules/application"
	"github.com/svetlyopet/heimdallr/internal/modules/automation"
	"github.com/svetlyopet/heimdallr/internal/modules/job"
	"github.com/svetlyopet/heimdallr/internal/modules/provider"
	"github.com/svetlyopet/heimdallr/internal/modules/release"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	"gorm.io/gorm"
)

func newAnalyticsTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	return testutil.NewPostgresDB(t)
}

func createAutomation(t *testing.T, db *gorm.DB, name string) automation.Automation {
	t.Helper()

	providerID := uuidMustParse(t, "e9a411c9-f862-4f7d-9e8e-e54f46fc06f1")
	require.NoError(t, db.FirstOrCreate(&provider.Provider{
		ID:   providerID,
		Name: "awx",
		Url:  "https://example.com/provider",
	}).Error)

	a := automation.Automation{
		Name:       name,
		Url:        "https://example.com/automation",
		Provider:   "awx",
		ProviderID: providerID,
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
	createJob(t, db, "job-3", a2, "skipped", "us")

	response, err := repo.GetAutomationOverview(t.Context())
	require.NoError(t, err)
	require.Equal(t, 2, response.TotalAutomations)
	require.Equal(t, 3, response.TotalJobs)
	require.Equal(t, 1, response.SuccessfulJobs)
	require.Equal(t, 1, response.FailedJobs)
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

func TestRepositoryGetAutomationOverviewExcludesDeletedAutomation(t *testing.T) {
	db := newAnalyticsTestDB(t)
	repo := NewRepository(db)

	a1 := createAutomation(t, db, "Deploy app")
	createJob(t, db, "job-1", a1, "success", "eu")
	require.NoError(t, db.Delete(&a1).Error)

	response, err := repo.GetAutomationOverview(t.Context())
	require.NoError(t, err)
	require.Equal(t, 0, response.TotalAutomations)
	require.Equal(t, 0, response.TotalJobs)
}

func newComplianceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	return testutil.NewPostgresDB(t)
}

func TestRepositoryGetComplianceOverviewSelectsHigherIDOnTimestampTie(t *testing.T) {
	db := newComplianceTestDB(t)
	repo := NewRepository(db)

	app := application.Application{Name: "payments", Description: "desc", RepositoryURL: "https://example.com/payments"}
	require.NoError(t, db.Create(&app).Error)

	createdAt := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	lowerID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	higherID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	require.NoError(t, db.Create(&release.Release{
		ID:            lowerID,
		ApplicationID: app.ID,
		Application:   app.Name,
		Version:       "1.0.0",
		CommitSHA:     "abc",
		PipelineURL:   "https://example.com/pipeline/1",
		Branch:        "main",
	}).Error)
	require.NoError(t, db.Model(&release.Release{}).Where("id = ?", lowerID).Updates(map[string]interface{}{
		"created_at": createdAt,
		"updated_at": createdAt,
	}).Error)

	require.NoError(t, db.Create(&release.Release{
		ID:            higherID,
		ApplicationID: app.ID,
		Application:   app.Name,
		Version:       "1.0.1",
		CommitSHA:     "def",
		PipelineURL:   "https://example.com/pipeline/2",
		Branch:        "main",
	}).Error)
	require.NoError(t, db.Model(&release.Release{}).Where("id = ?", higherID).Updates(map[string]interface{}{
		"created_at": createdAt,
		"updated_at": createdAt,
	}).Error)

	response, err := repo.GetComplianceOverview(t.Context())
	require.NoError(t, err)
	require.Len(t, response.ByApplication, 1)
	require.Equal(t, higherID.String(), response.ByApplication[0].LatestReleaseId)
	require.Equal(t, "1.0.1", response.ByApplication[0].LatestVersion)
}

func TestRepositoryGetComplianceOverviewIgnoresDeletedLatestRelease(t *testing.T) {
	db := newComplianceTestDB(t)
	repo := NewRepository(db)

	app := application.Application{Name: "billing", Description: "desc", RepositoryURL: "https://example.com/billing"}
	require.NoError(t, db.Create(&app).Error)

	older := release.Release{
		ID:            uuid.New(),
		ApplicationID: app.ID,
		Application:   app.Name,
		Version:       "1.0.0",
		CommitSHA:     "abc",
		PipelineURL:   "https://example.com/pipeline/1",
		Branch:        "main",
	}
	newer := release.Release{
		ID:            uuid.New(),
		ApplicationID: app.ID,
		Application:   app.Name,
		Version:       "2.0.0",
		CommitSHA:     "def",
		PipelineURL:   "https://example.com/pipeline/2",
		Branch:        "main",
	}
	require.NoError(t, db.Create(&older).Error)
	require.NoError(t, db.Create(&newer).Error)
	require.NoError(t, db.Delete(&newer).Error)

	response, err := repo.GetComplianceOverview(t.Context())
	require.NoError(t, err)
	require.Len(t, response.ByApplication, 1)
	require.Equal(t, older.ID.String(), response.ByApplication[0].LatestReleaseId)
}

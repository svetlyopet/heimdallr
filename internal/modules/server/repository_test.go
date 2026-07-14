package server

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/modules/application"
	"github.com/svetlyopet/heimdallr/internal/modules/automation"
	"github.com/svetlyopet/heimdallr/internal/modules/job"
	"github.com/svetlyopet/heimdallr/internal/modules/provider"
	"github.com/svetlyopet/heimdallr/internal/modules/release"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type testAgent struct {
	ID       uuid.UUID      `gorm:"type:uuid;primary_key"`
	Name     string         `gorm:"type:varchar(255);not null"`
	Type     string         `gorm:"type:varchar(255);not null"`
	Metadata datatypes.JSON `gorm:"type:json;not null"`
}

func (testAgent) TableName() string {
	return "agents"
}

type testServerAgent struct {
	ServerID uuid.UUID `gorm:"type:uuid;primaryKey"`
	AgentID  uuid.UUID `gorm:"type:uuid;primaryKey"`
}

func (testServerAgent) TableName() string {
	return "server_agents"
}

func newServerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	return testutil.NewPostgresDB(t)
}

func TestRepositoryFindAllReturnsServers(t *testing.T) {
	db := newServerTestDB(t)
	repo := NewRepository(db)

	created, err := repo.Create(context.Background(), Server{
		ID:              uuid.New(),
		Hostname:        "web-01.example.com",
		Metadata:        datatypes.JSON([]byte(`{}`)),
		OperatingSystem: "linux",
		Hypervisor:      "vmware",
		Location:        "dc1",
	})
	require.NoError(t, err)

	servers, total, err := repo.FindAll(context.Background(), "", 10, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, servers, 1)
	require.Equal(t, created.ID, servers[0].ID)
	require.Equal(t, int64(0), servers[0].AgentCount)
}

func TestRepositoryFindByIdReturnsNotFound(t *testing.T) {
	db := newServerTestDB(t)
	repo := NewRepository(db)

	_, err := repo.FindById(context.Background(), uuid.New().String())
	require.Error(t, err)
}

func TestRepositoryGetRelationCountsIncludesAgents(t *testing.T) {
	db := newServerTestDB(t)
	repo := NewRepository(db)

	serverEntity, err := repo.Create(context.Background(), Server{
		ID:              uuid.New(),
		Hostname:        "db-01.example.com",
		Metadata:        datatypes.JSON([]byte(`{}`)),
		OperatingSystem: "linux",
		Hypervisor:      "kvm",
		Location:        "dc2",
	})
	require.NoError(t, err)

	agentID := uuid.New()
	require.NoError(t, db.Create(&testAgent{
		ID:       agentID,
		Name:     "crowdstrike",
		Type:     "security",
		Metadata: datatypes.JSON([]byte(`{}`)),
	}).Error)
	require.NoError(t, db.Create(&testServerAgent{
		ServerID: serverEntity.ID,
		AgentID:  agentID,
	}).Error)

	counts, err := repo.GetRelationCounts(context.Background(), serverEntity.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), counts.AgentCount)
}

func TestRepositoryAssociateJobRequiresExistingJob(t *testing.T) {
	db := newServerTestDB(t)
	repo := NewRepository(db)

	serverEntity, err := repo.Create(context.Background(), Server{
		ID:              uuid.New(),
		Hostname:        "app-01.example.com",
		Metadata:        datatypes.JSON([]byte(`{}`)),
		OperatingSystem: "linux",
	})
	require.NoError(t, err)

	automationID := uuid.New()
	exists, err := repo.JobExists(context.Background(), "job-1", automationID)
	require.NoError(t, err)
	require.False(t, exists)

	providerID := seedProvider(t, db)
	seedAutomation(t, db, automationID, providerID)
	seedJob(t, db, "job-1", automationID, providerID)

	exists, err = repo.JobExists(context.Background(), "job-1", automationID)
	require.NoError(t, err)
	require.True(t, exists)

	err = repo.CreateJobAssociation(context.Background(), ServerJob{
		ServerID:     serverEntity.ID,
		JobID:        "job-1",
		AutomationID: automationID,
	})
	require.NoError(t, err)

	associated, err := repo.JobAssociationExists(context.Background(), serverEntity.ID, "job-1", automationID)
	require.NoError(t, err)
	require.True(t, associated)

	jobs, total, err := repo.FindAssociatedJobs(context.Background(), serverEntity.ID.String(), 10, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, jobs, 1)
	require.Equal(t, "job-1", jobs[0].JobID)
}

func TestRepositoryAssociateReleaseRequiresExistingRelease(t *testing.T) {
	db := newServerTestDB(t)
	repo := NewRepository(db)

	serverEntity, err := repo.Create(context.Background(), Server{
		ID:       uuid.New(),
		Hostname: "rel-01.example.com",
		Metadata: datatypes.JSON([]byte(`{}`)),
	})
	require.NoError(t, err)

	appID := uuid.New()
	releaseID := uuid.New()
	seedApplication(t, db, appID)
	seedRelease(t, db, releaseID, appID)

	exists, err := repo.ReleaseExists(context.Background(), releaseID, appID)
	require.NoError(t, err)
	require.True(t, exists)

	err = repo.CreateReleaseAssociation(context.Background(), ServerRelease{
		ServerID:      serverEntity.ID,
		ReleaseID:     releaseID,
		ApplicationID: appID,
	})
	require.NoError(t, err)

	releases, total, err := repo.FindAssociatedReleases(context.Background(), serverEntity.ID.String(), 10, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, releases, 1)
	require.Equal(t, releaseID, releases[0].ReleaseID)
}

func seedProvider(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()

	id := uuid.New()
	require.NoError(t, db.Create(&provider.Provider{
		ID:   id,
		Name: "ansible",
		Url:  "https://ansible.example.com",
	}).Error)

	return id
}

func seedAutomation(t *testing.T, db *gorm.DB, automationID uuid.UUID, providerID uuid.UUID) {
	t.Helper()

	require.NoError(t, db.Create(&automation.Automation{
		ID:         automationID,
		Name:       "patch-linux",
		Url:        "https://ansible.example.com/jobs/1",
		Provider:   "ansible",
		ProviderID: providerID,
	}).Error)
}

func seedJob(t *testing.T, db *gorm.DB, jobID string, automationID uuid.UUID, providerID uuid.UUID) {
	t.Helper()

	require.NoError(t, db.Create(&job.Job{
		ID:           jobID,
		AutomationID: automationID,
		Automation:   "patch-linux",
		ProviderID:   providerID,
		Provider:     "ansible",
		Status:       "success",
		Location:     "dc1",
		Url:          "https://ansible.example.com/jobs/1",
		Metadata:     datatypes.JSON([]byte(`{}`)),
	}).Error)
}

func seedApplication(t *testing.T, db *gorm.DB, appID uuid.UUID) {
	t.Helper()

	require.NoError(t, db.Create(&application.Application{
		ID:            appID,
		Name:          "demo-app",
		Description:   "demo",
		RepositoryURL: "https://example.com/demo",
	}).Error)
}

func seedRelease(t *testing.T, db *gorm.DB, releaseID uuid.UUID, appID uuid.UUID) {
	t.Helper()

	require.NoError(t, db.Create(&release.Release{
		ID:            releaseID,
		ApplicationID: appID,
		Application:   "demo-app",
		Version:       "1.0.0",
		CommitSHA:     "abc123",
		Branch:        "main",
	}).Error)
}

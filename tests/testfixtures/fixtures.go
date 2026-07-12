package testfixtures

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/database/model"
	"github.com/svetlyopet/heimdallr/modules/application"
	"github.com/svetlyopet/heimdallr/modules/automation"
	"github.com/svetlyopet/heimdallr/modules/provider"
	"github.com/svetlyopet/heimdallr/modules/release"
	"gorm.io/gorm"
)

type ApplicationFixture struct {
	Application application.Application
}

func SeedApplication(t *testing.T, db *gorm.DB, name string) ApplicationFixture {
	t.Helper()

	app := application.Application{
		ID:            uuid.New(),
		Name:          name,
		Description:   "test application",
		RepositoryURL: "https://example.com/" + name,
		Timestamp:     model.Timestamp{CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
	}
	require.NoError(t, db.Create(&app).Error)

	return ApplicationFixture{Application: app}
}

type ReleaseFixture struct {
	Release release.Release
}

func SeedRelease(t *testing.T, db *gorm.DB, app ApplicationFixture, version string) ReleaseFixture {
	t.Helper()

	rel := release.Release{
		ID:            uuid.New(),
		ApplicationID: app.Application.ID,
		Application:   app.Application.Name,
		Version:       version,
		CommitSHA:     "abc123",
		PipelineURL:   "https://example.com/pipeline/1",
		Branch:        "main",
		Timestamp:     model.Timestamp{CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
	}
	require.NoError(t, db.Create(&rel).Error)

	return ReleaseFixture{Release: rel}
}

type ProviderAutomationFixture struct {
	Provider   provider.Provider
	Automation automation.Automation
}

func SeedProviderAutomation(t *testing.T, db *gorm.DB, providerName, automationName string) ProviderAutomationFixture {
	t.Helper()

	prov := provider.Provider{
		ID:        uuid.New(),
		Name:      providerName,
		Url:       "https://awx.example.com",
		Timestamp: model.Timestamp{CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
	}
	require.NoError(t, db.Create(&prov).Error)

	auto := automation.Automation{
		ID:          uuid.New(),
		Name:        automationName,
		Url:         "https://awx.example.com/#/templates/job_template/1",
		Provider:    prov.Name,
		ProviderID:  prov.ID,
		CostSavings: 0,
		Timestamp:   model.Timestamp{CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
	}
	require.NoError(t, db.Create(&auto).Error)

	return ProviderAutomationFixture{Provider: prov, Automation: auto}
}

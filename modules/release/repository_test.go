package release_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	release2 "github.com/svetlyopet/heimdallr/modules/release"
	"github.com/svetlyopet/heimdallr/tests/testfixtures"
	"gorm.io/gorm"
)

func newReleaseTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testutil.NewPostgresDB(t)
}

func TestRepositoryUpsertCreatesAndUpdatesRelease(t *testing.T) {
	db := newReleaseTestDB(t)
	repo := release2.NewRepository(db)

	app := testfixtures.SeedApplication(t, db, "demo-app")
	releaseEntity := release2.Release{
		ID:            uuid.New(),
		ApplicationID: app.Application.ID,
		Application:   app.Application.Name,
		Version:       "v1.0.0",
		CommitSHA:     "abc123",
		PipelineURL:   "https://example.com/pipeline/1",
		Branch:        "main",
	}

	created, err := repo.Upsert(context.Background(), releaseEntity)
	require.NoError(t, err)
	require.Equal(t, "v1.0.0", created.Version)

	releaseEntity.CommitSHA = "def456"
	updated, err := repo.Upsert(context.Background(), releaseEntity)
	require.NoError(t, err)
	require.Equal(t, "def456", updated.CommitSHA)
}

func TestRepositoryFindByApplicationAndVersionReturnsRelease(t *testing.T) {
	db := newReleaseTestDB(t)
	repo := release2.NewRepository(db)

	app := testfixtures.SeedApplication(t, db, "find-app")
	fixture := testfixtures.SeedRelease(t, db, app, "v2.0.0")

	found, err := repo.FindByApplicationAndVersion(context.Background(), app.Application.ID, "v2.0.0")
	require.NoError(t, err)
	require.Equal(t, fixture.Release.ID, found.ID)
}

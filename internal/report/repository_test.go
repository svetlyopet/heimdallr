package report

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/application"
	"github.com/svetlyopet/heimdallr/internal/database/model"
	"github.com/svetlyopet/heimdallr/internal/release"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	"gorm.io/gorm"
)

func newReportTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	return testutil.NewPostgresDB(t)
}

func TestRepositoryFindAllGlobalFiltersByStatusAndType(t *testing.T) {
	db := newReportTestDB(t)
	repo := NewRepository(db)

	appID := uuid.MustParse("5d8dd803-fca6-4f7c-9dd2-24417622d630")
	releaseID := uuid.MustParse("8b1e2f4a-9c3d-4e5f-a6b7-c8d9e0f1a2b3")

	require.NoError(t, db.Create(&application.Application{
		ID:   appID,
		Name: "demo-app",
	}).Error)

	require.NoError(t, db.Create(&release.Release{
		ID:            releaseID,
		ApplicationID: appID,
		Application:   "demo-app",
		Version:       "v1.0.0",
	}).Error)

	now := time.Now().UTC()
	reports := []Report{
		{
			ID:            "sast-1",
			ReleaseID:     releaseID,
			ApplicationID: appID,
			Application:   "demo-app",
			Version:       "v1.0.0",
			Type:          "sast",
			Status:        "failed",
			Metadata:      []byte(`{}`),
			Timestamp:     model.Timestamp{CreatedAt: now, UpdatedAt: now},
		},
		{
			ID:            "sbom-1",
			ReleaseID:     releaseID,
			ApplicationID: appID,
			Application:   "demo-app",
			Version:       "v1.0.0",
			Type:          "sbom",
			Status:        "success",
			Metadata:      []byte(`{}`),
			Timestamp:     model.Timestamp{CreatedAt: now, UpdatedAt: now},
		},
	}

	for _, report := range reports {
		require.NoError(t, db.Create(&report).Error)
	}

	found, total, err := repo.FindAllGlobal(context.Background(), ListFilters{
		Status: "failed",
		Type:   "sast",
	}, 10, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, found, 1)
	require.Equal(t, "sast-1", found[0].ID)
}

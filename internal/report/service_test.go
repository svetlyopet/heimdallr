package report

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/application"
	"github.com/svetlyopet/heimdallr/internal/release"
	"github.com/svetlyopet/heimdallr/internal/testutil"
)

func newReportService(t *testing.T) (Service, application.Service, release.Service) {
	t.Helper()

	db := testutil.NewSQLiteDB(t, &application.Application{}, &release.Release{}, &Report{})
	appRepo := application.NewRepository(db)
	appSvc := application.NewService(appRepo, nil)
	releaseRepo := release.NewRepository(db)
	releaseSvc := release.NewService(releaseRepo, appSvc, nil)
	reportRepo := NewRepository(db)
	reportSvc := NewService(reportRepo, releaseSvc, nil)

	return reportSvc, appSvc, releaseSvc
}

func TestServiceCreateAndUpdateReportLifecycle(t *testing.T) {
	reportSvc, appSvc, releaseSvc := newReportService(t)

	app, err := appSvc.Create(context.Background(), application.CreateRequest{Name: "report-app"})
	require.NoError(t, err)

	rel, err := releaseSvc.Create(context.Background(), app.ID.String(), release.CreateRequest{
		Version:     "v1.0.0",
		CommitSHA:   "abc",
		PipelineURL: "https://example.com",
		Branch:      "main",
	}, true)
	require.NoError(t, err)

	created, err := reportSvc.Create(context.Background(), app.ID.String(), rel.ID.String(), CreateRequest{
		ID:       "sast-1",
		Type:     "sast",
		Status:   "started",
		Location: "ci",
		URL:      "https://example.com/run/1",
		Metadata: json.RawMessage(`{"tool":"example"}`),
	})
	require.NoError(t, err)
	require.Equal(t, "started", created.Status)

	updated, err := reportSvc.Update(context.Background(), app.ID.String(), rel.ID.String(), "sast-1", UpdateRequest{
		Status:   "success",
		Metadata: json.RawMessage(`{"findings":0}`),
		Output:   "dGVzdA==",
	})
	require.NoError(t, err)
	require.Equal(t, "success", updated.Status)
	require.Equal(t, "dGVzdA==", updated.Output)
}

func TestServiceGetByIdReturnsNotFound(t *testing.T) {
	reportSvc, appSvc, releaseSvc := newReportService(t)

	app, err := appSvc.Create(context.Background(), application.CreateRequest{Name: "missing-report-app"})
	require.NoError(t, err)

	rel, err := releaseSvc.Create(context.Background(), app.ID.String(), release.CreateRequest{
		Version: "v1.0.0", CommitSHA: "abc", PipelineURL: "https://example.com", Branch: "main",
	}, true)
	require.NoError(t, err)

	_, err = reportSvc.GetById(context.Background(), app.ID.String(), rel.ID.String(), "missing")
	require.ErrorIs(t, err, ErrReportNotFound)
}

func TestServiceCreateReturnsNotFoundForMissingRelease(t *testing.T) {
	reportSvc, _, _ := newReportService(t)

	_, err := reportSvc.Create(context.Background(), uuid.New().String(), uuid.New().String(), CreateRequest{
		ID: "sast-1", Type: "sast", Status: "started",
	})
	require.ErrorIs(t, err, ErrReportNotFound)
}

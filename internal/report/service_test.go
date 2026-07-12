package report

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/application"
	appapi "github.com/svetlyopet/heimdallr/internal/application/api"
	"github.com/svetlyopet/heimdallr/internal/release"
	releaseapi "github.com/svetlyopet/heimdallr/internal/release/api"
	"github.com/svetlyopet/heimdallr/internal/report/api"
	"github.com/svetlyopet/heimdallr/internal/requestlimits"
	"github.com/svetlyopet/heimdallr/internal/testutil"
)

func newReportService(t *testing.T) (Service, application.Service, release.Service) {
	t.Helper()

	db := testutil.NewPostgresDB(t)
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

	app, err := appSvc.Create(context.Background(), appapi.ApplicationCreateRequest{Name: "report-app"})
	require.NoError(t, err)

	pipelineURL := releaseapi.URL("https://example.com")
	rel, err := releaseSvc.Create(context.Background(), app.Id.String(), releaseapi.ReleaseCreateRequest{
		Version:     "v1.0.0",
		CommitSha:   ptr("abc"),
		PipelineUrl: &pipelineURL,
		Branch:      ptr("main"),
	}, true)
	require.NoError(t, err)

	metadata := api.ReportMetadata{"tool": "example"}
	url := api.URL("https://example.com/run/1")
	location := "ci"
	created, err := reportSvc.Create(context.Background(), app.Id.String(), rel.Id.String(), api.ReportCreateRequest{
		Id:       "sast-1",
		Type:     api.ReportTypeSast,
		Status:   api.JobStatusStarted,
		Location: &location,
		Url:      &url,
		Metadata: &metadata,
	})
	require.NoError(t, err)
	require.Equal(t, api.JobStatusStarted, created.Status)

	updateMetadata := api.ReportMetadata{"findings": 0}
	output := "dGVzdA=="
	updated, err := reportSvc.Update(context.Background(), app.Id.String(), rel.Id.String(), "sast-1", api.ReportUpdateRequest{
		Status:   api.JobStatusSuccess,
		Metadata: &updateMetadata,
		Output:   &output,
	})
	require.NoError(t, err)
	require.Equal(t, api.JobStatusSuccess, updated.Status)
	require.NotNil(t, updated.Output)
	require.Equal(t, "dGVzdA==", *updated.Output)
}

func TestServiceGetByIdReturnsNotFound(t *testing.T) {
	reportSvc, appSvc, releaseSvc := newReportService(t)

	app, err := appSvc.Create(context.Background(), appapi.ApplicationCreateRequest{Name: "missing-report-app"})
	require.NoError(t, err)

	pipelineURL := releaseapi.URL("https://example.com")
	rel, err := releaseSvc.Create(context.Background(), app.Id.String(), releaseapi.ReleaseCreateRequest{
		Version: "v1.0.0", CommitSha: ptr("abc"), PipelineUrl: &pipelineURL, Branch: ptr("main"),
	}, true)
	require.NoError(t, err)

	_, err = reportSvc.GetById(context.Background(), app.Id.String(), rel.Id.String(), "missing")
	require.ErrorIs(t, err, ErrReportNotFound)
}

func TestServiceCreateReturnsNotFoundForMissingRelease(t *testing.T) {
	reportSvc, _, _ := newReportService(t)

	_, err := reportSvc.Create(context.Background(), uuid.New().String(), uuid.New().String(), api.ReportCreateRequest{
		Id: "sast-1", Type: api.ReportTypeSast, Status: api.JobStatusStarted,
	})
	require.ErrorIs(t, err, ErrReportNotFound)
}

func TestServiceCreateRejectsInvalidOutputBeforeReleaseLookup(t *testing.T) {
	testCases := []struct {
		name       string
		output     string
		maxDecoded int64
	}{
		{name: "invalid encoding", output: "not-base64!", maxDecoded: 1024},
		{name: "decoded output too large", output: "dGVzdA==", maxDecoded: 3},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			lookupCalls := 0
			reportSvc := NewService(stubReportRepository{}, stubReleaseLookup{getByIDCalls: &lookupCalls}, nil)
			ctx := requestlimits.WithContext(context.Background(), requestlimits.Values{
				MaxDecodedOutputBytes: testCase.maxDecoded,
			})

			_, err := reportSvc.Create(ctx, uuid.NewString(), uuid.NewString(), api.ReportCreateRequest{
				Id: "sast-1", Type: api.ReportTypeSast, Status: api.JobStatusStarted, Output: &testCase.output,
			})

			require.Error(t, err)
			require.Zero(t, lookupCalls)
		})
	}
}

type stubReleaseLookup struct {
	getByIDCalls *int
}

func (s stubReleaseLookup) GetById(context.Context, string, string) (releaseapi.ReleaseWithCompliance, error) {
	if s.getByIDCalls != nil {
		(*s.getByIDCalls)++
	}

	return releaseapi.ReleaseWithCompliance{}, nil
}

type stubReportRepository struct{}

func (stubReportRepository) FindAll(context.Context, string, string, int, int) ([]Report, int64, error) {
	return nil, 0, nil
}

func (stubReportRepository) FindAllGlobal(context.Context, ListFilters, int, int) ([]Report, int64, error) {
	return nil, 0, nil
}

func (stubReportRepository) FindById(context.Context, string, string, string) (Report, error) {
	return Report{}, nil
}

func (stubReportRepository) Create(context.Context, Report) (Report, error) {
	return Report{}, nil
}

func (stubReportRepository) Update(context.Context, Report) (Report, error) {
	return Report{}, nil
}

func ptr(s string) *string {
	return &s
}

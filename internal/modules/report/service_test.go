package report

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/modules/application"
	appapi "github.com/svetlyopet/heimdallr/internal/modules/application/api"
	"github.com/svetlyopet/heimdallr/internal/modules/release"
	releaseapi "github.com/svetlyopet/heimdallr/internal/modules/release/api"
	"github.com/svetlyopet/heimdallr/internal/modules/report/api"
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

func TestServiceCreateReport(t *testing.T) {
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

	metadata := api.ReportMetadata{"tool": "example", "findings": 0}
	url := api.URL("https://example.com/run/1")
	location := "ci"
	output := "dGVzdA=="
	created, err := reportSvc.Create(context.Background(), app.Id.String(), rel.Id.String(), api.ReportCreateRequest{
		Id:       "sast-1",
		Type:     api.ReportTypeSast,
		Status:   api.Success,
		Location: &location,
		Url:      &url,
		Metadata: &metadata,
		Output:   &output,
	})
	require.NoError(t, err)
	require.Equal(t, api.Success, created.Status)
	require.NotNil(t, created.Output)
	require.Equal(t, "dGVzdA==", *created.Output)
}

func TestServiceCreateRejectsSuccessWithoutOutput(t *testing.T) {
	reportSvc, appSvc, releaseSvc := newReportService(t)

	app, err := appSvc.Create(context.Background(), appapi.ApplicationCreateRequest{Name: "no-output-app"})
	require.NoError(t, err)

	pipelineURL := releaseapi.URL("https://example.com")
	rel, err := releaseSvc.Create(context.Background(), app.Id.String(), releaseapi.ReleaseCreateRequest{
		Version: "v1.0.0", CommitSha: ptr("abc"), PipelineUrl: &pipelineURL, Branch: ptr("main"),
	}, true)
	require.NoError(t, err)

	_, err = reportSvc.Create(context.Background(), app.Id.String(), rel.Id.String(), api.ReportCreateRequest{
		Id:     "sast-1",
		Type:   api.ReportTypeSast,
		Status: api.Success,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidOutput)
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

	output := "dGVzdA=="
	_, err := reportSvc.Create(context.Background(), uuid.New().String(), uuid.New().String(), api.ReportCreateRequest{
		Id: "sast-1", Type: api.ReportTypeSast, Status: api.Success, Output: &output,
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
				Id: "sast-1", Type: api.ReportTypeSast, Status: api.Success, Output: &testCase.output,
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

func ptr(s string) *string {
	return &s
}

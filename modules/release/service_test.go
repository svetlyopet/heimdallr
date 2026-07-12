package release

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	application2 "github.com/svetlyopet/heimdallr/modules/application"
	appapi "github.com/svetlyopet/heimdallr/modules/application/api"
	"github.com/svetlyopet/heimdallr/modules/release/api"
	"gorm.io/gorm"
)

func newReleaseService(t *testing.T) (Service, application2.Service, *gorm.DB) {
	t.Helper()

	db := testutil.NewPostgresDB(t)
	appRepo := application2.NewRepository(db)
	appSvc := application2.NewService(appRepo, nil)
	releaseRepo := NewRepository(db)
	releaseSvc := NewService(releaseRepo, appSvc, nil)

	return releaseSvc, appSvc, db
}

func TestServiceCreateUpsertsRelease(t *testing.T) {
	releaseSvc, appSvc, _ := newReleaseService(t)

	repoURL := appapi.URL("https://example.com/upsert")
	app, err := appSvc.Create(context.Background(), appapi.ApplicationCreateRequest{
		Name:          "upsert-app",
		Description:   ptr("desc"),
		RepositoryUrl: &repoURL,
	})
	require.NoError(t, err)

	pipelineURL := api.URL("https://example.com/pipeline/1")
	req := api.ReleaseCreateRequest{
		Version:     "v1.0.0",
		CommitSha:   ptr("abc123"),
		PipelineUrl: &pipelineURL,
		Branch:      ptr("main"),
	}

	created, err := releaseSvc.Create(context.Background(), app.Id.String(), req, true)
	require.NoError(t, err)
	require.Equal(t, "v1.0.0", created.Version)

	commitSHA := "def456"
	req.CommitSha = &commitSHA
	updated, err := releaseSvc.Create(context.Background(), app.Id.String(), req, true)
	require.NoError(t, err)
	require.Equal(t, "def456", updated.CommitSha)
	require.Equal(t, created.Id, updated.Id)
}

func TestServiceCreateReturnsConflictWithoutUpsert(t *testing.T) {
	releaseSvc, appSvc, _ := newReleaseService(t)

	app, err := appSvc.Create(context.Background(), appapi.ApplicationCreateRequest{Name: "conflict-app"})
	require.NoError(t, err)

	pipelineURL := api.URL("https://example.com")
	req := api.ReleaseCreateRequest{
		Version:     "v1.0.0",
		CommitSha:   ptr("abc"),
		PipelineUrl: &pipelineURL,
		Branch:      ptr("main"),
	}
	_, err = releaseSvc.Create(context.Background(), app.Id.String(), req, false)
	require.NoError(t, err)

	_, err = releaseSvc.Create(context.Background(), app.Id.String(), req, false)
	require.ErrorIs(t, err, ErrReleaseAlreadyExists)
}

func TestServiceCreateReturnsApplicationNotFound(t *testing.T) {
	releaseSvc, _, _ := newReleaseService(t)

	_, err := releaseSvc.Create(context.Background(), uuid.New().String(), api.ReleaseCreateRequest{
		Version: "v1.0.0",
	}, false)
	require.ErrorIs(t, err, application2.ErrApplicationNotFound)
}

type stubApplicationLookup struct {
	getByIdResp  appapi.Application
	getByIdError error
}

func (s stubApplicationLookup) GetById(_ context.Context, _ string) (appapi.Application, error) {
	if s.getByIdError != nil {
		return appapi.Application{}, s.getByIdError
	}

	return s.getByIdResp, nil
}

func (s stubApplicationLookup) GetByName(_ context.Context, _ string) (appapi.Application, error) {
	return appapi.Application{}, nil
}

type stubReleaseRepository struct {
	findAllErr error
}

func (s stubReleaseRepository) FindAll(context.Context, string, int, int) ([]Release, int64, error) {
	return nil, 0, s.findAllErr
}

func (s stubReleaseRepository) FindById(context.Context, string, string) (Release, error) {
	return Release{}, nil
}

func (s stubReleaseRepository) FindByApplicationAndVersion(context.Context, uuid.UUID, string) (Release, error) {
	return Release{}, nil
}

func (s stubReleaseRepository) Create(context.Context, Release) (Release, error) {
	return Release{}, nil
}

func (s stubReleaseRepository) Upsert(context.Context, Release) (Release, error) {
	return Release{}, nil
}

func (s stubReleaseRepository) GetComplianceSummary(context.Context, uuid.UUID) (api.ComplianceSummary, error) {
	return api.ComplianceSummary{}, nil
}

func (s stubReleaseRepository) GetComplianceSummariesForReleases(context.Context, []uuid.UUID) (map[uuid.UUID]api.ComplianceSummary, error) {
	return map[uuid.UUID]api.ComplianceSummary{}, nil
}

func TestServiceGetAllReturnsApplicationNotFound(t *testing.T) {
	svc := NewService(stubReleaseRepository{}, stubApplicationLookup{
		getByIdError: application2.ErrApplicationNotFound,
	}, nil)

	_, _, err := svc.GetAll(context.Background(), uuid.New().String(), 1, 10)
	require.ErrorIs(t, err, application2.ErrApplicationNotFound)
}

func TestServiceGetAllReturnsListError(t *testing.T) {
	svc := NewService(stubReleaseRepository{findAllErr: errors.New("boom")}, stubApplicationLookup{
		getByIdResp: appapi.Application{Id: uuid.New()},
	}, nil)

	_, _, err := svc.GetAll(context.Background(), uuid.New().String(), 1, 10)
	require.ErrorIs(t, err, ErrListReleases)
}

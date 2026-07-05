package release

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/application"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	"gorm.io/gorm"
)

func newReleaseService(t *testing.T) (Service, application.Service, *gorm.DB) {
	t.Helper()

	db := testutil.NewSQLiteDB(t, &application.Application{}, &Release{})
	appRepo := application.NewRepository(db)
	appSvc := application.NewService(appRepo, nil)
	releaseRepo := NewRepository(db)
	releaseSvc := NewService(releaseRepo, appSvc, nil)

	return releaseSvc, appSvc, db
}

func TestServiceCreateUpsertsRelease(t *testing.T) {
	releaseSvc, appSvc, _ := newReleaseService(t)

	app, err := appSvc.Create(context.Background(), application.CreateRequest{
		Name:          "upsert-app",
		Description:   "desc",
		RepositoryURL: "https://example.com/upsert",
	})
	require.NoError(t, err)

	req := CreateRequest{
		Version:     "v1.0.0",
		CommitSHA:   "abc123",
		PipelineURL: "https://example.com/pipeline/1",
		Branch:      "main",
	}

	created, err := releaseSvc.Create(context.Background(), app.ID.String(), req, true)
	require.NoError(t, err)
	require.Equal(t, "v1.0.0", created.Version)

	req.CommitSHA = "def456"
	updated, err := releaseSvc.Create(context.Background(), app.ID.String(), req, true)
	require.NoError(t, err)
	require.Equal(t, "def456", updated.CommitSHA)
	require.Equal(t, created.ID, updated.ID)
}

func TestServiceCreateReturnsConflictWithoutUpsert(t *testing.T) {
	releaseSvc, appSvc, _ := newReleaseService(t)

	app, err := appSvc.Create(context.Background(), application.CreateRequest{Name: "conflict-app"})
	require.NoError(t, err)

	req := CreateRequest{Version: "v1.0.0", CommitSHA: "abc", PipelineURL: "https://example.com", Branch: "main"}
	_, err = releaseSvc.Create(context.Background(), app.ID.String(), req, false)
	require.NoError(t, err)

	_, err = releaseSvc.Create(context.Background(), app.ID.String(), req, false)
	require.ErrorIs(t, err, ErrReleaseAlreadyExists)
}

func TestServiceCreateReturnsApplicationNotFound(t *testing.T) {
	releaseSvc, _, _ := newReleaseService(t)

	_, err := releaseSvc.Create(context.Background(), uuid.New().String(), CreateRequest{
		Version: "v1.0.0",
	}, false)
	require.ErrorIs(t, err, application.ErrApplicationNotFound)
}

type stubApplicationLookup struct {
	getByIdResp  application.GetResponse
	getByIdError error
}

func (s stubApplicationLookup) GetById(_ context.Context, _ string) (application.GetResponse, error) {
	if s.getByIdError != nil {
		return application.GetResponse{}, s.getByIdError
	}

	return s.getByIdResp, nil
}

func (s stubApplicationLookup) GetByName(_ context.Context, _ string) (application.GetResponse, error) {
	return application.GetResponse{}, nil
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

func (s stubReleaseRepository) GetComplianceSummary(context.Context, uuid.UUID) (ComplianceSummary, error) {
	return ComplianceSummary{}, nil
}

func (s stubReleaseRepository) GetComplianceSummariesForReleases(context.Context, []uuid.UUID) (map[uuid.UUID]ComplianceSummary, error) {
	return map[uuid.UUID]ComplianceSummary{}, nil
}

func TestServiceGetAllReturnsApplicationNotFound(t *testing.T) {
	svc := NewService(stubReleaseRepository{}, stubApplicationLookup{
		getByIdError: application.ErrApplicationNotFound,
	}, nil)

	_, _, err := svc.GetAll(context.Background(), uuid.New().String(), 1, 10)
	require.ErrorIs(t, err, application.ErrApplicationNotFound)
}

func TestServiceGetAllReturnsListError(t *testing.T) {
	svc := NewService(stubReleaseRepository{findAllErr: errors.New("boom")}, stubApplicationLookup{
		getByIdResp: application.GetResponse{ID: uuid.New()},
	}, nil)

	_, _, err := svc.GetAll(context.Background(), uuid.New().String(), 1, 10)
	require.ErrorIs(t, err, ErrListReleases)
}

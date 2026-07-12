package release

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/application"
	"github.com/svetlyopet/heimdallr/internal/rbac"
	"github.com/svetlyopet/heimdallr/internal/release/api"
	"github.com/svetlyopet/heimdallr/internal/testutil"
)

type stubReleaseService struct {
	listResponse []api.ReleaseListItem
	listTotal    int64
	listError    error
	getResponse  api.ReleaseWithCompliance
	getError     error
	createResp   api.Release
	createError  error
}

func (s stubReleaseService) GetAll(_ context.Context, _ string, _ int, _ int) ([]api.ReleaseListItem, int64, error) {
	if s.listError != nil {
		return nil, 0, s.listError
	}

	return s.listResponse, s.listTotal, nil
}

func (s stubReleaseService) GetById(_ context.Context, _, _ string) (api.ReleaseWithCompliance, error) {
	if s.getError != nil {
		return api.ReleaseWithCompliance{}, s.getError
	}

	return s.getResponse, nil
}

func (s stubReleaseService) Create(_ context.Context, _ string, _ api.ReleaseCreateRequest, _ bool) (api.Release, error) {
	if s.createError != nil {
		return api.Release{}, s.createError
	}

	return s.createResp, nil
}

func newReleaseRouter(t *testing.T, svc Service) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)

	h, err := NewHandler(svc)
	require.NoError(t, err)

	r := gin.New()
	apiGroup := r.Group("/api")
	apiGroup.Use(testutil.AuthenticatedAdminMiddleware())
	RegisterRoutes(apiGroup, h, rbac.NewAuthorizer())

	return r
}

func TestHandlerListReturnsBadRequestForInvalidApplicationID(t *testing.T) {
	r := newReleaseRouter(t, stubReleaseService{})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, "/api/v1/application/not-a-uuid/release", nil, nil)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandlerCreateReturnsNotFoundForMissingApplication(t *testing.T) {
	appID := uuid.New()
	pipelineURL := api.URL("https://example.com")
	r := newReleaseRouter(t, stubReleaseService{createError: application.ErrApplicationNotFound})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, "/api/v1/application/"+appID.String()+"/release", api.ReleaseCreateRequest{
		Version:     "v1.0.0",
		CommitSha:   ptr("abc"),
		PipelineUrl: &pipelineURL,
		Branch:      ptr("main"),
	}, nil)
	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandlerCreateReturnsRelease(t *testing.T) {
	appID := uuid.New()
	releaseID := uuid.New()
	pipelineURL := api.URL("https://example.com")
	r := newReleaseRouter(t, stubReleaseService{
		createResp: api.Release{Id: releaseID, ApplicationId: appID, Version: "v1.0.0"},
	})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, "/api/v1/application/"+appID.String()+"/release?upsert=true", api.ReleaseCreateRequest{
		Version:     "v1.0.0",
		CommitSha:   ptr("abc"),
		PipelineUrl: &pipelineURL,
		Branch:      ptr("main"),
	}, nil)
	response := testutil.AssertJSONStatus(t, rr, http.StatusCreated)
	data, ok := response["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "v1.0.0", data["version"])
}

func TestHandlerGetReturnsNotFound(t *testing.T) {
	appID := uuid.New()
	r := newReleaseRouter(t, stubReleaseService{getError: ErrReleaseNotFound})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, "/api/v1/application/"+appID.String()+"/release/"+uuid.New().String(), nil, nil)
	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandlerListReturnsReleases(t *testing.T) {
	appID := uuid.New()
	releaseID := uuid.New()
	r := newReleaseRouter(t, stubReleaseService{
		listResponse: []api.ReleaseListItem{{
			Id:            releaseID,
			ApplicationId: appID,
			Version:       "v1.0.0",
			CreatedAt:     time.Now().UTC(),
		}},
		listTotal: 1,
	})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, "/api/v1/application/"+appID.String()+"/release", nil, nil)
	testutil.AssertJSONStatus(t, rr, http.StatusOK)
}

func TestHandlerListReturnsInternalServerError(t *testing.T) {
	appID := uuid.New()
	r := newReleaseRouter(t, stubReleaseService{listError: errors.New("boom")})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, "/api/v1/application/"+appID.String()+"/release", nil, nil)
	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

func ptr(s string) *string {
	return &s
}

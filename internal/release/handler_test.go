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
	"github.com/svetlyopet/heimdallr/internal/testutil"
)

type stubReleaseService struct {
	listResponse []ListItemResponse
	listTotal    int64
	listError    error
	getResponse  GetWithSummaryResponse
	getError     error
	createResp   GetResponse
	createError  error
}

func (s stubReleaseService) GetAll(_ context.Context, _ string, _ int, _ int) ([]ListItemResponse, int64, error) {
	if s.listError != nil {
		return nil, 0, s.listError
	}

	return s.listResponse, s.listTotal, nil
}

func (s stubReleaseService) GetById(_ context.Context, _, _ string) (GetWithSummaryResponse, error) {
	if s.getError != nil {
		return GetWithSummaryResponse{}, s.getError
	}

	return s.getResponse, nil
}

func (s stubReleaseService) Create(_ context.Context, _ string, _ CreateRequest, _ bool) (GetResponse, error) {
	if s.createError != nil {
		return GetResponse{}, s.createError
	}

	return s.createResp, nil
}

func newReleaseRouter(t *testing.T, svc Service) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)

	h, err := NewHandler(svc)
	require.NoError(t, err)

	r := gin.New()
	api := r.Group("/api")
	RegisterRoutes(api, h)

	return r
}

func TestHandlerListReturnsBadRequestForInvalidApplicationID(t *testing.T) {
	r := newReleaseRouter(t, stubReleaseService{})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, "/api/v1/application/not-a-uuid/release", nil, nil)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandlerCreateReturnsNotFoundForMissingApplication(t *testing.T) {
	appID := uuid.New()
	r := newReleaseRouter(t, stubReleaseService{createError: application.ErrApplicationNotFound})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, "/api/v1/application/"+appID.String()+"/release", CreateRequest{
		Version:     "v1.0.0",
		CommitSHA:   "abc",
		PipelineURL: "https://example.com",
		Branch:      "main",
	}, nil)
	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandlerCreateReturnsRelease(t *testing.T) {
	appID := uuid.New()
	releaseID := uuid.New()
	r := newReleaseRouter(t, stubReleaseService{
		createResp: GetResponse{ID: releaseID, ApplicationID: appID, Version: "v1.0.0"},
	})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, "/api/v1/application/"+appID.String()+"/release?upsert=true", CreateRequest{
		Version:     "v1.0.0",
		CommitSHA:   "abc",
		PipelineURL: "https://example.com",
		Branch:      "main",
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
		listResponse: []ListItemResponse{{
			GetResponse: GetResponse{ID: releaseID, ApplicationID: appID, Version: "v1.0.0", CreatedAt: time.Now().UTC()},
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

package report

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type stubService struct {
	getAllGlobalResponse []GetResponse
	getAllGlobalTotal    int64
	getAllGlobalError    error
}

func (s stubService) GetAll(_ context.Context, _ string, _ string, _ int, _ int) ([]GetResponse, int64, error) {
	return nil, 0, nil
}

func (s stubService) GetAllGlobal(_ context.Context, _ ListFilters, _ int, _ int) ([]GetResponse, int64, error) {
	if s.getAllGlobalError != nil {
		return nil, 0, s.getAllGlobalError
	}

	return s.getAllGlobalResponse, s.getAllGlobalTotal, nil
}

func (s stubService) GetById(_ context.Context, _ string, _ string, _ string) (GetResponse, error) {
	return GetResponse{}, nil
}

func (s stubService) Create(_ context.Context, _ string, _ string, _ CreateRequest) (GetResponse, error) {
	return GetResponse{}, nil
}

func (s stubService) Update(_ context.Context, _ string, _ string, _ string, _ UpdateRequest) (GetResponse, error) {
	return GetResponse{}, nil
}

func newReportRouter(t *testing.T, svc Service) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)

	h, err := NewHandler(svc)
	require.NoError(t, err)

	r := gin.New()
	api := r.Group("/api")
	RegisterRoutes(api, h)

	return r
}

func TestListAllReturnsBadRequestForInvalidApplicationID(t *testing.T) {
	r := newReportRouter(t, stubService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/report?application_id=not-a-uuid", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestListAllReturnsBadRequestForInvalidStatus(t *testing.T) {
	r := newReportRouter(t, stubService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/report?status=unknown", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestListAllReturnsReports(t *testing.T) {
	applicationID := uuid.MustParse("5d8dd803-fca6-4f7c-9dd2-24417622d630")
	releaseID := uuid.MustParse("8b1e2f4a-9c3d-4e5f-a6b7-c8d9e0f1a2b3")

	r := newReportRouter(t, stubService{
		getAllGlobalResponse: []GetResponse{
			{
				ID:            "sast-1",
				ApplicationID: applicationID,
				ReleaseID:     releaseID,
				Application:   "demo-app",
				Version:       "v1.0.0",
				Type:          "sast",
				Status:        "failed",
			},
		},
		getAllGlobalTotal: 1,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/report?status=failed", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestListAllReturnsInternalServerError(t *testing.T) {
	r := newReportRouter(t, stubService{getAllGlobalError: errors.New("boom")})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/report", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

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
	"github.com/svetlyopet/heimdallr/internal/modules/report/api"
	"github.com/svetlyopet/heimdallr/internal/rbac"
	"github.com/svetlyopet/heimdallr/internal/testutil"
)

type stubService struct {
	getAllGlobalResponse []api.Report
	getAllGlobalTotal    int64
	getAllGlobalError    error
	createResponse       api.Report
	createError          error
}

func (s stubService) GetAll(_ context.Context, _ string, _ string, _ int, _ int) ([]api.Report, int64, error) {
	return nil, 0, nil
}

func (s stubService) GetAllGlobal(_ context.Context, _ ListFilters, _ int, _ int) ([]api.Report, int64, error) {
	if s.getAllGlobalError != nil {
		return nil, 0, s.getAllGlobalError
	}

	return s.getAllGlobalResponse, s.getAllGlobalTotal, nil
}

func (s stubService) GetById(_ context.Context, _ string, _ string, _ string) (api.Report, error) {
	return api.Report{}, nil
}

func (s stubService) Create(_ context.Context, _ string, _ string, _ api.ReportCreateRequest) (api.Report, error) {
	if s.createError != nil {
		return api.Report{}, s.createError
	}

	return s.createResponse, nil
}

func newReportRouter(t *testing.T, svc Service) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)

	h, err := NewHandler(svc)
	require.NoError(t, err)

	r := gin.New()
	apiGroup := r.Group("/api")
	apiGroup.Use(testutil.AuthenticatedAdminMiddleware())
	RegisterRoutes(apiGroup, h, rbac.NewAuthorizer(), nil)

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
		getAllGlobalResponse: []api.Report{
			{
				Id:            "sast-1",
				ApplicationId: applicationID,
				ReleaseId:     releaseID,
				Application:   "demo-app",
				Version:       "v1.0.0",
				Type:          api.ReportTypeSast,
				Status:        api.Failed,
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

func TestCreateReportReturnsCreated(t *testing.T) {
	applicationID := uuid.MustParse("5d8dd803-fca6-4f7c-9dd2-24417622d630")
	releaseID := uuid.MustParse("8b1e2f4a-9c3d-4e5f-a6b7-c8d9e0f1a2b3")
	location := "ci"
	output := "dGVzdA=="

	r := newReportRouter(t, stubService{
		createResponse: api.Report{
			Id:            "sast-1",
			ApplicationId: applicationID,
			ReleaseId:     releaseID,
			Type:          api.ReportTypeSast,
			Status:        api.Success,
		},
	})

	path := "/api/v1/application/" + applicationID.String() + "/release/" + releaseID.String() + "/report"
	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, path, api.ReportCreateRequest{
		Id: "sast-1", Type: api.ReportTypeSast, Status: api.Success, Location: &location, Output: &output,
	}, nil)
	response := testutil.AssertJSONStatus(t, rr, http.StatusCreated)
	data, ok := response["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "success", data["status"])
}

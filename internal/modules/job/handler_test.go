package job

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/modules/job/api"
	"github.com/svetlyopet/heimdallr/internal/rbac"
	"github.com/svetlyopet/heimdallr/internal/testutil"
)

type stubJobService struct {
	listResponse       []api.Job
	listTotal          int64
	listError          error
	listGlobalResponse []api.Job
	listGlobalTotal    int64
	listGlobalError    error
	getResponse        api.Job
	getError           error
	createResp         api.Job
	createError        error
}

func (s stubJobService) GetAll(_ context.Context, _ string, _ int, _ int) ([]api.Job, int64, error) {
	if s.listError != nil {
		return nil, 0, s.listError
	}

	return s.listResponse, s.listTotal, nil
}

func (s stubJobService) GetAllGlobal(_ context.Context, _ ListFilters, _ int, _ int) ([]api.Job, int64, error) {
	if s.listGlobalError != nil {
		return nil, 0, s.listGlobalError
	}

	return s.listGlobalResponse, s.listGlobalTotal, nil
}

func (s stubJobService) GetById(_ context.Context, _, _ string) (api.Job, error) {
	if s.getError != nil {
		return api.Job{}, s.getError
	}

	return s.getResponse, nil
}

func (s stubJobService) Create(_ context.Context, _ string, _ api.JobCreateRequest) (api.Job, error) {
	if s.createError != nil {
		return api.Job{}, s.createError
	}

	return s.createResp, nil
}

func newJobRouter(t *testing.T, svc Service) *gin.Engine {
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

func TestHandlerCreateReturnsBadRequestForInvalidAutomationID(t *testing.T) {
	r := newJobRouter(t, stubJobService{})
	output := api.JobOutput("dGVzdA==")

	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, "/api/v1/automation/not-a-uuid/job", api.JobCreateRequest{
		Id: "1000", Status: api.Success, Location: "global", Url: "https://example.com/job/1", Output: &output,
	}, nil)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandlerCreateReturnsCreated(t *testing.T) {
	automationID := uuid.New()
	r := newJobRouter(t, stubJobService{
		createResp: api.Job{Id: "1000", Status: api.Success},
	})

	path := "/api/v1/automation/" + automationID.String() + "/job"
	metadata := api.JobMetadata{"inventory": "true"}
	output := api.JobOutput("dGVzdA==")
	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, path, api.JobCreateRequest{
		Id: "1000", Status: api.Success, Location: "global", Url: "https://example.com/job/1",
		Metadata: &metadata, Output: &output,
	}, nil)
	response := testutil.AssertJSONStatus(t, rr, http.StatusCreated)
	data, ok := response["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "success", data["status"])
}

func TestHandlerGetReturnsJob(t *testing.T) {
	automationID := uuid.New()
	output := api.JobOutput("dGVzdA==")
	r := newJobRouter(t, stubJobService{
		getResponse: api.Job{Id: "1000", Status: api.Success, Output: &output},
	})

	path := "/api/v1/automation/" + automationID.String() + "/job/1000"
	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, path, nil, nil)
	response := testutil.AssertJSONStatus(t, rr, http.StatusOK)
	data, ok := response["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "success", data["status"])
}

func TestHandlerListReturnsInternalServerError(t *testing.T) {
	automationID := uuid.New()
	r := newJobRouter(t, stubJobService{listError: errors.New("boom")})

	path := "/api/v1/automation/" + automationID.String() + "/job"
	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, path, nil, nil)
	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestHandlerListGlobalReturnsBadRequestForInvalidStatus(t *testing.T) {
	r := newJobRouter(t, stubJobService{})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, "/api/v1/job?status=unknown", nil, nil)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandlerListGlobalReturnsJobs(t *testing.T) {
	automationID := uuid.New()
	r := newJobRouter(t, stubJobService{
		listGlobalResponse: []api.Job{
			{Id: "1000", AutomationId: automationID, Status: api.Success},
		},
		listGlobalTotal: 1,
	})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, "/api/v1/job?status=success", nil, nil)
	require.Equal(t, http.StatusOK, rr.Code)
}

func TestHandlerListGlobalReturnsInternalServerError(t *testing.T) {
	r := newJobRouter(t, stubJobService{listGlobalError: errors.New("boom")})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, "/api/v1/job", nil, nil)
	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

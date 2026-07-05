package job

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/testutil"
)

type stubJobService struct {
	listResponse []GetResponse
	listTotal    int64
	listError    error
	getResponse  GetResponse
	getError     error
	createResp   GetResponse
	createError  error
	updateResp   GetResponse
	updateError  error
}

func (s stubJobService) GetAll(_ context.Context, _ string, _ int, _ int) ([]GetResponse, int64, error) {
	if s.listError != nil {
		return nil, 0, s.listError
	}

	return s.listResponse, s.listTotal, nil
}

func (s stubJobService) GetById(_ context.Context, _, _ string) (GetResponse, error) {
	if s.getError != nil {
		return GetResponse{}, s.getError
	}

	return s.getResponse, nil
}

func (s stubJobService) Create(_ context.Context, _ string, _ CreateRequest) (GetResponse, error) {
	if s.createError != nil {
		return GetResponse{}, s.createError
	}

	return s.createResp, nil
}

func (s stubJobService) Update(_ context.Context, _, _ string, _ UpdateRequest) (GetResponse, error) {
	if s.updateError != nil {
		return GetResponse{}, s.updateError
	}

	return s.updateResp, nil
}

func newJobRouter(t *testing.T, svc Service) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)

	h, err := NewHandler(svc)
	require.NoError(t, err)

	r := gin.New()
	api := r.Group("/api")
	RegisterRoutes(api, h)

	return r
}

func TestHandlerCreateReturnsBadRequestForInvalidAutomationID(t *testing.T) {
	r := newJobRouter(t, stubJobService{})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, "/api/v1/automation/not-a-uuid/job", CreateRequest{
		ID: "1000", Status: "started", Location: "global", URL: "https://example.com/job/1",
	}, nil)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandlerCreateReturnsCreated(t *testing.T) {
	automationID := uuid.New()
	r := newJobRouter(t, stubJobService{
		createResp: GetResponse{ID: "1000", Status: "started"},
	})

	path := "/api/v1/automation/" + automationID.String() + "/job"
	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, path, CreateRequest{
		ID: "1000", Status: "started", Location: "global", URL: "https://example.com/job/1",
		Metadata: json.RawMessage(`{"inventory":"true"}`),
	}, nil)
	response := testutil.AssertJSONStatus(t, rr, http.StatusCreated)
	data, ok := response["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "started", data["status"])
}

func TestHandlerUpdateReturnsNotFound(t *testing.T) {
	automationID := uuid.New()
	r := newJobRouter(t, stubJobService{updateError: ErrJobNotFound})

	path := "/api/v1/automation/" + automationID.String() + "/job/1000"
	rr := testutil.DoGinJSONRequest(t, r, http.MethodPatch, path, UpdateRequest{Status: "success"}, nil)
	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandlerUpdateReturnsBadRequestForInvalidOutput(t *testing.T) {
	automationID := uuid.New()
	r := newJobRouter(t, stubJobService{})

	path := "/api/v1/automation/" + automationID.String() + "/job/1000"
	rr := testutil.DoGinJSONRequest(t, r, http.MethodPatch, path, UpdateRequest{
		Status: "success",
		Output: "not-base64!",
	}, nil)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandlerGetReturnsJob(t *testing.T) {
	automationID := uuid.New()
	r := newJobRouter(t, stubJobService{
		getResponse: GetResponse{ID: "1000", Status: "success", Output: "dGVzdA=="},
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

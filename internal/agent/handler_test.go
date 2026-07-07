package agent

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/server"
	"github.com/svetlyopet/heimdallr/internal/testutil"
)

type stubAgentService struct {
	listResponse []GetResponse
	listTotal    int64
	listError    error
	getResponse  GetResponse
	getError     error
	createResp   GetResponse
	createError  error
	deleteError  error
}

func (s stubAgentService) GetAll(_ context.Context, _ string, _ int, _ int) ([]GetResponse, int64, error) {
	if s.listError != nil {
		return nil, 0, s.listError
	}

	return s.listResponse, s.listTotal, nil
}

func (s stubAgentService) GetById(_ context.Context, _, _ string) (GetResponse, error) {
	if s.getError != nil {
		return GetResponse{}, s.getError
	}

	return s.getResponse, nil
}

func (s stubAgentService) Create(_ context.Context, _ string, _ CreateRequest) (GetResponse, error) {
	if s.createError != nil {
		return GetResponse{}, s.createError
	}

	return s.createResp, nil
}

func (s stubAgentService) Delete(_ context.Context, _ string, _ string) error {
	return s.deleteError
}

func (s stubAgentService) ListGlobal(_ context.Context, _ bool, _ int, _ int) ([]GetResponse, int64, error) {
	return s.listResponse, s.listTotal, s.listError
}

func (s stubAgentService) GetByIdGlobal(_ context.Context, _ string) (GetResponse, error) {
	if s.getError != nil {
		return GetResponse{}, s.getError
	}

	return s.getResponse, nil
}

func (s stubAgentService) CreateUnassigned(_ context.Context, _ CreateRequest) (GetResponse, error) {
	if s.createError != nil {
		return GetResponse{}, s.createError
	}

	return s.createResp, nil
}

func uuidPtr(id uuid.UUID) *uuid.UUID {
	return &id
}

func newAgentRouter(t *testing.T, svc Service) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)

	h, err := NewHandler(svc)
	require.NoError(t, err)

	r := gin.New()
	api := r.Group("/api")
	RegisterRoutes(api, h)

	return r
}

func TestHandlerListReturnsBadRequestForInvalidPage(t *testing.T) {
	serverID := uuid.New()
	r := newAgentRouter(t, stubAgentService{})

	path := "/api/v1/server/" + serverID.String() + "/agent?page=0"
	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, path, nil, nil)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandlerListReturnsAgents(t *testing.T) {
	serverID := uuid.New()
	agentID := uuid.New()
	r := newAgentRouter(t, stubAgentService{
		listResponse: []GetResponse{{ID: agentID, Name: "datadog", Server: "web-01.example.com"}},
		listTotal:    1,
	})

	path := "/api/v1/server/" + serverID.String() + "/agent"
	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, path, nil, nil)
	response := testutil.AssertJSONStatus(t, rr, http.StatusOK)

	data, ok := response["data"].([]any)
	require.True(t, ok)
	require.Len(t, data, 1)
}

func TestHandlerCreateReturnsCreated(t *testing.T) {
	serverID := uuid.New()
	agentID := uuid.New()
	r := newAgentRouter(t, stubAgentService{
		createResp: GetResponse{ID: agentID, ServerID: uuidPtr(serverID), Server: "web-01.example.com", Name: "datadog"},
	})

	path := "/api/v1/server/" + serverID.String() + "/agent"
	body := CreateRequest{Name: "datadog"}
	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, path, body, nil)
	response := testutil.AssertJSONStatus(t, rr, http.StatusCreated)

	data, ok := response["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "datadog", data["name"])
	require.Equal(t, "web-01.example.com", data["server"])
}

func TestHandlerGetReturnsAgent(t *testing.T) {
	serverID := uuid.New()
	agentID := uuid.New()
	r := newAgentRouter(t, stubAgentService{
		getResponse: GetResponse{ID: agentID, ServerID: uuidPtr(serverID), Name: "datadog"},
	})

	path := "/api/v1/server/" + serverID.String() + "/agent/" + agentID.String()
	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, path, nil, nil)
	response := testutil.AssertJSONStatus(t, rr, http.StatusOK)

	data, ok := response["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "datadog", data["name"])
}

func TestHandlerGetReturnsNotFound(t *testing.T) {
	serverID := uuid.New()
	agentID := uuid.New()
	r := newAgentRouter(t, stubAgentService{getError: ErrAgentNotFound})

	path := "/api/v1/server/" + serverID.String() + "/agent/" + agentID.String()
	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, path, nil, nil)
	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandlerCreateReturnsServerNotFound(t *testing.T) {
	serverID := uuid.New()
	r := newAgentRouter(t, stubAgentService{createError: server.ErrServerNotFound})

	path := "/api/v1/server/" + serverID.String() + "/agent"
	body := CreateRequest{Name: "datadog"}
	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, path, body, nil)
	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandlerListReturnsServerNotFound(t *testing.T) {
	serverID := uuid.New()
	r := newAgentRouter(t, stubAgentService{listError: server.ErrServerNotFound})

	path := "/api/v1/server/" + serverID.String() + "/agent"
	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, path, nil, nil)
	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandlerCreateReturnsBadRequestForInvalidBody(t *testing.T) {
	serverID := uuid.New()
	r := newAgentRouter(t, stubAgentService{})

	path := "/api/v1/server/" + serverID.String() + "/agent"
	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, path, json.RawMessage(`{}`), nil)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandlerDeleteAgentReturnsNoContent(t *testing.T) {
	serverID := uuid.New()
	agentID := uuid.New()
	r := newAgentRouter(t, stubAgentService{})

	path := "/api/v1/server/" + serverID.String() + "/agent/" + agentID.String()
	rr := testutil.DoGinJSONRequest(t, r, http.MethodDelete, path, nil, nil)
	require.Equal(t, http.StatusNoContent, rr.Code)
}

func TestHandlerGetReturnsBadRequestForInvalidAgentID(t *testing.T) {
	serverID := uuid.New()
	r := newAgentRouter(t, stubAgentService{})

	path := "/api/v1/server/" + serverID.String() + "/agent/not-a-uuid"
	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, path, nil, nil)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandlerGetReturnsBadRequestForInvalidServerID(t *testing.T) {
	r := newAgentRouter(t, stubAgentService{})

	path := "/api/v1/server/not-a-uuid/agent/" + uuid.New().String()
	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, path, nil, nil)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandlerListReturnsInternalServerError(t *testing.T) {
	serverID := uuid.New()
	r := newAgentRouter(t, stubAgentService{listError: errors.New("db down")})

	path := "/api/v1/server/" + serverID.String() + "/agent"
	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, path, nil, nil)
	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

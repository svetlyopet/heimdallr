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
	"github.com/svetlyopet/heimdallr/internal/rbac"
	"github.com/svetlyopet/heimdallr/internal/server"
	"github.com/svetlyopet/heimdallr/internal/testutil"
	"github.com/svetlyopet/heimdallr/modules/agent/api"
)

type stubAgentService struct {
	listResponse []api.Agent
	listTotal    int64
	listError    error
	getResponse  api.Agent
	getError     error
	detailResp   api.AgentDetail
	createResp   api.Agent
	createError  error
	detachError  error
}

func (s stubAgentService) GetAll(_ context.Context, _ string, _ int, _ int) ([]api.Agent, int64, error) {
	if s.listError != nil {
		return nil, 0, s.listError
	}

	return s.listResponse, s.listTotal, nil
}

func (s stubAgentService) GetById(_ context.Context, _, _ string) (api.Agent, error) {
	if s.getError != nil {
		return api.Agent{}, s.getError
	}

	return s.getResponse, nil
}

func (s stubAgentService) CreateOnServer(_ context.Context, _ string, _ api.ServerAgentRequest) (api.Agent, error) {
	if s.createError != nil {
		return api.Agent{}, s.createError
	}

	return s.createResp, nil
}

func (s stubAgentService) Detach(_ context.Context, _ string, _ string) error {
	return s.detachError
}

func (s stubAgentService) ListGlobal(_ context.Context, _ ListFilters, _ int, _ int) ([]api.Agent, int64, error) {
	return s.listResponse, s.listTotal, s.listError
}

func (s stubAgentService) GetByIdGlobal(_ context.Context, _ string) (api.AgentDetail, error) {
	if s.getError != nil {
		return api.AgentDetail{}, s.getError
	}

	if s.detailResp.Id != uuid.Nil {
		return s.detailResp, nil
	}

	return api.AgentDetail{
		Id:          s.getResponse.Id,
		Name:        s.getResponse.Name,
		Type:        s.getResponse.Type,
		Version:     s.getResponse.Version,
		Metadata:    s.getResponse.Metadata,
		ServerCount: s.getResponse.ServerCount,
	}, nil
}

func (s stubAgentService) ListServers(_ context.Context, _ string, _ int, _ int) ([]api.AgentServer, int64, error) {
	return nil, 0, s.getError
}

func (s stubAgentService) CreateUnassigned(_ context.Context, _ api.AgentCreateRequest) (api.Agent, error) {
	if s.createError != nil {
		return api.Agent{}, s.createError
	}

	return s.createResp, nil
}

func (s stubAgentService) DeleteGlobal(_ context.Context, _ string) error {
	return s.detachError
}

func newAgentRouter(t *testing.T, svc Service) *gin.Engine {
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
		listResponse: []api.Agent{{Id: agentID, Name: "datadog", ServerCount: 1}},
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
	name := "datadog"
	r := newAgentRouter(t, stubAgentService{
		createResp: api.Agent{Id: agentID, Name: "datadog", ServerCount: 1},
	})

	path := "/api/v1/server/" + serverID.String() + "/agent"
	body := api.ServerAgentRequest{Name: &name}
	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, path, body, nil)
	response := testutil.AssertJSONStatus(t, rr, http.StatusCreated)

	data, ok := response["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "datadog", data["name"])
	require.Equal(t, float64(1), data["server_count"])
}

func TestHandlerGetReturnsAgent(t *testing.T) {
	serverID := uuid.New()
	agentID := uuid.New()
	r := newAgentRouter(t, stubAgentService{
		getResponse: api.Agent{Id: agentID, Name: "datadog"},
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
	name := "datadog"
	r := newAgentRouter(t, stubAgentService{createError: server.ErrServerNotFound})

	path := "/api/v1/server/" + serverID.String() + "/agent"
	body := api.ServerAgentRequest{Name: &name}
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

func TestHandlerDetachAgentReturnsNoContent(t *testing.T) {
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

func TestHandlerCreateReturnsConflictForDuplicateName(t *testing.T) {
	serverID := uuid.New()
	name := "datadog"
	r := newAgentRouter(t, stubAgentService{createError: ErrAgentAlreadyExists})

	path := "/api/v1/server/" + serverID.String() + "/agent"
	body := api.ServerAgentRequest{Name: &name}
	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, path, body, nil)
	require.Equal(t, http.StatusConflict, rr.Code)
}

func TestHandlerCreateUnassignedReturnsConflictForDuplicateName(t *testing.T) {
	r := newAgentRouter(t, stubAgentService{createError: ErrAgentAlreadyExists})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, "/api/v1/agent", api.AgentCreateRequest{Name: "datadog"}, nil)
	require.Equal(t, http.StatusConflict, rr.Code)
}

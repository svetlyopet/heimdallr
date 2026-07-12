package server

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
	"github.com/svetlyopet/heimdallr/internal/server/api"
	"github.com/svetlyopet/heimdallr/internal/testutil"
)

type stubServerService struct {
	getAllResponse      []api.ServerListItem
	getAllTotal         int64
	getAllError         error
	getByIdResp         api.ServerWithRelations
	getByIdError        error
	createResp          api.Server
	createError         error
	listJobsResp        []api.ServerJobAssociation
	listJobsErr         error
	associateJobErr     error
	listReleasesResp    []api.ServerReleaseAssociation
	listReleasesErr     error
	associateReleaseErr error
}

func (s stubServerService) GetAll(_ context.Context, _ string, _ int, _ int) ([]api.ServerListItem, int64, error) {
	if s.getAllError != nil {
		return nil, 0, s.getAllError
	}

	return s.getAllResponse, s.getAllTotal, nil
}

func (s stubServerService) GetById(_ context.Context, _ string) (api.ServerWithRelations, error) {
	if s.getByIdError != nil {
		return api.ServerWithRelations{}, s.getByIdError
	}

	return s.getByIdResp, nil
}

func (s stubServerService) Create(_ context.Context, _ api.ServerCreateRequest) (api.Server, error) {
	if s.createError != nil {
		return api.Server{}, s.createError
	}

	return s.createResp, nil
}

func (s stubServerService) Update(_ context.Context, _ string, _ api.ServerUpdateRequest) (api.ServerWithRelations, error) {
	return api.ServerWithRelations{}, nil
}

func (s stubServerService) ListJobs(_ context.Context, _ string, _ int, _ int) ([]api.ServerJobAssociation, int64, error) {
	if s.listJobsErr != nil {
		return nil, 0, s.listJobsErr
	}

	return s.listJobsResp, int64(len(s.listJobsResp)), nil
}

func (s stubServerService) AssociateJob(_ context.Context, _ string, _ api.ServerJobAssociateRequest) error {
	return s.associateJobErr
}

func (s stubServerService) DissociateJob(_ context.Context, _ string, _ string, _ uuid.UUID) error {
	return nil
}

func (s stubServerService) ListReleases(_ context.Context, _ string, _ int, _ int) ([]api.ServerReleaseAssociation, int64, error) {
	if s.listReleasesErr != nil {
		return nil, 0, s.listReleasesErr
	}

	return s.listReleasesResp, int64(len(s.listReleasesResp)), nil
}

func (s stubServerService) AssociateRelease(_ context.Context, _ string, _ api.ServerReleaseAssociateRequest) error {
	return s.associateReleaseErr
}

func (s stubServerService) DissociateRelease(_ context.Context, _ string, _ uuid.UUID) error {
	return nil
}

func newServerRouter(t *testing.T, svc Service) *gin.Engine {
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
	r := newServerRouter(t, stubServerService{})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, "/api/v1/server?page=0", nil, nil)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandlerListReturnsServers(t *testing.T) {
	serverID := uuid.New()
	r := newServerRouter(t, stubServerService{
		getAllResponse: []api.ServerListItem{{
			Id:       serverID,
			Hostname: "web-01.example.com",
		}},
		getAllTotal: 1,
	})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, "/api/v1/server", nil, nil)
	response := testutil.AssertJSONStatus(t, rr, http.StatusOK)

	data, ok := response["data"].([]any)
	require.True(t, ok)
	require.Len(t, data, 1)
}

func TestHandlerCreateReturnsCreated(t *testing.T) {
	serverID := uuid.New()
	r := newServerRouter(t, stubServerService{
		createResp: api.Server{Id: serverID, Hostname: "new-host.example.com"},
	})

	body := api.ServerCreateRequest{Hostname: "new-host.example.com"}
	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, "/api/v1/server", body, nil)
	response := testutil.AssertJSONStatus(t, rr, http.StatusCreated)

	data, ok := response["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "new-host.example.com", data["hostname"])
}

func TestHandlerCreateReturnsConflict(t *testing.T) {
	r := newServerRouter(t, stubServerService{createError: ErrServerAlreadyExists})

	body := api.ServerCreateRequest{Hostname: "dup-host.example.com"}
	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, "/api/v1/server", body, nil)
	require.Equal(t, http.StatusConflict, rr.Code)
}

func TestHandlerGetReturnsNotFound(t *testing.T) {
	serverID := uuid.New()
	r := newServerRouter(t, stubServerService{getByIdError: ErrServerNotFound})

	path := "/api/v1/server/" + serverID.String()
	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, path, nil, nil)
	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandlerGetReturnsBadRequestForInvalidID(t *testing.T) {
	r := newServerRouter(t, stubServerService{})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, "/api/v1/server/not-a-uuid", nil, nil)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandlerAssociateJobReturnsConflict(t *testing.T) {
	serverID := uuid.New()
	r := newServerRouter(t, stubServerService{associateJobErr: ErrJobAlreadyAssociated})

	path := "/api/v1/server/" + serverID.String() + "/job"
	body := api.ServerJobAssociateRequest{JobId: "1000", AutomationId: uuid.New()}
	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, path, body, nil)
	require.Equal(t, http.StatusConflict, rr.Code)
}

func TestHandlerDissociateJobRequiresAutomationID(t *testing.T) {
	serverID := uuid.New()
	r := newServerRouter(t, stubServerService{})

	path := "/api/v1/server/" + serverID.String() + "/job/1000"
	rr := testutil.DoGinJSONRequest(t, r, http.MethodDelete, path, nil, nil)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandlerAssociateReleaseReturnsNotFound(t *testing.T) {
	serverID := uuid.New()
	r := newServerRouter(t, stubServerService{associateReleaseErr: ErrReleaseNotFound})

	path := "/api/v1/server/" + serverID.String() + "/release"
	body := api.ServerReleaseAssociateRequest{ReleaseId: uuid.New(), ApplicationId: uuid.New()}
	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, path, body, nil)
	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandlerCreateReturnsBadRequestForInvalidBody(t *testing.T) {
	r := newServerRouter(t, stubServerService{})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, "/api/v1/server", json.RawMessage(`{}`), nil)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandlerListJobsReturnsData(t *testing.T) {
	serverID := uuid.New()
	r := newServerRouter(t, stubServerService{
		listJobsResp: []api.ServerJobAssociation{{
			JobId:  "1000",
			Status: api.Success,
		}},
	})

	path := "/api/v1/server/" + serverID.String() + "/job"
	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, path, nil, nil)
	response := testutil.AssertJSONStatus(t, rr, http.StatusOK)

	data, ok := response["data"].([]any)
	require.True(t, ok)
	require.Len(t, data, 1)
}

func TestHandlerListReturnsInternalServerError(t *testing.T) {
	r := newServerRouter(t, stubServerService{getAllError: errors.New("db down")})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, "/api/v1/server", nil, nil)
	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

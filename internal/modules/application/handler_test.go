package application

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/modules/application/api"
	"github.com/svetlyopet/heimdallr/internal/rbac"
	"github.com/svetlyopet/heimdallr/internal/testutil"
)

type stubApplicationService struct {
	getAllResponse []api.Application
	getAllTotal    int64
	getAllError    error
	getByIdResp    api.Application
	getByIdError   error
	createResp     api.Application
	createError    error
}

func (s stubApplicationService) GetAll(_ context.Context, _ int, _ int) ([]api.Application, int64, error) {
	if s.getAllError != nil {
		return nil, 0, s.getAllError
	}

	return s.getAllResponse, s.getAllTotal, nil
}

func (s stubApplicationService) GetById(_ context.Context, _ string) (api.Application, error) {
	if s.getByIdError != nil {
		return api.Application{}, s.getByIdError
	}

	return s.getByIdResp, nil
}

func (s stubApplicationService) GetByName(_ context.Context, _ string) (api.Application, error) {
	return api.Application{}, nil
}

func (s stubApplicationService) Create(_ context.Context, _ api.ApplicationCreateRequest) (api.Application, error) {
	if s.createError != nil {
		return api.Application{}, s.createError
	}

	return s.createResp, nil
}

func newApplicationRouter(t *testing.T, svc Service) *gin.Engine {
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
	r := newApplicationRouter(t, stubApplicationService{})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, "/api/v1/application?page=0", nil, nil)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandlerListReturnsApplications(t *testing.T) {
	appID := uuid.New()
	r := newApplicationRouter(t, stubApplicationService{
		getAllResponse: []api.Application{{Id: appID, Name: "demo-app"}},
		getAllTotal:    1,
	})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, "/api/v1/application", nil, nil)
	response := testutil.AssertJSONStatus(t, rr, http.StatusOK)
	require.NotNil(t, response["data"])
}

func TestHandlerGetReturnsNotFound(t *testing.T) {
	r := newApplicationRouter(t, stubApplicationService{getByIdError: ErrApplicationNotFound})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, "/api/v1/application/"+uuid.New().String(), nil, nil)
	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandlerCreateReturnsCreated(t *testing.T) {
	appID := uuid.New()
	repoURL := api.URL("https://example.com/new")
	r := newApplicationRouter(t, stubApplicationService{
		createResp: api.Application{Id: appID, Name: "new-app"},
	})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, "/api/v1/application", api.ApplicationCreateRequest{
		Name:          "new-app",
		Description:   ptr("desc"),
		RepositoryUrl: &repoURL,
	}, nil)
	response := testutil.AssertJSONStatus(t, rr, http.StatusCreated)
	data, ok := response["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "new-app", data["name"])
}

func TestHandlerCreateReturnsConflict(t *testing.T) {
	r := newApplicationRouter(t, stubApplicationService{createError: ErrApplicationAlreadyExists})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, "/api/v1/application", api.ApplicationCreateRequest{Name: "dup"}, nil)
	require.Equal(t, http.StatusConflict, rr.Code)
}

func TestHandlerListReturnsInternalServerError(t *testing.T) {
	r := newApplicationRouter(t, stubApplicationService{getAllError: errors.New("boom")})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, "/api/v1/application", nil, nil)
	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

func ptr(s string) *string {
	return &s
}

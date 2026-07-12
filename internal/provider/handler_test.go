package provider

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/provider/api"
	"github.com/svetlyopet/heimdallr/internal/rbac"
	"github.com/svetlyopet/heimdallr/internal/testutil"
)

type stubProviderService struct {
	listResponse []api.Provider
	listTotal    int64
	listError    error
	getError     error
	createResp   api.Provider
	createError  error
}

func (s stubProviderService) GetAll(_ context.Context, _ int, _ int) ([]api.Provider, int64, error) {
	if s.listError != nil {
		return nil, 0, s.listError
	}

	return s.listResponse, s.listTotal, nil
}

func (s stubProviderService) GetById(_ context.Context, _ string) (api.Provider, error) {
	if s.getError != nil {
		return api.Provider{}, s.getError
	}

	return api.Provider{}, nil
}

func (s stubProviderService) GetByName(_ context.Context, _ string) (api.Provider, error) {
	return api.Provider{}, nil
}

func (s stubProviderService) Create(_ context.Context, _ api.ProviderCreateRequest) (api.Provider, error) {
	if s.createError != nil {
		return api.Provider{}, s.createError
	}

	return s.createResp, nil
}

func newProviderRouter(t *testing.T, svc Service) *gin.Engine {
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

func TestHandlerListReturnsBadRequestForInvalidPage(t *testing.T) {
	r := newProviderRouter(t, stubProviderService{})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, "/api/v1/provider?page=0", nil, nil)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandlerCreateReturnsCreated(t *testing.T) {
	r := newProviderRouter(t, stubProviderService{
		createResp: api.Provider{Id: uuid.New(), Name: "awx", Url: api.URL("https://awx.example.com")},
	})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, "/api/v1/provider", api.ProviderCreateRequest{
		Name: "awx",
		Url:  api.URL("https://awx.example.com"),
	}, nil)
	response := testutil.AssertJSONStatus(t, rr, http.StatusCreated)
	data, ok := response["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "awx", data["name"])
}

func TestHandlerGetReturnsNotFound(t *testing.T) {
	r := newProviderRouter(t, stubProviderService{getError: ErrProviderNotFound})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, "/api/v1/provider/"+uuid.New().String(), nil, nil)
	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandlerListReturnsInternalServerError(t *testing.T) {
	r := newProviderRouter(t, stubProviderService{listError: errors.New("boom")})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, "/api/v1/provider", nil, nil)
	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

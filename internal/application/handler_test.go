package application

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/testutil"
)

type stubApplicationService struct {
	getAllResponse []GetResponse
	getAllTotal    int64
	getAllError    error
	getByIdResp    GetResponse
	getByIdError   error
	createResp     GetResponse
	createError    error
}

func (s stubApplicationService) GetAll(_ context.Context, _ int, _ int) ([]GetResponse, int64, error) {
	if s.getAllError != nil {
		return nil, 0, s.getAllError
	}

	return s.getAllResponse, s.getAllTotal, nil
}

func (s stubApplicationService) GetById(_ context.Context, _ string) (GetResponse, error) {
	if s.getByIdError != nil {
		return GetResponse{}, s.getByIdError
	}

	return s.getByIdResp, nil
}

func (s stubApplicationService) GetByName(_ context.Context, _ string) (GetResponse, error) {
	return GetResponse{}, nil
}

func (s stubApplicationService) Create(_ context.Context, _ CreateRequest) (GetResponse, error) {
	if s.createError != nil {
		return GetResponse{}, s.createError
	}

	return s.createResp, nil
}

func newApplicationRouter(t *testing.T, svc Service) *gin.Engine {
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
	r := newApplicationRouter(t, stubApplicationService{})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, "/api/v1/application?page=0", nil, nil)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandlerListReturnsApplications(t *testing.T) {
	appID := uuid.New()
	r := newApplicationRouter(t, stubApplicationService{
		getAllResponse: []GetResponse{{ID: appID, Name: "demo-app"}},
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
	r := newApplicationRouter(t, stubApplicationService{
		createResp: GetResponse{ID: appID, Name: "new-app"},
	})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, "/api/v1/application", CreateRequest{
		Name:          "new-app",
		Description:   "desc",
		RepositoryURL: "https://example.com/new",
	}, nil)
	response := testutil.AssertJSONStatus(t, rr, http.StatusCreated)
	data, ok := response["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "new-app", data["name"])
}

func TestHandlerCreateReturnsConflict(t *testing.T) {
	r := newApplicationRouter(t, stubApplicationService{createError: ErrApplicationAlreadyExists})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, "/api/v1/application", CreateRequest{Name: "dup"}, nil)
	require.Equal(t, http.StatusConflict, rr.Code)
}

func TestHandlerListReturnsInternalServerError(t *testing.T) {
	r := newApplicationRouter(t, stubApplicationService{getAllError: errors.New("boom")})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, "/api/v1/application", nil, nil)
	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

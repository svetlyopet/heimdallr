package automation

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/modules/automation/api"
	"github.com/svetlyopet/heimdallr/internal/rbac"
	"github.com/svetlyopet/heimdallr/internal/testutil"
)

type stubAutomationService struct {
	listResponse []api.Automation
	listTotal    int64
	listError    error
	getError     error
	createResp   api.Automation
	createError  error
}

func (s stubAutomationService) GetAll(_ context.Context, _ int, _ int) ([]api.Automation, int64, error) {
	if s.listError != nil {
		return nil, 0, s.listError
	}

	return s.listResponse, s.listTotal, nil
}

func (s stubAutomationService) GetByName(_ context.Context, _ string) (api.Automation, error) {
	return api.Automation{}, nil
}

func (s stubAutomationService) GetById(_ context.Context, _ string) (api.Automation, error) {
	if s.getError != nil {
		return api.Automation{}, s.getError
	}

	return api.Automation{}, nil
}

func (s stubAutomationService) Create(_ context.Context, _ api.AutomationCreateRequest) (api.Automation, error) {
	if s.createError != nil {
		return api.Automation{}, s.createError
	}

	return s.createResp, nil
}

func (s stubAutomationService) Update(_ context.Context, _ api.AutomationUpdateRequest, _ string) (api.Automation, error) {
	return api.Automation{}, nil
}

func (s stubAutomationService) Delete(_ context.Context, _ string) error {
	return nil
}

func newAutomationRouter(t *testing.T, svc Service) *gin.Engine {
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
	r := newAutomationRouter(t, stubAutomationService{})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, "/api/v1/automation?page=0", nil, nil)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandlerCreateReturnsInternalServerErrorForMissingProvider(t *testing.T) {
	r := newAutomationRouter(t, stubAutomationService{createError: ErrCreateAutomation})

	url := "https://awx.example.com/#/templates/job_template/1"
	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, "/api/v1/automation", api.AutomationCreateRequest{
		Name:       "deploy",
		Url:        &url,
		ProviderId: uuid.New(),
	}, nil)
	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestHandlerCreateReturnsCreated(t *testing.T) {
	r := newAutomationRouter(t, stubAutomationService{
		createResp: api.Automation{Id: uuid.New(), Name: "deploy", Provider: "awx"},
	})

	url := "https://awx.example.com/#/templates/job_template/1"
	rr := testutil.DoGinJSONRequest(t, r, http.MethodPost, "/api/v1/automation", api.AutomationCreateRequest{
		Name:       "deploy",
		Url:        &url,
		ProviderId: uuid.New(),
	}, nil)
	response := testutil.AssertJSONStatus(t, rr, http.StatusCreated)
	data, ok := response["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "deploy", data["name"])
}

func TestHandlerGetReturnsNotFound(t *testing.T) {
	r := newAutomationRouter(t, stubAutomationService{getError: ErrAutomationNotFound})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, "/api/v1/automation/"+uuid.New().String(), nil, nil)
	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestHandlerListReturnsInternalServerError(t *testing.T) {
	r := newAutomationRouter(t, stubAutomationService{listError: errors.New("boom")})

	rr := testutil.DoGinJSONRequest(t, r, http.MethodGet, "/api/v1/automation", nil, nil)
	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

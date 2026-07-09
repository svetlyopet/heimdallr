package analytics

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/analytics/api"
)

type stubService struct {
	getOverviewResponse     api.AutomationAnalytics
	getOverviewError        error
	getOverviewByIDResponse api.AutomationAnalytics
	getOverviewByIDError    error
	getComplianceResponse   api.ComplianceAnalytics
	getComplianceError      error
}

func (s stubService) GetAutomationOverview(_ context.Context) (api.AutomationAnalytics, error) {
	if s.getOverviewError != nil {
		return api.AutomationAnalytics{}, s.getOverviewError
	}

	return s.getOverviewResponse, nil
}

func (s stubService) GetAutomationOverviewByID(_ context.Context, _ string) (api.AutomationAnalytics, error) {
	if s.getOverviewByIDError != nil {
		return api.AutomationAnalytics{}, s.getOverviewByIDError
	}

	return s.getOverviewByIDResponse, nil
}

func (s stubService) GetComplianceOverview(_ context.Context) (api.ComplianceAnalytics, error) {
	if s.getComplianceError != nil {
		return api.ComplianceAnalytics{}, s.getComplianceError
	}

	return s.getComplianceResponse, nil
}

func newAnalyticsRouter(t *testing.T, svc Service) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)

	h, err := NewHandler(svc)
	require.NoError(t, err)

	r := gin.New()
	apiGroup := r.Group("/api")
	RegisterRoutes(apiGroup, h)

	return r
}

func TestNewHandlerFailsWithoutService(t *testing.T) {
	h, err := NewHandler(nil)
	require.Nil(t, h)
	require.Error(t, err)
}

func TestGetAutomationOverviewByIDReturnsBadRequestForInvalidID(t *testing.T) {
	r := newAnalyticsRouter(t, stubService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/automation/not-a-uuid", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestGetAutomationOverviewByIDReturnsNotFound(t *testing.T) {
	r := newAnalyticsRouter(t, stubService{getOverviewByIDError: ErrAutomationNotFound})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/automation/5d8dd803-fca6-4f7c-9dd2-24417622d630", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestGetAutomationOverviewReturnsInternalServerError(t *testing.T) {
	r := newAnalyticsRouter(t, stubService{getOverviewError: errors.New("db failed")})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/automation", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestGetComplianceOverviewReturnsOverview(t *testing.T) {
	r := newAnalyticsRouter(t, stubService{
		getComplianceResponse: api.ComplianceAnalytics{
			TotalApplications: 2,
			TotalReleases:     5,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/compliance", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
}

func TestGetComplianceOverviewReturnsInternalServerError(t *testing.T) {
	r := newAnalyticsRouter(t, stubService{getComplianceError: errors.New("db failed")})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/compliance", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

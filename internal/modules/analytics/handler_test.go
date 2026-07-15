package analytics

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/modules/analytics/api"
	"github.com/svetlyopet/heimdallr/internal/pagination"
	"github.com/svetlyopet/heimdallr/internal/rbac"
	"github.com/svetlyopet/heimdallr/internal/testutil"
)

type stubService struct {
	getOverviewResponse     api.AutomationAnalytics
	getOverviewError        error
	getOverviewByIDResponse api.AutomationAnalytics
	getOverviewByIDError    error
	getComplianceResponse   api.ComplianceAnalytics
	getComplianceError      error
	getFleetResponse        api.FleetComplianceAnalytics
	getFleetError           error
	listNonCompliantResp    []api.ServerFleetComplianceDetail
	listNonCompliantTotal   int64
	listNonCompliantError   error
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

func (s stubService) GetFleetComplianceOverview(_ context.Context) (api.FleetComplianceAnalytics, error) {
	if s.getFleetError != nil {
		return api.FleetComplianceAnalytics{}, s.getFleetError
	}

	return s.getFleetResponse, nil
}

func (s stubService) ListNonCompliantServers(_ context.Context, _ int, _ int) ([]api.ServerFleetComplianceDetail, api.Pagination, error) {
	if s.listNonCompliantError != nil {
		return nil, api.Pagination{}, s.listNonCompliantError
	}

	safeTotal, totalPages := pagination.SafeTotals(s.listNonCompliantTotal, 10)

	return s.listNonCompliantResp, api.Pagination{
		Page:       1,
		Limit:      10,
		Total:      safeTotal,
		TotalPages: totalPages,
	}, nil
}

func newAnalyticsRouter(t *testing.T, svc Service) *gin.Engine {
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

package analytics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/svetlyopet/heimdallr/internal/modules/analytics/api"
)

type stubRepository struct {
	getAutomationOverviewResponse     api.AutomationAnalytics
	getAutomationOverviewError        error
	getAutomationOverviewByIDResponse api.AutomationAnalytics
	getAutomationOverviewByIDError    error
}

func (s stubRepository) GetAutomationOverview(_ context.Context) (api.AutomationAnalytics, error) {
	if s.getAutomationOverviewError != nil {
		return api.AutomationAnalytics{}, s.getAutomationOverviewError
	}

	return s.getAutomationOverviewResponse, nil
}

func (s stubRepository) GetAutomationOverviewByID(_ context.Context, _ string) (api.AutomationAnalytics, error) {
	if s.getAutomationOverviewByIDError != nil {
		return api.AutomationAnalytics{}, s.getAutomationOverviewByIDError
	}

	return s.getAutomationOverviewByIDResponse, nil
}

func (s stubRepository) GetComplianceOverview(_ context.Context) (api.ComplianceAnalytics, error) {
	return api.ComplianceAnalytics{}, nil
}

func (s stubRepository) GetFleetComplianceOverview(_ context.Context) (api.FleetComplianceAnalytics, error) {
	return api.FleetComplianceAnalytics{}, nil
}

func TestServiceGetAutomationOverviewByIDReturnsNotFound(t *testing.T) {
	svc := NewService(stubRepository{getAutomationOverviewByIDError: ErrAutomationNotFound}, nil)

	_, err := svc.GetAutomationOverviewByID(t.Context(), "5d8dd803-fca6-4f7c-9dd2-24417622d630")
	require.ErrorIs(t, err, ErrAutomationNotFound)
}

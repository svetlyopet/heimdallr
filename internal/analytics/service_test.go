package analytics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubRepository struct {
	getAutomationOverviewResponse     AutomationAnalyticsResponse
	getAutomationOverviewError        error
	getAutomationOverviewByIDResponse AutomationAnalyticsResponse
	getAutomationOverviewByIDError    error
}

func (s stubRepository) GetAutomationOverview(_ context.Context) (AutomationAnalyticsResponse, error) {
	if s.getAutomationOverviewError != nil {
		return AutomationAnalyticsResponse{}, s.getAutomationOverviewError
	}

	return s.getAutomationOverviewResponse, nil
}

func (s stubRepository) GetAutomationOverviewByID(_ context.Context, _ string) (AutomationAnalyticsResponse, error) {
	if s.getAutomationOverviewByIDError != nil {
		return AutomationAnalyticsResponse{}, s.getAutomationOverviewByIDError
	}

	return s.getAutomationOverviewByIDResponse, nil
}

func (s stubRepository) GetComplianceOverview(_ context.Context) (ComplianceAnalyticsResponse, error) {
	return ComplianceAnalyticsResponse{}, nil
}

func TestServiceGetAutomationOverviewByIDReturnsNotFound(t *testing.T) {
	svc := NewService(stubRepository{getAutomationOverviewByIDError: ErrAutomationNotFound}, nil)

	_, err := svc.GetAutomationOverviewByID(t.Context(), "5d8dd803-fca6-4f7c-9dd2-24417622d630")
	require.ErrorIs(t, err, ErrAutomationNotFound)
}

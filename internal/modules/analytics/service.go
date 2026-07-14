package analytics

import (
	"context"
	"errors"
	"log/slog"

	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/modules/analytics/api"
)

type Service interface {
	GetAutomationOverview(ctx context.Context) (api.AutomationAnalytics, error)
	GetAutomationOverviewByID(ctx context.Context, automationID string) (api.AutomationAnalytics, error)
	GetComplianceOverview(ctx context.Context) (api.ComplianceAnalytics, error)
	GetFleetComplianceOverview(ctx context.Context) (api.FleetComplianceAnalytics, error)
}

type service struct {
	repository Repository
	logger     *logger.Logger
}

func (s service) GetAutomationOverview(ctx context.Context) (api.AutomationAnalytics, error) {
	response, err := s.repository.GetAutomationOverview(ctx)
	if err != nil {
		s.logger.ErrorWithStack(
			ctx,
			"failed to get automation analytics",
			err,
		)
		return api.AutomationAnalytics{}, ErrGetAutomationAnalytics
	}

	return response, nil
}

func (s service) GetAutomationOverviewByID(ctx context.Context, automationID string) (api.AutomationAnalytics, error) {
	response, err := s.repository.GetAutomationOverviewByID(ctx, automationID)
	if err != nil {
		if errors.Is(err, ErrAutomationNotFound) {
			return api.AutomationAnalytics{}, ErrAutomationNotFound
		}

		s.logger.ErrorWithStack(
			ctx,
			"failed to get automation analytics by id",
			err,
			slog.String("automation_id", automationID),
		)
		return api.AutomationAnalytics{}, ErrGetAutomationAnalytics
	}

	return response, nil
}

func (s service) GetComplianceOverview(ctx context.Context) (api.ComplianceAnalytics, error) {
	response, err := s.repository.GetComplianceOverview(ctx)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to get compliance analytics", err)
		return api.ComplianceAnalytics{}, ErrGetComplianceAnalytics
	}

	return response, nil
}

func (s service) GetFleetComplianceOverview(ctx context.Context) (api.FleetComplianceAnalytics, error) {
	response, err := s.repository.GetFleetComplianceOverview(ctx)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to get fleet compliance analytics", err)
		return api.FleetComplianceAnalytics{}, ErrGetFleetAnalytics
	}

	return response, nil
}

func NewService(repository Repository, appLogger *logger.Logger) Service {
	if appLogger == nil {
		appLogger = logger.Default()
	}

	return &service{
		repository: repository,
		logger:     appLogger,
	}
}

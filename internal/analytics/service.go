package analytics

import (
	"context"
	"log/slog"

	"github.com/svetlyopet/heimdallr/internal/logger"
)

type Service interface {
	GetAutomationOverview(ctx context.Context) (AutomationAnalyticsResponse, error)
	GetAutomationOverviewByID(ctx context.Context, automationID string) (AutomationAnalyticsResponse, error)
}

type service struct {
	repository Repository
	logger     *logger.Logger
}

func (s service) GetAutomationOverview(ctx context.Context) (AutomationAnalyticsResponse, error) {
	response, err := s.repository.GetAutomationOverview(ctx)
	if err != nil {
		s.logger.ErrorWithStack(
			ctx,
			"failed to get automation analytics",
			err,
		)
		return AutomationAnalyticsResponse{}, ErrGetAutomationAnalytics
	}

	return response, nil
}

func (s service) GetAutomationOverviewByID(ctx context.Context, automationID string) (AutomationAnalyticsResponse, error) {
	response, err := s.repository.GetAutomationOverviewByID(ctx, automationID)
	if err != nil {
		s.logger.ErrorWithStack(
			ctx,
			"failed to get automation analytics by id",
			err,
			slog.String("automation_id", automationID),
		)
		return AutomationAnalyticsResponse{}, ErrGetAutomationAnalytics
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

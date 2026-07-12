package automation

import (
	"context"
	"errors"
	"log/slog"

	"github.com/svetlyopet/heimdallr/internal/database"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/modules/automation/api"
	"github.com/svetlyopet/heimdallr/internal/modules/provider"
	"gorm.io/gorm"
)

type LookupService interface {
	GetByName(ctx context.Context, automationName string) (api.Automation, error)
	GetById(ctx context.Context, automationId string) (api.Automation, error)
}

type Service interface {
	GetAll(ctx context.Context, page int, limit int) ([]api.Automation, int64, error)
	GetByName(ctx context.Context, automationName string) (api.Automation, error)
	GetById(ctx context.Context, automationId string) (api.Automation, error)
	Create(ctx context.Context, req api.AutomationCreateRequest) (api.Automation, error)
	Update(ctx context.Context, req api.AutomationUpdateRequest, automationId string) (api.Automation, error)
	Delete(ctx context.Context, automationId string) error
}

type service struct {
	repository            Repository
	providerLookupService provider.LookupService
	logger                *logger.Logger
}

func (s service) GetAll(ctx context.Context, page int, limit int) ([]api.Automation, int64, error) {
	offset := (page - 1) * limit

	automations, total, err := s.repository.FindAll(ctx, limit, offset)
	if err != nil {
		s.logger.ErrorWithStack(
			ctx,
			"failed to find automations",
			err,
			slog.Int("page", page),
			slog.Int("limit", limit),
			slog.Int("offset", offset),
		)
		return nil, 0, ErrListAutomations
	}

	responses := make([]api.Automation, 0, len(automations))
	for _, automation := range automations {
		responses = append(responses, mapEntityToResponse(automation))
	}

	return responses, total, nil
}

func (s service) GetByName(ctx context.Context, automationName string) (api.Automation, error) {
	automation, err := s.repository.FindByName(ctx, automationName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.Automation{}, ErrAutomationNotFound
		}

		s.logger.ErrorWithStack(
			ctx,
			"failed to find automation by name",
			err,
			slog.String("automation_name", automationName),
		)
		return api.Automation{}, ErrGetAutomation
	}

	return mapEntityToResponse(automation), nil
}

func (s service) GetById(ctx context.Context, automationId string) (api.Automation, error) {
	automation, err := s.repository.FindById(ctx, automationId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.Automation{}, ErrAutomationNotFound
		}

		s.logger.ErrorWithStack(
			ctx,
			"failed to find automation by id",
			err,
			slog.String("automation_id", automationId),
		)
		return api.Automation{}, ErrGetAutomation
	}

	return mapEntityToResponse(automation), nil
}

func (s service) Create(ctx context.Context, req api.AutomationCreateRequest) (api.Automation, error) {
	exists, err := s.repository.ExistsByName(ctx, req.Name)
	if err != nil {
		s.logger.ErrorWithStack(
			ctx,
			"failed to check automation existence before create",
			err,
			slog.String("automation_name", req.Name),
		)
		return api.Automation{}, ErrCreateAutomation
	}

	if exists {
		return api.Automation{}, ErrAutomationAlreadyExists
	}

	automationProvider, err := s.providerLookupService.GetById(ctx, req.ProviderId.String())
	if err != nil {
		s.logger.ErrorWithStack(
			ctx,
			"failed to find provider before creating automation",
			err,
			slog.String("automation_name", req.Name),
			slog.String("provider_id", req.ProviderId.String()),
		)
		return api.Automation{}, ErrCreateAutomation
	}

	costSavings := float64(0)
	if req.CostSavings != nil {
		costSavings = *req.CostSavings
	}

	url := ""
	if req.Url != nil {
		url = *req.Url
	}

	automation := Automation{
		Name:        req.Name,
		Url:         url,
		Provider:    automationProvider.Name,
		ProviderID:  automationProvider.Id,
		CostSavings: costSavings,
	}

	createdAutomation, err := s.repository.Create(ctx, automation)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return api.Automation{}, ErrAutomationAlreadyExists
		}

		s.logger.ErrorWithStack(
			ctx,
			"failed to create automation",
			err,
			slog.String("automation_name", req.Name),
			slog.String("provider_id", req.ProviderId.String()),
		)
		return api.Automation{}, ErrCreateAutomation
	}

	automationWithProvider, err := s.repository.FindById(ctx, createdAutomation.ID.String())
	if err != nil {
		s.logger.ErrorWithStack(
			ctx,
			"failed to find created automation with provider",
			err,
			slog.String("automation_id", createdAutomation.ID.String()),
		)
		return api.Automation{}, ErrCreateAutomation
	}

	return mapEntityToResponse(automationWithProvider), nil
}

func (s service) Update(ctx context.Context, req api.AutomationUpdateRequest, automationId string) (api.Automation, error) {
	automationWithProvider, err := s.repository.FindById(ctx, automationId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.Automation{}, ErrAutomationNotFound
		}

		s.logger.ErrorWithStack(
			ctx,
			"failed to find automation before update",
			err,
			slog.String("automation_id", automationId),
		)
		return api.Automation{}, ErrUpdateAutomation
	}

	url := automationWithProvider.Url
	if req.Url != nil {
		url = *req.Url
	}

	costSavings := automationWithProvider.CostSavings
	if req.CostSavings != nil {
		costSavings = *req.CostSavings
	}

	automation := Automation{
		ID:          automationWithProvider.ID,
		Name:        automationWithProvider.Name,
		Url:         url,
		ProviderID:  automationWithProvider.ProviderID,
		CostSavings: costSavings,
	}

	updatedAutomation, err := s.repository.Update(ctx, automation)
	if err != nil {
		s.logger.ErrorWithStack(
			ctx,
			"failed to update automation",
			err,
			slog.String("automation_id", automationId),
		)
		return api.Automation{}, ErrUpdateAutomation
	}

	updatedAutomationWithProvider, err := s.repository.FindById(ctx, updatedAutomation.ID.String())
	if err != nil {
		s.logger.ErrorWithStack(
			ctx,
			"failed to find updated automation with provider",
			err,
			slog.String("automation_id", updatedAutomation.ID.String()),
		)
		return api.Automation{}, ErrUpdateAutomation
	}

	return mapEntityToResponse(updatedAutomationWithProvider), nil
}

func (s service) Delete(ctx context.Context, automationId string) error {
	if err := s.repository.Delete(ctx, automationId); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAutomationNotFound
		}

		s.logger.ErrorWithStack(
			ctx,
			"failed to delete automation",
			err,
			slog.String("automation_id", automationId),
		)
		return ErrDeleteAutomation
	}

	return nil
}

func NewService(repository Repository,
	providerLookupService provider.LookupService,
	appLogger *logger.Logger) Service {
	if appLogger == nil {
		appLogger = logger.Default()
	}

	return &service{
		repository:            repository,
		providerLookupService: providerLookupService,
		logger:                appLogger,
	}
}

func mapEntityToResponse(automation Automation) api.Automation {
	return api.Automation{
		Id:          automation.ID,
		Name:        automation.Name,
		Provider:    automation.Provider,
		ProviderId:  automation.ProviderID,
		Url:         automation.Url,
		CostSavings: automation.CostSavings,
	}
}

package automation

import (
	"context"
	"errors"
	"log/slog"

	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/provider"
	"gorm.io/gorm"
)

type LookupService interface {
	GetByName(ctx context.Context, automationName string) (GetResponse, error)
	GetById(ctx context.Context, automationId string) (GetResponse, error)
}

type Service interface {
	GetAll(ctx context.Context, page int, limit int) ([]GetResponse, int64, error)
	GetByName(ctx context.Context, automationName string) (GetResponse, error)
	GetById(ctx context.Context, automationId string) (GetResponse, error)
	Create(ctx context.Context, req CreateRequest) (GetResponse, error)
	Update(ctx context.Context, req UpdateRequest, automationId string) (GetResponse, error)
	Delete(ctx context.Context, automationId string) error
}

type service struct {
	repository            Repository
	providerLookupService provider.LookupService
	logger                *logger.Logger
}

func (s service) GetAll(ctx context.Context, page int, limit int) ([]GetResponse, int64, error) {
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

	responses := make([]GetResponse, 0, len(automations))
	for _, automation := range automations {
		responses = append(responses, mapEntityToResponse(automation))
	}

	return responses, total, nil
}

func (s service) GetByName(ctx context.Context, automationName string) (GetResponse, error) {
	automation, err := s.repository.FindByName(ctx, automationName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return GetResponse{}, ErrAutomationNotFound
		}

		s.logger.ErrorWithStack(
			ctx,
			"failed to find automation by name",
			err,
			slog.String("automation_name", automationName),
		)
		return GetResponse{}, ErrGetAutomation
	}

	return mapEntityToResponse(automation), nil
}

func (s service) GetById(ctx context.Context, automationId string) (GetResponse, error) {
	automation, err := s.repository.FindById(ctx, automationId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return GetResponse{}, ErrAutomationNotFound
		}

		s.logger.ErrorWithStack(
			ctx,
			"failed to find automation by id",
			err,
			slog.String("automation_id", automationId),
		)
		return GetResponse{}, ErrGetAutomation
	}

	return mapEntityToResponse(automation), nil
}

func (s service) Create(ctx context.Context, req CreateRequest) (GetResponse, error) {
	exists, err := s.repository.ExistsByName(ctx, req.Name)
	if err != nil {
		s.logger.ErrorWithStack(
			ctx,
			"failed to check automation existence before create",
			err,
			slog.String("automation_name", req.Name),
		)
		return GetResponse{}, ErrCreateAutomation
	}

	if exists {
		return GetResponse{}, ErrAutomationAlreadyExists
	}

	automationProvider, err := s.providerLookupService.GetById(ctx, req.ProviderID.String())
	if err != nil {
		s.logger.ErrorWithStack(
			ctx,
			"failed to find provider before creating automation",
			err,
			slog.String("automation_name", req.Name),
			slog.String("provider_id", req.ProviderID.String()),
		)
		return GetResponse{}, ErrCreateAutomation
	}

	automation := Automation{
		Name:        req.Name,
		Url:         req.URL,
		Provider:    automationProvider.Name,
		ProviderID:  automationProvider.ID,
		CostSavings: req.CostSavings,
	}

	createdAutomation, err := s.repository.Create(ctx, automation)
	if err != nil {
		s.logger.ErrorWithStack(
			ctx,
			"failed to create automation",
			err,
			slog.String("automation_name", req.Name),
			slog.String("provider_id", req.ProviderID.String()),
		)
		return GetResponse{}, ErrCreateAutomation
	}

	automationWithProvider, err := s.repository.FindById(ctx, createdAutomation.ID.String())
	if err != nil {
		s.logger.ErrorWithStack(
			ctx,
			"failed to find created automation with provider",
			err,
			slog.String("automation_id", createdAutomation.ID.String()),
		)
		return GetResponse{}, ErrCreateAutomation
	}

	return mapEntityToResponse(automationWithProvider), nil
}

func (s service) Update(ctx context.Context, req UpdateRequest, automationId string) (GetResponse, error) {
	automationWithProvider, err := s.repository.FindById(ctx, automationId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return GetResponse{}, ErrAutomationNotFound
		}

		s.logger.ErrorWithStack(
			ctx,
			"failed to find automation before update",
			err,
			slog.String("automation_id", automationId),
		)
		return GetResponse{}, ErrUpdateAutomation
	}

	automation := Automation{
		ID:          automationWithProvider.ID,
		Name:        automationWithProvider.Name,
		Url:         req.URL,
		ProviderID:  automationWithProvider.ProviderID,
		CostSavings: req.CostSavings,
	}

	updatedAutomation, err := s.repository.Update(ctx, automation)
	if err != nil {
		s.logger.ErrorWithStack(
			ctx,
			"failed to update automation",
			err,
			slog.String("automation_id", automationId),
		)
		return GetResponse{}, ErrUpdateAutomation
	}

	updatedAutomationWithProvider, err := s.repository.FindById(ctx, updatedAutomation.ID.String())
	if err != nil {
		s.logger.ErrorWithStack(
			ctx,
			"failed to find updated automation with provider",
			err,
			slog.String("automation_id", updatedAutomation.ID.String()),
		)
		return GetResponse{}, ErrUpdateAutomation
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

func mapEntityToResponse(automation Automation) GetResponse {
	return GetResponse{
		ID:          automation.ID,
		Name:        automation.Name,
		Provider:    automation.Provider,
		ProviderID:  automation.ProviderID,
		URL:         automation.Url,
		CostSavings: automation.CostSavings,
	}
}

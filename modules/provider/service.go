package provider

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/modules/provider/api"
	"gorm.io/gorm"
)

type LookupService interface {
	GetByName(ctx context.Context, providerName string) (api.Provider, error)
	GetById(ctx context.Context, providerId string) (api.Provider, error)
}

type Service interface {
	GetAll(ctx context.Context, page int, limit int) ([]api.Provider, int64, error)
	GetById(ctx context.Context, providerId string) (api.Provider, error)
	GetByName(ctx context.Context, providerName string) (api.Provider, error)
	Create(ctx context.Context, req api.ProviderCreateRequest) (api.Provider, error)
}

type service struct {
	repository Repository
	logger     *logger.Logger
}

func (s service) GetAll(ctx context.Context, page int, limit int) ([]api.Provider, int64, error) {
	offset := (page - 1) * limit

	providers, total, err := s.repository.FindAll(ctx, limit, offset)
	if err != nil {
		s.logger.ErrorWithStack(
			ctx,
			"failed to find providers",
			err,
			slog.Int("page", page),
			slog.Int("limit", limit),
			slog.Int("offset", offset),
		)
		return nil, 0, ErrListProviders
	}

	responses := make([]api.Provider, 0, len(providers))
	for _, provider := range providers {
		responses = append(responses, mapEntityToResponse(provider))
	}

	return responses, total, nil
}

func (s service) GetById(ctx context.Context, providerId string) (api.Provider, error) {
	provider, err := s.repository.FindById(ctx, providerId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.Provider{}, ErrProviderNotFound
		}

		s.logger.ErrorWithStack(
			ctx,
			"failed to find provider by id",
			err,
			slog.String("provider_id", providerId),
		)
		return api.Provider{}, ErrGetProvider
	}

	return mapEntityToResponse(provider), nil
}

func (s service) GetByName(ctx context.Context, providerName string) (api.Provider, error) {
	provider, err := s.repository.FindByName(ctx, providerName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.Provider{}, ErrProviderNotFound
		}

		s.logger.ErrorWithStack(
			ctx,
			"failed to find provider by name",
			err,
			slog.String("provider_name", providerName),
		)
		return api.Provider{}, ErrGetProvider
	}

	return mapEntityToResponse(provider), nil
}

func (s service) Create(ctx context.Context, req api.ProviderCreateRequest) (api.Provider, error) {
	_, err := s.repository.FindByName(ctx, req.Name)
	if err == nil {
		return api.Provider{}, ErrProviderAlreadyExists
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.ErrorWithStack(
			ctx,
			"failed to check provider existence before create",
			err,
			slog.String("provider_name", req.Name),
		)
		return api.Provider{}, ErrCreateProvider
	}

	provider := Provider{
		ID:   uuid.New(),
		Name: req.Name,
		Url:  string(req.Url),
	}

	createdProvider, err := s.repository.Create(ctx, provider)
	if err != nil {
		s.logger.ErrorWithStack(
			ctx,
			"failed to create provider",
			err,
			slog.String("provider_id", provider.ID.String()),
			slog.String("provider_name", provider.Name),
		)
		return api.Provider{}, ErrCreateProvider
	}

	return mapEntityToResponse(createdProvider), nil
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

func mapEntityToResponse(provider Provider) api.Provider {
	return api.Provider{
		Id:   provider.ID,
		Name: provider.Name,
		Url:  api.URL(provider.Url),
	}
}

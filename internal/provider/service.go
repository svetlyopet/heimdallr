package provider

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"gorm.io/gorm"
)

type LookupService interface {
	GetByName(ctx context.Context, providerName string) (GetResponse, error)
	GetById(ctx context.Context, providerId string) (GetResponse, error)
}

type Service interface {
	GetAll(ctx context.Context, page int, limit int) ([]GetResponse, int64, error)
	GetById(ctx context.Context, providerId string) (GetResponse, error)
	GetByName(ctx context.Context, providerName string) (GetResponse, error)
	Create(ctx context.Context, req CreateRequest) (GetResponse, error)
}

type service struct {
	repository Repository
	logger     *logger.Logger
}

func (s service) GetAll(ctx context.Context, page int, limit int) ([]GetResponse, int64, error) {
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

	responses := make([]GetResponse, 0, len(providers))
	for _, provider := range providers {
		responses = append(responses, mapEntityToResponse(provider))
	}

	return responses, total, nil
}

func (s service) GetById(ctx context.Context, providerId string) (GetResponse, error) {
	provider, err := s.repository.FindById(ctx, providerId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return GetResponse{}, ErrProviderNotFound
		}

		s.logger.ErrorWithStack(
			ctx,
			"failed to find provider by id",
			err,
			slog.String("provider_id", providerId),
		)
		return GetResponse{}, ErrGetProvider
	}

	return mapEntityToResponse(provider), nil
}

func (s service) GetByName(ctx context.Context, providerName string) (GetResponse, error) {
	provider, err := s.repository.FindByName(ctx, providerName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return GetResponse{}, ErrProviderNotFound
		}
	}

	return mapEntityToResponse(provider), nil
}

func (s service) Create(ctx context.Context, req CreateRequest) (GetResponse, error) {
	_, err := s.repository.FindByName(ctx, req.Name)
	if err == nil {
		return GetResponse{}, ErrProviderAlreadyExists
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.ErrorWithStack(
			ctx,
			"failed to check provider existence before create",
			err,
			slog.String("provider_name", req.Name),
		)
		return GetResponse{}, ErrCreateProvider
	}

	provider := Provider{
		ID:   uuid.New(),
		Name: req.Name,
		Url:  req.URL,
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
		return GetResponse{}, ErrCreateProvider
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

func mapEntityToResponse(provider Provider) GetResponse {
	return GetResponse{
		ID:   provider.ID,
		Name: provider.Name,
		URL:  provider.Url,
	}
}

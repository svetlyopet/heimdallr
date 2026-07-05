package application

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"gorm.io/gorm"
)

type LookupService interface {
	GetById(ctx context.Context, applicationID string) (GetResponse, error)
	GetByName(ctx context.Context, name string) (GetResponse, error)
}

type Service interface {
	GetAll(ctx context.Context, page int, limit int) ([]GetResponse, int64, error)
	GetById(ctx context.Context, applicationID string) (GetResponse, error)
	GetByName(ctx context.Context, name string) (GetResponse, error)
	Create(ctx context.Context, req CreateRequest) (GetResponse, error)
}

type service struct {
	repository Repository
	logger     *logger.Logger
}

func (s service) GetAll(ctx context.Context, page int, limit int) ([]GetResponse, int64, error) {
	offset := (page - 1) * limit

	applications, total, err := s.repository.FindAll(ctx, limit, offset)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to find applications", err,
			slog.Int("page", page),
			slog.Int("limit", limit),
		)
		return nil, 0, ErrListApplications
	}

	responses := make([]GetResponse, 0, len(applications))
	for _, application := range applications {
		responses = append(responses, mapEntityToResponse(application))
	}

	return responses, total, nil
}

func (s service) GetById(ctx context.Context, applicationID string) (GetResponse, error) {
	application, err := s.repository.FindById(ctx, applicationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return GetResponse{}, ErrApplicationNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to find application by id", err,
			slog.String("application_id", applicationID),
		)
		return GetResponse{}, ErrGetApplication
	}

	return mapEntityToResponse(application), nil
}

func (s service) GetByName(ctx context.Context, name string) (GetResponse, error) {
	application, err := s.repository.FindByName(ctx, name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return GetResponse{}, ErrApplicationNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to find application by name", err,
			slog.String("application_name", name),
		)
		return GetResponse{}, ErrGetApplication
	}

	return mapEntityToResponse(application), nil
}

func (s service) Create(ctx context.Context, req CreateRequest) (GetResponse, error) {
	_, err := s.repository.FindByName(ctx, req.Name)
	if err == nil {
		return GetResponse{}, ErrApplicationAlreadyExists
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.ErrorWithStack(ctx, "failed to check application existence before create", err,
			slog.String("application_name", req.Name),
		)
		return GetResponse{}, ErrCreateApplication
	}

	application := Application{
		ID:            uuid.New(),
		Name:          req.Name,
		Description:   req.Description,
		RepositoryURL: req.RepositoryURL,
	}

	created, err := s.repository.Create(ctx, application)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to create application", err,
			slog.String("application_name", req.Name),
		)
		return GetResponse{}, ErrCreateApplication
	}

	return mapEntityToResponse(created), nil
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

func mapEntityToResponse(application Application) GetResponse {
	return GetResponse{
		ID:            application.ID,
		Name:          application.Name,
		Description:   application.Description,
		RepositoryURL: application.RepositoryURL,
	}
}

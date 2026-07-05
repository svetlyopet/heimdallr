package release

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/application"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"gorm.io/gorm"
)

type LookupService interface {
	GetById(ctx context.Context, releaseID string, applicationID string) (GetWithSummaryResponse, error)
}

type Service interface {
	GetAll(ctx context.Context, applicationID string, page int, limit int) ([]ListItemResponse, int64, error)
	GetById(ctx context.Context, releaseID string, applicationID string) (GetWithSummaryResponse, error)
	Create(ctx context.Context, applicationID string, req CreateRequest, upsert bool) (GetResponse, error)
}

type service struct {
	repository               Repository
	applicationLookupService application.LookupService
	logger                   *logger.Logger
}

func (s service) GetAll(ctx context.Context, applicationID string, page int, limit int) ([]ListItemResponse, int64, error) {
	if _, err := uuid.Parse(applicationID); err != nil {
		return nil, 0, ErrInvalidApplicationID
	}

	if _, err := s.applicationLookupService.GetById(ctx, applicationID); err != nil {
		if errors.Is(err, application.ErrApplicationNotFound) {
			return nil, 0, application.ErrApplicationNotFound
		}

		return nil, 0, ErrListReleases
	}

	offset := (page - 1) * limit

	releases, total, err := s.repository.FindAll(ctx, applicationID, limit, offset)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to find releases", err,
			slog.String("application_id", applicationID),
		)
		return nil, 0, ErrListReleases
	}

	releaseIDs := make([]uuid.UUID, 0, len(releases))
	for _, release := range releases {
		releaseIDs = append(releaseIDs, release.ID)
	}

	summaries, err := s.repository.GetComplianceSummariesForReleases(ctx, releaseIDs)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to get release compliance summaries", err,
			slog.String("application_id", applicationID),
		)
		return nil, 0, ErrListReleases
	}

	responses := make([]ListItemResponse, 0, len(releases))
	for _, release := range releases {
		summary := summaries[release.ID]
		responses = append(responses, ListItemResponse{
			GetResponse: mapEntityToResponse(release),
			Compliance:  summary,
		})
	}

	return responses, total, nil
}

func (s service) GetById(ctx context.Context, releaseID string, applicationID string) (GetWithSummaryResponse, error) {
	if _, err := uuid.Parse(applicationID); err != nil {
		return GetWithSummaryResponse{}, ErrInvalidApplicationID
	}

	release, err := s.repository.FindById(ctx, releaseID, applicationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return GetWithSummaryResponse{}, ErrReleaseNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to find release by id", err,
			slog.String("release_id", releaseID),
			slog.String("application_id", applicationID),
		)
		return GetWithSummaryResponse{}, ErrGetRelease
	}

	summary, err := s.repository.GetComplianceSummary(ctx, release.ID)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to get release compliance summary", err,
			slog.String("release_id", releaseID),
		)
		return GetWithSummaryResponse{}, ErrGetRelease
	}

	return GetWithSummaryResponse{
		GetResponse: mapEntityToResponse(release),
		Compliance:  summary,
	}, nil
}

func (s service) Create(ctx context.Context, applicationID string, req CreateRequest, upsert bool) (GetResponse, error) {
	parsedApplicationID, err := uuid.Parse(applicationID)
	if err != nil {
		return GetResponse{}, ErrInvalidApplicationID
	}

	app, err := s.applicationLookupService.GetById(ctx, applicationID)
	if err != nil {
		if errors.Is(err, application.ErrApplicationNotFound) {
			return GetResponse{}, application.ErrApplicationNotFound
		}

		return GetResponse{}, ErrCreateRelease
	}

	version := strings.TrimSpace(req.Version)
	if version == "" {
		return GetResponse{}, ErrCreateRelease
	}

	release := Release{
		ID:            uuid.New(),
		ApplicationID: parsedApplicationID,
		Application:   app.Name,
		Version:       version,
		CommitSHA:     strings.TrimSpace(req.CommitSHA),
		PipelineURL:   strings.TrimSpace(req.PipelineURL),
		Branch:        strings.TrimSpace(req.Branch),
	}

	if upsert {
		created, upsertErr := s.repository.Upsert(ctx, release)
		if upsertErr != nil {
			s.logger.ErrorWithStack(ctx, "failed to upsert release", upsertErr,
				slog.String("application_id", applicationID),
				slog.String("version", version),
			)
			return GetResponse{}, ErrCreateRelease
		}

		return mapEntityToResponse(created), nil
	}

	_, findErr := s.repository.FindByApplicationAndVersion(ctx, parsedApplicationID, version)
	if findErr == nil {
		return GetResponse{}, ErrReleaseAlreadyExists
	}

	if !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return GetResponse{}, ErrCreateRelease
	}

	created, err := s.repository.Create(ctx, release)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to create release", err,
			slog.String("application_id", applicationID),
			slog.String("version", version),
		)
		return GetResponse{}, ErrCreateRelease
	}

	return mapEntityToResponse(created), nil
}

func NewService(
	repository Repository,
	applicationLookupService application.LookupService,
	appLogger *logger.Logger,
) Service {
	if appLogger == nil {
		appLogger = logger.Default()
	}

	return &service{
		repository:               repository,
		applicationLookupService: applicationLookupService,
		logger:                   appLogger,
	}
}

func mapEntityToResponse(release Release) GetResponse {
	return GetResponse{
		ID:            release.ID,
		ApplicationID: release.ApplicationID,
		Application:   release.Application,
		Version:       release.Version,
		CommitSHA:     release.CommitSHA,
		PipelineURL:   release.PipelineURL,
		Branch:        release.Branch,
		CreatedAt:     release.CreatedAt,
	}
}

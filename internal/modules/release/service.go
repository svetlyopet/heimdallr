package release

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/database"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/modules/application"
	"github.com/svetlyopet/heimdallr/internal/modules/release/api"
	"gorm.io/gorm"
)

type LookupService interface {
	GetById(ctx context.Context, releaseID string, applicationID string) (api.ReleaseWithCompliance, error)
}

type Service interface {
	GetAll(ctx context.Context, applicationID string, page int, limit int) ([]api.ReleaseListItem, int64, error)
	GetById(ctx context.Context, releaseID string, applicationID string) (api.ReleaseWithCompliance, error)
	Create(ctx context.Context, applicationID string, req api.ReleaseCreateRequest, upsert bool) (api.Release, error)
}

type service struct {
	repository               Repository
	applicationLookupService application.LookupService
	logger                   *logger.Logger
}

func (s service) GetAll(ctx context.Context, applicationID string, page int, limit int) ([]api.ReleaseListItem, int64, error) {
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

	responses := make([]api.ReleaseListItem, 0, len(releases))
	for _, release := range releases {
		mapped := mapEntityToResponse(release)
		responses = append(responses, api.ReleaseListItem{
			Id:            mapped.Id,
			ApplicationId: mapped.ApplicationId,
			Application:   mapped.Application,
			Version:       mapped.Version,
			CommitSha:     mapped.CommitSha,
			PipelineUrl:   mapped.PipelineUrl,
			Branch:        mapped.Branch,
			CreatedAt:     mapped.CreatedAt,
			Compliance:    summaries[release.ID],
		})
	}

	return responses, total, nil
}

func (s service) GetById(ctx context.Context, releaseID string, applicationID string) (api.ReleaseWithCompliance, error) {
	if _, err := uuid.Parse(applicationID); err != nil {
		return api.ReleaseWithCompliance{}, ErrInvalidApplicationID
	}

	release, err := s.repository.FindById(ctx, releaseID, applicationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.ReleaseWithCompliance{}, ErrReleaseNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to find release by id", err,
			slog.String("release_id", releaseID),
			slog.String("application_id", applicationID),
		)
		return api.ReleaseWithCompliance{}, ErrGetRelease
	}

	summary, err := s.repository.GetComplianceSummary(ctx, release.ID)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to get release compliance summary", err,
			slog.String("release_id", releaseID),
		)
		return api.ReleaseWithCompliance{}, ErrGetRelease
	}

	mapped := mapEntityToResponse(release)
	return api.ReleaseWithCompliance{
		Id:            mapped.Id,
		ApplicationId: mapped.ApplicationId,
		Application:   mapped.Application,
		Version:       mapped.Version,
		CommitSha:     mapped.CommitSha,
		PipelineUrl:   mapped.PipelineUrl,
		Branch:        mapped.Branch,
		CreatedAt:     mapped.CreatedAt,
		Compliance:    summary,
	}, nil
}

func (s service) Create(ctx context.Context, applicationID string, req api.ReleaseCreateRequest, upsert bool) (api.Release, error) {
	parsedApplicationID, err := uuid.Parse(applicationID)
	if err != nil {
		return api.Release{}, ErrInvalidApplicationID
	}

	app, err := s.applicationLookupService.GetById(ctx, applicationID)
	if err != nil {
		if errors.Is(err, application.ErrApplicationNotFound) {
			return api.Release{}, application.ErrApplicationNotFound
		}

		return api.Release{}, ErrCreateRelease
	}

	version := strings.TrimSpace(req.Version)
	if version == "" {
		return api.Release{}, ErrCreateRelease
	}

	commitSHA := ""
	if req.CommitSha != nil {
		commitSHA = strings.TrimSpace(*req.CommitSha)
	}

	pipelineURL := ""
	if req.PipelineUrl != nil {
		pipelineURL = string(*req.PipelineUrl)
	}

	branch := ""
	if req.Branch != nil {
		branch = strings.TrimSpace(*req.Branch)
	}

	release := Release{
		ID:            uuid.New(),
		ApplicationID: parsedApplicationID,
		Application:   app.Name,
		Version:       version,
		CommitSHA:     commitSHA,
		PipelineURL:   pipelineURL,
		Branch:        branch,
	}

	if upsert {
		created, upsertErr := s.repository.Upsert(ctx, release)
		if upsertErr != nil {
			s.logger.ErrorWithStack(ctx, "failed to upsert release", upsertErr,
				slog.String("application_id", applicationID),
				slog.String("version", version),
			)
			return api.Release{}, ErrCreateRelease
		}

		return mapEntityToResponse(created), nil
	}

	_, findErr := s.repository.FindByApplicationAndVersion(ctx, parsedApplicationID, version)
	if findErr == nil {
		return api.Release{}, ErrReleaseAlreadyExists
	}

	if !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return api.Release{}, ErrCreateRelease
	}

	created, err := s.repository.Create(ctx, release)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return api.Release{}, ErrReleaseAlreadyExists
		}

		s.logger.ErrorWithStack(ctx, "failed to create release", err,
			slog.String("application_id", applicationID),
			slog.String("version", version),
		)
		return api.Release{}, ErrCreateRelease
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

func mapEntityToResponse(release Release) api.Release {
	return api.Release{
		Id:            release.ID,
		ApplicationId: release.ApplicationID,
		Application:   release.Application,
		Version:       release.Version,
		CommitSha:     release.CommitSHA,
		PipelineUrl:   api.URL(release.PipelineURL),
		Branch:        release.Branch,
		CreatedAt:     release.CreatedAt,
	}
}

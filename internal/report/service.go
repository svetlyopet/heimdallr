package report

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/release"
	"gorm.io/gorm"
)

type Service interface {
	GetAll(ctx context.Context, applicationID string, releaseID string, page int, limit int) ([]GetResponse, int64, error)
	GetAllGlobal(ctx context.Context, filters ListFilters, page int, limit int) ([]GetResponse, int64, error)
	GetById(ctx context.Context, applicationID string, releaseID string, reportID string) (GetResponse, error)
	Create(ctx context.Context, applicationID string, releaseID string, req CreateRequest) (GetResponse, error)
	Update(ctx context.Context, applicationID string, releaseID string, reportID string, req UpdateRequest) (GetResponse, error)
}

type service struct {
	repository           Repository
	releaseLookupService release.LookupService
	logger               *logger.Logger
}

func (s service) GetAll(ctx context.Context, applicationID string, releaseID string, page int, limit int) ([]GetResponse, int64, error) {
	if _, err := s.releaseLookupService.GetById(ctx, releaseID, applicationID); err != nil {
		if errors.Is(err, release.ErrReleaseNotFound) {
			return nil, 0, ErrReportNotFound
		}

		return nil, 0, ErrListReports
	}

	offset := (page - 1) * limit

	reports, total, err := s.repository.FindAll(ctx, releaseID, applicationID, limit, offset)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to find reports", err,
			slog.String("release_id", releaseID),
			slog.String("application_id", applicationID),
		)
		return nil, 0, ErrListReports
	}

	responses := make([]GetResponse, 0, len(reports))
	for _, report := range reports {
		reportResponse, mapErr := mapEntityToResponse(report)
		if mapErr != nil {
			return nil, 0, ErrListReports
		}

		responses = append(responses, reportResponse)
	}

	return responses, total, nil
}

func (s service) GetAllGlobal(ctx context.Context, filters ListFilters, page int, limit int) ([]GetResponse, int64, error) {
	if filters.ApplicationID != "" {
		if _, err := uuid.Parse(filters.ApplicationID); err != nil {
			return nil, 0, ErrInvalidApplicationID
		}
	}

	if filters.ReleaseID != "" {
		if _, err := uuid.Parse(filters.ReleaseID); err != nil {
			return nil, 0, ErrInvalidReleaseID
		}
	}

	offset := (page - 1) * limit

	reports, total, err := s.repository.FindAllGlobal(ctx, filters, limit, offset)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to find reports globally", err)
		return nil, 0, ErrListReports
	}

	responses := make([]GetResponse, 0, len(reports))
	for _, report := range reports {
		reportResponse, mapErr := mapEntityToResponse(report)
		if mapErr != nil {
			return nil, 0, ErrListReports
		}

		responses = append(responses, reportResponse)
	}

	return responses, total, nil
}

func (s service) GetById(ctx context.Context, applicationID string, releaseID string, reportID string) (GetResponse, error) {
	report, err := s.repository.FindById(ctx, reportID, releaseID, applicationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return GetResponse{}, ErrReportNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to find report by id", err,
			slog.String("report_id", reportID),
			slog.String("release_id", releaseID),
		)
		return GetResponse{}, ErrGetReport
	}

	return mapEntityToResponse(report)
}

func (s service) Create(ctx context.Context, applicationID string, releaseID string, req CreateRequest) (GetResponse, error) {
	parsedReleaseID, err := uuid.Parse(releaseID)
	if err != nil {
		return GetResponse{}, ErrInvalidReleaseID
	}

	parsedApplicationID, err := uuid.Parse(applicationID)
	if err != nil {
		return GetResponse{}, ErrInvalidApplicationID
	}

	if _, err := s.releaseLookupService.GetById(ctx, releaseID, applicationID); err != nil {
		if errors.Is(err, release.ErrReleaseNotFound) {
			return GetResponse{}, ErrReportNotFound
		}

		return GetResponse{}, ErrCreateReport
	}

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		return GetResponse{}, NewInvalidMetadataError(err)
	}

	report := Report{
		ID:            req.ID,
		ReleaseID:     parsedReleaseID,
		ApplicationID: parsedApplicationID,
		Type:          req.Type,
		Status:        req.Status,
		Location:      req.Location,
		URL:           req.URL,
		Metadata:      metadata,
		Output:        req.Output,
	}

	created, err := s.repository.Create(ctx, report)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to create report", err,
			slog.String("report_id", req.ID),
			slog.String("release_id", releaseID),
		)
		return GetResponse{}, ErrCreateReport
	}

	return mapEntityToResponse(created)
}

func (s service) Update(ctx context.Context, applicationID string, releaseID string, reportID string, req UpdateRequest) (GetResponse, error) {
	parsedReleaseID, err := uuid.Parse(releaseID)
	if err != nil {
		return GetResponse{}, ErrInvalidReleaseID
	}

	parsedApplicationID, err := uuid.Parse(applicationID)
	if err != nil {
		return GetResponse{}, ErrInvalidApplicationID
	}

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		return GetResponse{}, NewInvalidMetadataError(err)
	}

	report := Report{
		ID:            reportID,
		ReleaseID:     parsedReleaseID,
		ApplicationID: parsedApplicationID,
		Status:        req.Status,
		Metadata:      metadata,
		Output:        req.Output,
	}

	updated, err := s.repository.Update(ctx, report)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return GetResponse{}, ErrReportNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to update report", err,
			slog.String("report_id", reportID),
			slog.String("release_id", releaseID),
		)
		return GetResponse{}, ErrUpdateReport
	}

	return mapEntityToResponse(updated)
}

func NewService(
	repository Repository,
	releaseLookupService release.LookupService,
	appLogger *logger.Logger,
) Service {
	if appLogger == nil {
		appLogger = logger.Default()
	}

	return &service{
		repository:           repository,
		releaseLookupService: releaseLookupService,
		logger:               appLogger,
	}
}

func mapEntityToResponse(report Report) (GetResponse, error) {
	metadata, err := json.Marshal(report.Metadata)
	if err != nil {
		return GetResponse{}, err
	}

	return GetResponse{
		ID:            report.ID,
		ApplicationID: report.ApplicationID,
		ReleaseID:     report.ReleaseID,
		Application:   report.Application,
		Version:       report.Version,
		Type:          report.Type,
		Status:        report.Status,
		Location:      report.Location,
		URL:           report.URL,
		Metadata:      metadata,
		Output:        report.Output,
		CreatedAt:     report.CreatedAt,
	}, nil
}

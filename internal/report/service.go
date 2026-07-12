package report

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/release"
	"github.com/svetlyopet/heimdallr/internal/report/api"
	"github.com/svetlyopet/heimdallr/internal/requestlimits"
	"github.com/svetlyopet/heimdallr/internal/validation"
	"gorm.io/gorm"
)

type Service interface {
	GetAll(ctx context.Context, applicationID string, releaseID string, page int, limit int) ([]api.Report, int64, error)
	GetAllGlobal(ctx context.Context, filters ListFilters, page int, limit int) ([]api.Report, int64, error)
	GetById(ctx context.Context, applicationID string, releaseID string, reportID string) (api.Report, error)
	Create(ctx context.Context, applicationID string, releaseID string, req api.ReportCreateRequest) (api.Report, error)
	Update(ctx context.Context, applicationID string, releaseID string, reportID string, req api.ReportUpdateRequest) (api.Report, error)
}

type service struct {
	repository           Repository
	releaseLookupService release.LookupService
	logger               *logger.Logger
}

func (s service) GetAll(ctx context.Context, applicationID string, releaseID string, page int, limit int) ([]api.Report, int64, error) {
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

	responses := make([]api.Report, 0, len(reports))
	for _, report := range reports {
		reportResponse, mapErr := mapEntityToResponse(report)
		if mapErr != nil {
			return nil, 0, ErrListReports
		}

		responses = append(responses, reportResponse)
	}

	return responses, total, nil
}

func (s service) GetAllGlobal(ctx context.Context, filters ListFilters, page int, limit int) ([]api.Report, int64, error) {
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

	responses := make([]api.Report, 0, len(reports))
	for _, report := range reports {
		reportResponse, mapErr := mapEntityToResponse(report)
		if mapErr != nil {
			return nil, 0, ErrListReports
		}

		responses = append(responses, reportResponse)
	}

	return responses, total, nil
}

func (s service) GetById(ctx context.Context, applicationID string, releaseID string, reportID string) (api.Report, error) {
	report, err := s.repository.FindById(ctx, reportID, releaseID, applicationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.Report{}, ErrReportNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to find report by id", err,
			slog.String("report_id", reportID),
			slog.String("release_id", releaseID),
		)
		return api.Report{}, ErrGetReport
	}

	return mapEntityToResponse(report)
}

func (s service) Create(ctx context.Context, applicationID string, releaseID string, req api.ReportCreateRequest) (api.Report, error) {
	parsedReleaseID, err := uuid.Parse(releaseID)
	if err != nil {
		return api.Report{}, ErrInvalidReleaseID
	}

	parsedApplicationID, err := uuid.Parse(applicationID)
	if err != nil {
		return api.Report{}, ErrInvalidApplicationID
	}

	output := ""
	if req.Output != nil {
		output = *req.Output
	}
	if outputErr := validation.ValidateBase64Output(output, requestlimits.MaxDecodedOutputBytes(ctx)); outputErr != nil {
		return api.Report{}, NewInvalidOutputError(outputErr)
	}

	if _, err := s.releaseLookupService.GetById(ctx, releaseID, applicationID); err != nil {
		if errors.Is(err, release.ErrReleaseNotFound) {
			return api.Report{}, ErrReportNotFound
		}

		return api.Report{}, ErrCreateReport
	}

	metadata, err := marshalMetadata(req.Metadata)
	if err != nil {
		return api.Report{}, NewInvalidMetadataError(err)
	}

	location := ""
	if req.Location != nil {
		location = *req.Location
	}

	url := ""
	if req.Url != nil {
		url = string(*req.Url)
	}

	report := Report{
		ID:            req.Id,
		ReleaseID:     parsedReleaseID,
		ApplicationID: parsedApplicationID,
		Type:          string(req.Type),
		Status:        string(req.Status),
		Location:      location,
		URL:           url,
		Metadata:      metadata,
		Output:        output,
	}

	created, err := s.repository.Create(ctx, report)
	if err != nil {
		s.logger.ErrorWithStack(ctx, "failed to create report", err,
			slog.String("report_id", req.Id),
			slog.String("release_id", releaseID),
		)
		return api.Report{}, ErrCreateReport
	}

	return mapEntityToResponse(created)
}

func (s service) Update(ctx context.Context, applicationID string, releaseID string, reportID string, req api.ReportUpdateRequest) (api.Report, error) {
	parsedReleaseID, err := uuid.Parse(releaseID)
	if err != nil {
		return api.Report{}, ErrInvalidReleaseID
	}

	parsedApplicationID, err := uuid.Parse(applicationID)
	if err != nil {
		return api.Report{}, ErrInvalidApplicationID
	}

	metadata, err := marshalMetadata(req.Metadata)
	if err != nil {
		return api.Report{}, NewInvalidMetadataError(err)
	}

	output := ""
	if req.Output != nil {
		output = *req.Output
	}
	if outputErr := validation.ValidateBase64Output(output, requestlimits.MaxDecodedOutputBytes(ctx)); outputErr != nil {
		return api.Report{}, NewInvalidOutputError(outputErr)
	}

	report := Report{
		ID:            reportID,
		ReleaseID:     parsedReleaseID,
		ApplicationID: parsedApplicationID,
		Status:        string(req.Status),
		Metadata:      metadata,
		Output:        output,
	}

	updated, err := s.repository.Update(ctx, report)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.Report{}, ErrReportNotFound
		}

		s.logger.ErrorWithStack(ctx, "failed to update report", err,
			slog.String("report_id", reportID),
			slog.String("release_id", releaseID),
		)
		return api.Report{}, ErrUpdateReport
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

func mapEntityToResponse(report Report) (api.Report, error) {
	var metadataPtr *api.ReportMetadata
	if len(report.Metadata) > 0 && string(report.Metadata) != "null" {
		var metadata api.ReportMetadata
		if err := json.Unmarshal(report.Metadata, &metadata); err != nil {
			return api.Report{}, err
		}
		metadataPtr = &metadata
	}

	var locationPtr *string
	if report.Location != "" {
		locationPtr = &report.Location
	}

	var urlPtr *string
	if report.URL != "" {
		urlPtr = &report.URL
	}

	var outputPtr *string
	if report.Output != "" {
		outputPtr = &report.Output
	}

	return api.Report{
		Id:            report.ID,
		ApplicationId: report.ApplicationID,
		ReleaseId:     report.ReleaseID,
		Application:   report.Application,
		Version:       report.Version,
		Type:          api.ReportType(report.Type),
		Status:        api.JobStatus(report.Status),
		Location:      locationPtr,
		Url:           urlPtr,
		Metadata:      metadataPtr,
		Output:        outputPtr,
		CreatedAt:     report.CreatedAt,
	}, nil
}

func marshalMetadata(metadata *api.ReportMetadata) ([]byte, error) {
	if metadata == nil {
		return []byte("null"), nil
	}

	return json.Marshal(metadata)
}

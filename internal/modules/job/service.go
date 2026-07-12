package job

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/modules/automation"
	"github.com/svetlyopet/heimdallr/internal/modules/job/api"
	"github.com/svetlyopet/heimdallr/internal/requestlimits"
	"github.com/svetlyopet/heimdallr/internal/validation"
	"gorm.io/gorm"
)

type Service interface {
	GetAll(ctx context.Context, automationId string, page int, limit int) ([]api.Job, int64, error)
	GetById(ctx context.Context, jobId string, automationId string) (api.Job, error)
	Create(ctx context.Context, automationId string, req api.JobCreateRequest) (api.Job, error)
	Update(ctx context.Context, automationId string, jobId string, req api.JobUpdateRequest) (api.Job, error)
}

type service struct {
	repository              Repository
	automationLookupService automation.LookupService
	logger                  *logger.Logger
}

func (s service) GetAll(ctx context.Context, automationId string, page int, limit int) ([]api.Job, int64, error) {
	offset := (page - 1) * limit

	jobs, total, err := s.repository.FindAll(ctx, automationId, limit, offset)
	if err != nil {
		s.logger.ErrorWithStack(
			ctx,
			"failed to find jobs",
			err,
			slog.String("automation_id", automationId),
			slog.Int("page", page),
			slog.Int("limit", limit),
			slog.Int("offset", offset),
		)
		return nil, 0, ErrListJobs
	}

	responses := make([]api.Job, 0, len(jobs))
	for _, job := range jobs {
		jobResponse, err := mapEntityToResponse(job)
		if err != nil {
			s.logger.ErrorWithStack(
				ctx,
				"failed to map job to response",
				err,
				slog.String("job_id", job.ID),
				slog.String("automation_id", automationId),
			)
			return nil, 0, ErrListJobs
		}
		responses = append(responses, jobResponse)
	}

	return responses, total, nil
}

func (s service) GetById(ctx context.Context, jobId string, automationId string) (api.Job, error) {
	job, err := s.repository.FindById(ctx, jobId, automationId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.Job{}, ErrJobNotFound
		}

		s.logger.ErrorWithStack(
			ctx,
			"failed to find job by id",
			err,
			slog.String("job_id", jobId),
			slog.String("automation_id", automationId),
		)
		return api.Job{}, ErrGetJob
	}

	jobResponse, err := mapEntityToResponse(job)
	if err != nil {
		s.logger.ErrorWithStack(
			ctx,
			"failed to map job to response",
			err,
			slog.String("job_id", job.ID),
			slog.String("automation_id", automationId),
		)
		return api.Job{}, ErrGetJob
	}
	return jobResponse, nil
}

func (s service) Create(ctx context.Context, automationId string, req api.JobCreateRequest) (api.Job, error) {
	parsedAutomationID, err := uuid.Parse(automationId)
	if err != nil {
		return api.Job{}, ErrInvalidAutomationID
	}

	output := ""
	if req.Output != nil {
		output = *req.Output
	}
	if outputErr := validation.ValidateBase64Output(output, requestlimits.MaxDecodedOutputBytes(ctx)); outputErr != nil {
		return api.Job{}, NewInvalidOutputError(outputErr)
	}

	if _, err := s.automationLookupService.GetById(ctx, automationId); err != nil {
		if errors.Is(err, automation.ErrAutomationNotFound) {
			return api.Job{}, automation.ErrAutomationNotFound
		}

		s.logger.ErrorWithStack(
			ctx,
			"failed to find automation before creating job",
			err,
			slog.String("automation_id", automationId),
			slog.String("job_id", req.Id),
		)
		return api.Job{}, ErrCreateJob
	}

	metadata, err := marshalMetadata(req.Metadata)
	if err != nil {
		return api.Job{}, NewInvalidMetadataError(err)
	}

	job := Job{
		ID:           req.Id,
		AutomationID: parsedAutomationID,
		Status:       string(req.Status),
		Location:     req.Location,
		Url:          req.Url,
		Metadata:     metadata,
		Output:       output,
	}

	createdJob, err := s.repository.Create(ctx, job)
	if err != nil {
		s.logger.ErrorWithStack(
			ctx,
			"failed to create job",
			err,
			slog.String("job_id", req.Id),
			slog.String("automation_id", automationId),
		)
		return api.Job{}, ErrCreateJob
	}

	jobResponse, err := mapEntityToResponse(createdJob)
	if err != nil {
		s.logger.ErrorWithStack(
			ctx,
			"failed to map job to response",
			err,
			slog.String("job_id", createdJob.ID),
			slog.String("automation_id", automationId),
		)
		return api.Job{}, ErrCreateJob
	}
	return jobResponse, nil
}

func (s service) Update(ctx context.Context, automationId string, jobId string, req api.JobUpdateRequest) (api.Job, error) {
	parsedAutomationID, err := uuid.Parse(automationId)
	if err != nil {
		return api.Job{}, ErrInvalidAutomationID
	}

	output := ""
	if req.Output != nil {
		output = *req.Output
	}

	if outputErr := validation.ValidateBase64Output(output, requestlimits.MaxDecodedOutputBytes(ctx)); outputErr != nil {
		return api.Job{}, NewInvalidOutputError(outputErr)
	}

	metadata, err := marshalMetadata(req.Metadata)
	if err != nil {
		return api.Job{}, NewInvalidMetadataError(err)
	}

	job := Job{
		ID:           jobId,
		AutomationID: parsedAutomationID,
		Status:       string(req.Status),
		Metadata:     metadata,
		Output:       output,
	}

	updatedJob, err := s.repository.Update(ctx, job)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.Job{}, ErrJobNotFound
		}

		s.logger.ErrorWithStack(
			ctx,
			"failed to update job",
			err,
			slog.String("job_id", jobId),
			slog.String("automation_id", automationId),
		)
		return api.Job{}, ErrUpdateJob
	}

	jobResponse, err := mapEntityToResponse(updatedJob)
	if err != nil {
		s.logger.ErrorWithStack(
			ctx,
			"failed to map job to response",
			err,
			slog.String("job_id", updatedJob.ID),
			slog.String("automation_id", automationId),
		)
		return api.Job{}, ErrUpdateJob
	}
	return jobResponse, nil
}

func NewService(repository Repository,
	automationLookupService automation.LookupService,
	appLogger *logger.Logger) Service {
	if appLogger == nil {
		appLogger = logger.Default()
	}

	return &service{
		repository:              repository,
		automationLookupService: automationLookupService,
		logger:                  appLogger,
	}
}

func mapEntityToResponse(job Job) (api.Job, error) {
	var metadataPtr *api.JobMetadata
	if len(job.Metadata) > 0 && string(job.Metadata) != "null" {
		var metadata api.JobMetadata
		if err := json.Unmarshal(job.Metadata, &metadata); err != nil {
			return api.Job{}, err
		}
		metadataPtr = &metadata
	}

	var outputPtr *api.JobOutput
	if job.Output != "" {
		output := api.JobOutput(job.Output)
		outputPtr = &output
	}

	return api.Job{
		Id:         job.ID,
		Automation: job.Automation,
		Provider:   job.Provider,
		Status:     api.JobStatus(job.Status),
		Location:   job.Location,
		Url:        job.Url,
		Metadata:   metadataPtr,
		Output:     outputPtr,
	}, nil
}

func marshalMetadata(metadata *api.JobMetadata) ([]byte, error) {
	if metadata == nil {
		return []byte("null"), nil
	}

	return json.Marshal(metadata)
}

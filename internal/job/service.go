package job

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/automation"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"gorm.io/gorm"
)

type Service interface {
	GetAll(ctx context.Context, automationId string, page int, limit int) ([]GetResponse, int64, error)
	GetById(ctx context.Context, jobId string, automationId string) (GetResponse, error)
	Create(ctx context.Context, automationId string, req CreateRequest) (GetResponse, error)
	Update(ctx context.Context, automationId string, jobId string, req UpdateRequest) (GetResponse, error)
}

type service struct {
	repository              Repository
	automationLookupService automation.LookupService
	logger                  *logger.Logger
}

func (s service) GetAll(ctx context.Context, automationId string, page int, limit int) ([]GetResponse, int64, error) {
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

	responses := make([]GetResponse, 0, len(jobs))
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

func (s service) GetById(ctx context.Context, jobId string, automationId string) (GetResponse, error) {
	job, err := s.repository.FindById(ctx, jobId, automationId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return GetResponse{}, ErrJobNotFound
		}

		s.logger.ErrorWithStack(
			ctx,
			"failed to find job by id",
			err,
			slog.String("job_id", jobId),
			slog.String("automation_id", automationId),
		)
		return GetResponse{}, ErrGetJob
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
		return GetResponse{}, ErrGetJob
	}
	return jobResponse, nil
}

func (s service) Create(ctx context.Context, automationId string, req CreateRequest) (GetResponse, error) {
	parsedAutomationID, err := uuid.Parse(automationId)
	if err != nil {
		return GetResponse{}, ErrInvalidAutomationID
	}

	if _, err := s.automationLookupService.GetById(ctx, automationId); err != nil {
		s.logger.ErrorWithStack(
			ctx,
			"failed to find automation before creating job",
			err,
			slog.String("automation_id", automationId),
			slog.String("job_id", req.ID),
		)
		return GetResponse{}, ErrCreateJob
	}

	if !isValidBase64(req.Output) {
		return GetResponse{}, NewInvalidOutputError(errors.New("not valid encoding"))
	}

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		return GetResponse{}, NewInvalidOutputError(err)
	}

	job := Job{
		ID:           req.ID,
		AutomationID: parsedAutomationID,
		Status:       req.Status,
		Location:     req.Location,
		Url:          req.URL,
		Metadata:     metadata,
		Output:       req.Output,
	}

	createdJob, err := s.repository.Create(ctx, job)
	if err != nil {
		s.logger.ErrorWithStack(
			ctx,
			"failed to create job",
			err,
			slog.String("job_id", req.ID),
			slog.String("automation_id", automationId),
		)
		return GetResponse{}, ErrCreateJob
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
		return GetResponse{}, ErrCreateJob
	}
	return jobResponse, nil
}

func (s service) Update(ctx context.Context, automationId string, jobId string, req UpdateRequest) (GetResponse, error) {
	parsedAutomationID, err := uuid.Parse(automationId)
	if err != nil {
		return GetResponse{}, ErrInvalidAutomationID
	}

	if !isValidBase64(req.Output) {
		return GetResponse{}, NewInvalidOutputError(errors.New("not valid encoding"))
	}

	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		return GetResponse{}, NewInvalidOutputError(err)
	}

	job := Job{
		ID:           jobId,
		AutomationID: parsedAutomationID,
		Status:       req.Status,
		Metadata:     metadata,
		Output:       req.Output,
	}

	updatedJob, err := s.repository.Update(ctx, job)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return GetResponse{}, ErrJobNotFound
		}

		s.logger.ErrorWithStack(
			ctx,
			"failed to update job",
			err,
			slog.String("job_id", jobId),
			slog.String("automation_id", automationId),
		)
		return GetResponse{}, ErrUpdateJob
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
		return GetResponse{}, ErrUpdateJob
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

func mapEntityToResponse(job Job) (GetResponse, error) {
	metadata, err := json.Marshal(job.Metadata)
	if err != nil {
		return GetResponse{}, err
	}

	return GetResponse{
		ID:         job.ID,
		Automation: job.Automation,
		Provider:   job.Provider,
		Status:     job.Status,
		Location:   job.Location,
		URL:        job.Url,
		Metadata:   metadata,
		Output:     job.Output,
	}, nil
}

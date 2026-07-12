package job

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/svetlyopet/heimdallr/internal/modules/automation"
	"github.com/svetlyopet/heimdallr/internal/modules/job/api"
	"github.com/svetlyopet/heimdallr/internal/pagination"
	"github.com/svetlyopet/heimdallr/internal/requestlimits"
	"github.com/svetlyopet/heimdallr/internal/validation"
)

type Handler interface {
	api.StrictServerInterface
}

type handler struct {
	service Service
}

func (h handler) ListAutomationJobs(ctx context.Context, request api.ListAutomationJobsRequestObject) (api.ListAutomationJobsResponseObject, error) {
	page, limit, ok := paginationParams(request.Params.Page, request.Params.Limit)
	if !ok {
		return api.ListAutomationJobs400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "page and limit must be positive integers"},
		}, nil
	}

	jobs, total, err := h.service.GetAll(ctx, request.AutomationId.String(), page, limit)
	if err != nil {
		return api.ListAutomationJobs500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: jobErrorMessage(err, "failed to list jobs")},
		}, nil
	}

	return api.ListAutomationJobs200JSONResponse{
		Data:       jobs,
		Pagination: buildPagination(page, limit, total),
	}, nil
}

func (h handler) CreateAutomationJob(ctx context.Context, request api.CreateAutomationJobRequestObject) (api.CreateAutomationJobResponseObject, error) {
	if request.Body == nil {
		return api.CreateAutomationJob400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid request body"},
		}, nil
	}

	if request.Body.Metadata != nil {
		if _, err := json.Marshal(request.Body.Metadata); err != nil {
			return api.CreateAutomationJob400JSONResponse{
				BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid metadata"},
			}, nil
		}
	}

	if request.Body.Output != nil && validateOutput(ctx, *request.Body.Output) != nil {
		return api.CreateAutomationJob400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid output"},
		}, nil
	}

	job, err := h.service.Create(ctx, request.AutomationId.String(), *request.Body)
	if err != nil {
		if errors.Is(err, automation.ErrAutomationNotFound) {
			return api.CreateAutomationJob404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: jobErrorMessage(err, "automation not found")},
			}, nil
		}

		if errors.Is(err, ErrInvalidAutomationID) {
			return api.CreateAutomationJob400JSONResponse{
				BadRequestJSONResponse: api.BadRequestJSONResponse{Error: jobErrorMessage(err, "invalid automation id")},
			}, nil
		}

		if errors.Is(err, ErrJobNotFound) {
			return api.CreateAutomationJob404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: jobErrorMessage(err, "automation not found")},
			}, nil
		}

		if _, ok := errors.AsType[JobError](err); ok {
			return api.CreateAutomationJob400JSONResponse{
				BadRequestJSONResponse: api.BadRequestJSONResponse{Error: jobErrorMessage(err, "invalid request")},
			}, nil
		}

		return api.CreateAutomationJob500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: jobErrorMessage(err, "failed to create job")},
		}, nil
	}

	return api.CreateAutomationJob201JSONResponse{Data: job}, nil
}

func (h handler) GetAutomationJob(ctx context.Context, request api.GetAutomationJobRequestObject) (api.GetAutomationJobResponseObject, error) {
	job, err := h.service.GetById(ctx, request.JobId, request.AutomationId.String())
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			return api.GetAutomationJob404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: jobErrorMessage(err, "job not found")},
			}, nil
		}

		return api.GetAutomationJob500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: jobErrorMessage(err, "failed to get job")},
		}, nil
	}

	return api.GetAutomationJob200JSONResponse{Data: job}, nil
}

func (h handler) UpdateAutomationJob(ctx context.Context, request api.UpdateAutomationJobRequestObject) (api.UpdateAutomationJobResponseObject, error) {
	if request.Body == nil {
		return api.UpdateAutomationJob400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid request body"},
		}, nil
	}

	if request.Body.Metadata != nil {
		if _, err := json.Marshal(request.Body.Metadata); err != nil {
			return api.UpdateAutomationJob400JSONResponse{
				BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid metadata"},
			}, nil
		}
	}

	if request.Body.Output != nil && validateOutput(ctx, *request.Body.Output) != nil {
		return api.UpdateAutomationJob400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid output"},
		}, nil
	}

	job, err := h.service.Update(ctx, request.AutomationId.String(), request.JobId, *request.Body)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			return api.UpdateAutomationJob404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: jobErrorMessage(err, "job not found")},
			}, nil
		}

		if _, ok := errors.AsType[JobError](err); ok {
			return api.UpdateAutomationJob400JSONResponse{
				BadRequestJSONResponse: api.BadRequestJSONResponse{Error: jobErrorMessage(err, "invalid request")},
			}, nil
		}

		return api.UpdateAutomationJob500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: jobErrorMessage(err, "failed to update job")},
		}, nil
	}

	return api.UpdateAutomationJob200JSONResponse{Data: job}, nil
}

func NewHandler(service Service) (Handler, error) {
	return &handler{
		service: service,
	}, nil
}

func paginationParams(pagePtr, limitPtr *api.Page) (page int, limit int, ok bool) {
	page = 1
	limit = 10

	if pagePtr != nil {
		page = int(*pagePtr)
	}
	if limitPtr != nil {
		limit = int(*limitPtr)
	}

	return page, limit, page >= 1 && limit >= 1
}

func buildPagination(page, limit int, total int64) api.Pagination {
	safeTotal, totalPages := pagination.SafeTotals(total, limit)

	return api.Pagination{
		Page:       page,
		Limit:      limit,
		Total:      safeTotal,
		TotalPages: totalPages,
	}
}

func jobErrorMessage(err error, fallback string) string {
	if jobErr, ok := errors.AsType[JobError](err); ok {
		return jobErr.Message
	}

	return fallback
}

func validateOutput(ctx context.Context, value string) error {
	return validation.ValidateBase64Output(value, requestlimits.MaxDecodedOutputBytes(ctx))
}

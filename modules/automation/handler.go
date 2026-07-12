package automation

import (
	"context"
	"errors"

	"github.com/svetlyopet/heimdallr/internal/pagination"
	"github.com/svetlyopet/heimdallr/modules/automation/api"
)

type Handler interface {
	api.StrictServerInterface
}

type handler struct {
	service Service
}

func (h handler) ListAutomations(ctx context.Context, request api.ListAutomationsRequestObject) (api.ListAutomationsResponseObject, error) {
	page, limit, ok := paginationParams(request.Params.Page, request.Params.Limit)
	if !ok {
		return api.ListAutomations400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "page and limit must be positive integers"},
		}, nil
	}

	automations, total, err := h.service.GetAll(ctx, page, limit)
	if err != nil {
		return api.ListAutomations500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: automationErrorMessage(err, "failed to list automations")},
		}, nil
	}

	return api.ListAutomations200JSONResponse{
		Data:       automations,
		Pagination: buildPagination(page, limit, total),
	}, nil
}

func (h handler) CreateAutomation(ctx context.Context, request api.CreateAutomationRequestObject) (api.CreateAutomationResponseObject, error) {
	if request.Body == nil {
		return api.CreateAutomation400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid request body"},
		}, nil
	}

	automation, err := h.service.Create(ctx, *request.Body)
	if err != nil {
		if errors.Is(err, ErrAutomationAlreadyExists) {
			return api.CreateAutomation409JSONResponse{
				ConflictJSONResponse: api.ConflictJSONResponse{Error: automationErrorMessage(err, "automation already exists")},
			}, nil
		}

		return api.CreateAutomation500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: automationErrorMessage(err, "failed to create automation")},
		}, nil
	}

	return api.CreateAutomation201JSONResponse{Data: automation}, nil
}

func (h handler) GetAutomation(ctx context.Context, request api.GetAutomationRequestObject) (api.GetAutomationResponseObject, error) {
	automation, err := h.service.GetById(ctx, request.AutomationId.String())
	if err != nil {
		if errors.Is(err, ErrAutomationNotFound) {
			return api.GetAutomation404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: automationErrorMessage(err, "automation not found")},
			}, nil
		}

		return api.GetAutomation500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: automationErrorMessage(err, "failed to get automation")},
		}, nil
	}

	return api.GetAutomation200JSONResponse{Data: automation}, nil
}

func (h handler) UpdateAutomation(ctx context.Context, request api.UpdateAutomationRequestObject) (api.UpdateAutomationResponseObject, error) {
	if request.Body == nil {
		return api.UpdateAutomation400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid request body"},
		}, nil
	}

	automation, err := h.service.Update(ctx, *request.Body, request.AutomationId.String())
	if err != nil {
		if errors.Is(err, ErrAutomationNotFound) {
			return api.UpdateAutomation404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: automationErrorMessage(err, "automation not found")},
			}, nil
		}

		return api.UpdateAutomation500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: automationErrorMessage(err, "failed to update automation")},
		}, nil
	}

	return api.UpdateAutomation200JSONResponse{Data: automation}, nil
}

func (h handler) DeleteAutomation(ctx context.Context, request api.DeleteAutomationRequestObject) (api.DeleteAutomationResponseObject, error) {
	if err := h.service.Delete(ctx, request.AutomationId.String()); err != nil {
		if errors.Is(err, ErrAutomationNotFound) {
			return api.DeleteAutomation404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: automationErrorMessage(err, "automation not found")},
			}, nil
		}

		return api.DeleteAutomation500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: automationErrorMessage(err, "failed to delete automation")},
		}, nil
	}

	return api.DeleteAutomation204Response{}, nil
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

func automationErrorMessage(err error, fallback string) string {
	if automationErr, ok := errors.AsType[AutomationError](err); ok {
		return automationErr.Message
	}

	return fallback
}

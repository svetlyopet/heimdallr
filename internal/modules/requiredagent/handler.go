package requiredagent

import (
	"context"
	"errors"

	"github.com/svetlyopet/heimdallr/internal/modules/requiredagent/api"
	"github.com/svetlyopet/heimdallr/internal/pagination"
)

type Handler interface {
	api.StrictServerInterface
}

type handler struct {
	service Service
}

func (h handler) ListRequiredAgents(ctx context.Context, request api.ListRequiredAgentsRequestObject) (api.ListRequiredAgentsResponseObject, error) {
	page, limit, ok := paginationParams(request.Params.Page, request.Params.Limit)
	if !ok {
		return api.ListRequiredAgents400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "page and limit must be positive integers"},
		}, nil
	}

	requiredAgents, total, err := h.service.GetAll(ctx, page, limit)
	if err != nil {
		return api.ListRequiredAgents500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: requiredAgentErrorMessage(err, "failed to list required agents")},
		}, nil
	}

	return api.ListRequiredAgents200JSONResponse{
		Data:       requiredAgents,
		Pagination: buildPagination(page, limit, total),
	}, nil
}

func (h handler) CreateRequiredAgent(ctx context.Context, request api.CreateRequiredAgentRequestObject) (api.CreateRequiredAgentResponseObject, error) {
	if request.Body == nil {
		return api.CreateRequiredAgent400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid request body"},
		}, nil
	}

	requiredAgent, err := h.service.Create(ctx, *request.Body)
	if err != nil {
		if errors.Is(err, ErrRequiredAgentAlreadyExists) {
			return api.CreateRequiredAgent409JSONResponse{
				ConflictJSONResponse: api.ConflictJSONResponse{Error: requiredAgentErrorMessage(err, "required agent already exists")},
			}, nil
		}

		return api.CreateRequiredAgent500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: requiredAgentErrorMessage(err, "failed to create required agent")},
		}, nil
	}

	return api.CreateRequiredAgent201JSONResponse{Data: requiredAgent}, nil
}

func (h handler) GetRequiredAgent(ctx context.Context, request api.GetRequiredAgentRequestObject) (api.GetRequiredAgentResponseObject, error) {
	requiredAgent, err := h.service.GetById(ctx, request.RequiredAgentId.String())
	if err != nil {
		if errors.Is(err, ErrRequiredAgentNotFound) {
			return api.GetRequiredAgent404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: requiredAgentErrorMessage(err, "required agent not found")},
			}, nil
		}

		return api.GetRequiredAgent500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: requiredAgentErrorMessage(err, "failed to get required agent")},
		}, nil
	}

	return api.GetRequiredAgent200JSONResponse{Data: requiredAgent}, nil
}

func (h handler) UpdateRequiredAgent(ctx context.Context, request api.UpdateRequiredAgentRequestObject) (api.UpdateRequiredAgentResponseObject, error) {
	if request.Body == nil {
		return api.UpdateRequiredAgent400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid request body"},
		}, nil
	}

	requiredAgent, err := h.service.Update(ctx, *request.Body, request.RequiredAgentId.String())
	if err != nil {
		if errors.Is(err, ErrRequiredAgentNotFound) {
			return api.UpdateRequiredAgent404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: requiredAgentErrorMessage(err, "required agent not found")},
			}, nil
		}

		if errors.Is(err, ErrRequiredAgentAlreadyExists) {
			return api.UpdateRequiredAgent409JSONResponse{
				ConflictJSONResponse: api.ConflictJSONResponse{Error: requiredAgentErrorMessage(err, "required agent already exists")},
			}, nil
		}

		return api.UpdateRequiredAgent500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: requiredAgentErrorMessage(err, "failed to update required agent")},
		}, nil
	}

	return api.UpdateRequiredAgent200JSONResponse{Data: requiredAgent}, nil
}

func (h handler) DeleteRequiredAgent(ctx context.Context, request api.DeleteRequiredAgentRequestObject) (api.DeleteRequiredAgentResponseObject, error) {
	if err := h.service.Delete(ctx, request.RequiredAgentId.String()); err != nil {
		if errors.Is(err, ErrRequiredAgentNotFound) {
			return api.DeleteRequiredAgent404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: requiredAgentErrorMessage(err, "required agent not found")},
			}, nil
		}

		return api.DeleteRequiredAgent500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: requiredAgentErrorMessage(err, "failed to delete required agent")},
		}, nil
	}

	return api.DeleteRequiredAgent204Response{}, nil
}

func NewHandler(service Service) (Handler, error) {
	if service == nil {
		return nil, errors.New("required agent service is required")
	}

	return &handler{service: service}, nil
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

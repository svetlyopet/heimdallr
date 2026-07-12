package agent

import (
	"context"
	"errors"
	"strings"

	"github.com/svetlyopet/heimdallr/internal/modules/agent/api"
	"github.com/svetlyopet/heimdallr/internal/modules/server"
	"github.com/svetlyopet/heimdallr/internal/pagination"
)

type Handler interface {
	api.StrictServerInterface
}

type handler struct {
	service Service
}

func (h handler) ListAgents(ctx context.Context, request api.ListAgentsRequestObject) (api.ListAgentsResponseObject, error) {
	page, limit, ok := paginationParams(request.Params.Page, request.Params.Limit)
	if !ok {
		return api.ListAgents400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "page and limit must be positive integers"},
		}, nil
	}

	agents, total, err := h.service.GetAll(ctx, request.ServerId.String(), page, limit)
	if err != nil {
		if errors.Is(err, server.ErrServerNotFound) {
			return api.ListAgents404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: agentErrorMessage(err, server.ErrServerNotFound.Error())},
			}, nil
		}

		return api.ListAgents500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: agentErrorMessage(err, "failed to list agents")},
		}, nil
	}

	return api.ListAgents200JSONResponse{
		Data:       agents,
		Pagination: buildPagination(page, limit, total),
	}, nil
}

func (h handler) GetAgent(ctx context.Context, request api.GetAgentRequestObject) (api.GetAgentResponseObject, error) {
	agent, err := h.service.GetById(ctx, request.AgentId.String(), request.ServerId.String())
	if err != nil {
		if errors.Is(err, ErrAgentNotFound) {
			return api.GetAgent404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: agentErrorMessage(err, "agent not found")},
			}, nil
		}

		return api.GetAgent500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: agentErrorMessage(err, "failed to get agent")},
		}, nil
	}

	return api.GetAgent200JSONResponse{Data: agent}, nil
}

func (h handler) CreateAgent(ctx context.Context, request api.CreateAgentRequestObject) (api.CreateAgentResponseObject, error) {
	if request.Body == nil {
		return api.CreateAgent400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid request body"},
		}, nil
	}

	if request.Body.AgentId == nil && strings.TrimSpace(stringValue(request.Body.Name)) == "" {
		return api.CreateAgent400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "agent_id or name is required"},
		}, nil
	}

	agent, err := h.service.CreateOnServer(ctx, request.ServerId.String(), *request.Body)
	if err != nil {
		switch {
		case errors.Is(err, server.ErrServerNotFound), errors.Is(err, ErrAgentNotFound):
			return api.CreateAgent404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: agentErrorMessage(err, err.Error())},
			}, nil
		case errors.Is(err, ErrAgentAlreadyLinked), errors.Is(err, ErrAgentAlreadyExists):
			return api.CreateAgent409JSONResponse{
				ConflictJSONResponse: api.ConflictJSONResponse{Error: agentErrorMessage(err, err.Error())},
			}, nil
		}

		return api.CreateAgent500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: agentErrorMessage(err, "failed to create agent")},
		}, nil
	}

	return api.CreateAgent201JSONResponse{Data: agent}, nil
}

func (h handler) DetachAgent(ctx context.Context, request api.DetachAgentRequestObject) (api.DetachAgentResponseObject, error) {
	if err := h.service.Detach(ctx, request.ServerId.String(), request.AgentId.String()); err != nil {
		switch {
		case errors.Is(err, server.ErrServerNotFound), errors.Is(err, ErrAgentNotFound):
			return api.DetachAgent404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: agentErrorMessage(err, err.Error())},
			}, nil
		}

		return api.DetachAgent500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: agentErrorMessage(err, "failed to detach agent")},
		}, nil
	}

	return api.DetachAgent204Response{}, nil
}

func (h handler) ListGlobalAgents(ctx context.Context, request api.ListGlobalAgentsRequestObject) (api.ListGlobalAgentsResponseObject, error) {
	page, limit, ok := paginationParams(request.Params.Page, request.Params.Limit)
	if !ok {
		return api.ListGlobalAgents400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "page and limit must be positive integers"},
		}, nil
	}

	filter := ListFilters{
		UnassignedOnly: request.Params.Unassigned != nil && *request.Params.Unassigned,
	}
	if request.Params.ServerId != nil {
		filter.ServerID = request.Params.ServerId.String()
	}
	if request.Params.AgentId != nil {
		filter.AgentID = request.Params.AgentId.String()
	}

	agents, total, err := h.service.ListGlobal(ctx, filter, page, limit)
	if err != nil {
		return api.ListGlobalAgents500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: agentErrorMessage(err, "failed to list agents")},
		}, nil
	}

	return api.ListGlobalAgents200JSONResponse{
		Data:       agents,
		Pagination: buildPagination(page, limit, total),
	}, nil
}

func (h handler) GetGlobalAgent(ctx context.Context, request api.GetGlobalAgentRequestObject) (api.GetGlobalAgentResponseObject, error) {
	agent, err := h.service.GetByIdGlobal(ctx, request.AgentId.String())
	if err != nil {
		if errors.Is(err, ErrAgentNotFound) {
			return api.GetGlobalAgent404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: agentErrorMessage(err, "agent not found")},
			}, nil
		}

		return api.GetGlobalAgent500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: agentErrorMessage(err, "failed to get agent")},
		}, nil
	}

	return api.GetGlobalAgent200JSONResponse{Data: agent}, nil
}

func (h handler) CreateUnassignedAgent(ctx context.Context, request api.CreateUnassignedAgentRequestObject) (api.CreateUnassignedAgentResponseObject, error) {
	if request.Body == nil || request.Body.Name == "" {
		return api.CreateUnassignedAgent400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid request body"},
		}, nil
	}

	agent, err := h.service.CreateUnassigned(ctx, *request.Body)
	if err != nil {
		if errors.Is(err, ErrAgentAlreadyExists) {
			return api.CreateUnassignedAgent409JSONResponse{
				ConflictJSONResponse: api.ConflictJSONResponse{Error: agentErrorMessage(err, "agent already exists")},
			}, nil
		}

		return api.CreateUnassignedAgent500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: agentErrorMessage(err, "failed to create agent")},
		}, nil
	}

	return api.CreateUnassignedAgent201JSONResponse{Data: agent}, nil
}

func (h handler) DeleteGlobalAgent(ctx context.Context, request api.DeleteGlobalAgentRequestObject) (api.DeleteGlobalAgentResponseObject, error) {
	if err := h.service.DeleteGlobal(ctx, request.AgentId.String()); err != nil {
		if errors.Is(err, ErrAgentNotFound) {
			return api.DeleteGlobalAgent404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: agentErrorMessage(err, "agent not found")},
			}, nil
		}

		return api.DeleteGlobalAgent500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: agentErrorMessage(err, "failed to delete agent")},
		}, nil
	}

	return api.DeleteGlobalAgent204Response{}, nil
}

func (h handler) ListAgentServers(ctx context.Context, request api.ListAgentServersRequestObject) (api.ListAgentServersResponseObject, error) {
	page, limit, ok := paginationParams(request.Params.Page, request.Params.Limit)
	if !ok {
		return api.ListAgentServers400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "page and limit must be positive integers"},
		}, nil
	}

	servers, total, err := h.service.ListServers(ctx, request.AgentId.String(), page, limit)
	if err != nil {
		if errors.Is(err, ErrAgentNotFound) {
			return api.ListAgentServers404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: agentErrorMessage(err, "agent not found")},
			}, nil
		}

		return api.ListAgentServers500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: agentErrorMessage(err, "failed to list agent servers")},
		}, nil
	}

	return api.ListAgentServers200JSONResponse{
		Data:       servers,
		Pagination: buildPagination(page, limit, total),
	}, nil
}

func NewHandler(service Service) (Handler, error) {
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

func agentErrorMessage(err error, fallback string) string {
	if agentErr, ok := errors.AsType[AgentError](err); ok {
		return agentErr.Message
	}

	return fallback
}

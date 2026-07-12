package server

import (
	"context"
	"errors"

	"github.com/svetlyopet/heimdallr/internal/pagination"
	"github.com/svetlyopet/heimdallr/internal/server/api"
)

type Handler interface {
	api.StrictServerInterface
}

type handler struct {
	service Service
}

func (h handler) ListServers(ctx context.Context, request api.ListServersRequestObject) (api.ListServersResponseObject, error) {
	page, limit, ok := paginationParams(request.Params.Page, request.Params.Limit)
	if !ok {
		return api.ListServers400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "page and limit must be positive integers"},
		}, nil
	}

	agentID := ""
	if request.Params.AgentId != nil {
		agentID = request.Params.AgentId.String()
	}

	servers, total, err := h.service.GetAll(ctx, agentID, page, limit)
	if err != nil {
		return api.ListServers500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: serverErrorMessage(err, "failed to list servers")},
		}, nil
	}

	return api.ListServers200JSONResponse{
		Data:       servers,
		Pagination: buildPagination(page, limit, total),
	}, nil
}

func (h handler) CreateServer(ctx context.Context, request api.CreateServerRequestObject) (api.CreateServerResponseObject, error) {
	if request.Body == nil {
		return api.CreateServer400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid request body"},
		}, nil
	}
	if request.Body.Hostname == "" {
		return api.CreateServer400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid request body"},
		}, nil
	}

	server, err := h.service.Create(ctx, *request.Body)
	if err != nil {
		if errors.Is(err, ErrServerAlreadyExists) {
			return api.CreateServer409JSONResponse{
				ConflictJSONResponse: api.ConflictJSONResponse{Error: serverErrorMessage(err, "server already exists")},
			}, nil
		}
		if errors.Is(err, ErrAgentAlreadyLinked) || errors.Is(err, ErrAgentAlreadyExists) {
			return api.CreateServer409JSONResponse{
				ConflictJSONResponse: api.ConflictJSONResponse{Error: serverErrorMessage(err, err.Error())},
			}, nil
		}
		if errors.Is(err, ErrDuplicateAgentIDs) || errors.Is(err, ErrDuplicateAgentNames) {
			return api.CreateServer400JSONResponse{
				BadRequestJSONResponse: api.BadRequestJSONResponse{Error: serverErrorMessage(err, err.Error())},
			}, nil
		}

		return api.CreateServer500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: serverErrorMessage(err, "failed to create server")},
		}, nil
	}

	return api.CreateServer201JSONResponse{Data: server}, nil
}

func (h handler) GetServer(ctx context.Context, request api.GetServerRequestObject) (api.GetServerResponseObject, error) {
	server, err := h.service.GetById(ctx, request.ServerId.String())
	if err != nil {
		if errors.Is(err, ErrServerNotFound) {
			return api.GetServer404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: serverErrorMessage(err, "server not found")},
			}, nil
		}

		return api.GetServer500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: serverErrorMessage(err, "failed to get server")},
		}, nil
	}

	return api.GetServer200JSONResponse{Data: server}, nil
}

func (h handler) UpdateServer(ctx context.Context, request api.UpdateServerRequestObject) (api.UpdateServerResponseObject, error) {
	if request.Body == nil {
		return api.UpdateServer400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid request body"},
		}, nil
	}

	server, err := h.service.Update(ctx, request.ServerId.String(), *request.Body)
	if err != nil {
		switch {
		case errors.Is(err, ErrServerNotFound):
			return api.UpdateServer404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: serverErrorMessage(err, "server not found")},
			}, nil
		case errors.Is(err, ErrAgentAlreadyLinked), errors.Is(err, ErrAgentAlreadyExists):
			return api.UpdateServer409JSONResponse{
				ConflictJSONResponse: api.ConflictJSONResponse{Error: serverErrorMessage(err, err.Error())},
			}, nil
		case errors.Is(err, ErrDuplicateAgentIDs), errors.Is(err, ErrDuplicateAgentNames):
			return api.UpdateServer400JSONResponse{
				BadRequestJSONResponse: api.BadRequestJSONResponse{Error: serverErrorMessage(err, err.Error())},
			}, nil
		}

		return api.UpdateServer500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: serverErrorMessage(err, "failed to update server")},
		}, nil
	}

	return api.UpdateServer200JSONResponse{Data: server}, nil
}

func (h handler) ListServerJobs(ctx context.Context, request api.ListServerJobsRequestObject) (api.ListServerJobsResponseObject, error) {
	page, limit, ok := paginationParams(request.Params.Page, request.Params.Limit)
	if !ok {
		return api.ListServerJobs400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "page and limit must be positive integers"},
		}, nil
	}

	jobs, total, err := h.service.ListJobs(ctx, request.ServerId.String(), page, limit)
	if err != nil {
		if errors.Is(err, ErrServerNotFound) {
			return api.ListServerJobs404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: serverErrorMessage(err, "server not found")},
			}, nil
		}

		return api.ListServerJobs500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: serverErrorMessage(err, "failed to list jobs")},
		}, nil
	}

	return api.ListServerJobs200JSONResponse{
		Data:       jobs,
		Pagination: buildPagination(page, limit, total),
	}, nil
}

func (h handler) AssociateServerJob(ctx context.Context, request api.AssociateServerJobRequestObject) (api.AssociateServerJobResponseObject, error) {
	if request.Body == nil {
		return api.AssociateServerJob400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid request body"},
		}, nil
	}

	if err := h.service.AssociateJob(ctx, request.ServerId.String(), *request.Body); err != nil {
		switch {
		case errors.Is(err, ErrServerNotFound), errors.Is(err, ErrJobNotFound):
			return api.AssociateServerJob404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: serverErrorMessage(err, err.Error())},
			}, nil
		case errors.Is(err, ErrJobAlreadyAssociated):
			return api.AssociateServerJob409JSONResponse{
				ConflictJSONResponse: api.ConflictJSONResponse{Error: serverErrorMessage(err, "job already associated with server")},
			}, nil
		}

		return api.AssociateServerJob500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: serverErrorMessage(err, "failed to associate job")},
		}, nil
	}

	return api.AssociateServerJob201Response{}, nil
}

func (h handler) DissociateServerJob(ctx context.Context, request api.DissociateServerJobRequestObject) (api.DissociateServerJobResponseObject, error) {
	if request.Params.AutomationId == (api.UUID{}) {
		return api.DissociateServerJob400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "automation_id query param is required"},
		}, nil
	}

	if err := h.service.DissociateJob(ctx, request.ServerId.String(), request.JobId, request.Params.AutomationId); err != nil {
		switch {
		case errors.Is(err, ErrServerNotFound), errors.Is(err, ErrJobNotFound):
			return api.DissociateServerJob404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: serverErrorMessage(err, err.Error())},
			}, nil
		}

		return api.DissociateServerJob500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: serverErrorMessage(err, "failed to dissociate job")},
		}, nil
	}

	return api.DissociateServerJob204Response{}, nil
}

func (h handler) ListServerReleases(ctx context.Context, request api.ListServerReleasesRequestObject) (api.ListServerReleasesResponseObject, error) {
	page, limit, ok := paginationParams(request.Params.Page, request.Params.Limit)
	if !ok {
		return api.ListServerReleases400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "page and limit must be positive integers"},
		}, nil
	}

	releases, total, err := h.service.ListReleases(ctx, request.ServerId.String(), page, limit)
	if err != nil {
		if errors.Is(err, ErrServerNotFound) {
			return api.ListServerReleases404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: serverErrorMessage(err, "server not found")},
			}, nil
		}

		return api.ListServerReleases500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: serverErrorMessage(err, "failed to list releases")},
		}, nil
	}

	return api.ListServerReleases200JSONResponse{
		Data:       releases,
		Pagination: buildPagination(page, limit, total),
	}, nil
}

func (h handler) AssociateServerRelease(ctx context.Context, request api.AssociateServerReleaseRequestObject) (api.AssociateServerReleaseResponseObject, error) {
	if request.Body == nil {
		return api.AssociateServerRelease400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid request body"},
		}, nil
	}

	if err := h.service.AssociateRelease(ctx, request.ServerId.String(), *request.Body); err != nil {
		switch {
		case errors.Is(err, ErrServerNotFound), errors.Is(err, ErrReleaseNotFound):
			return api.AssociateServerRelease404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: serverErrorMessage(err, err.Error())},
			}, nil
		case errors.Is(err, ErrReleaseAlreadyAssociated):
			return api.AssociateServerRelease409JSONResponse{
				ConflictJSONResponse: api.ConflictJSONResponse{Error: serverErrorMessage(err, "release already associated with server")},
			}, nil
		}

		return api.AssociateServerRelease500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: serverErrorMessage(err, "failed to associate release")},
		}, nil
	}

	return api.AssociateServerRelease201Response{}, nil
}

func (h handler) DissociateServerRelease(ctx context.Context, request api.DissociateServerReleaseRequestObject) (api.DissociateServerReleaseResponseObject, error) {
	if err := h.service.DissociateRelease(ctx, request.ServerId.String(), request.ReleaseId); err != nil {
		switch {
		case errors.Is(err, ErrServerNotFound), errors.Is(err, ErrReleaseNotFound):
			return api.DissociateServerRelease404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: serverErrorMessage(err, err.Error())},
			}, nil
		}

		return api.DissociateServerRelease500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: serverErrorMessage(err, "failed to dissociate release")},
		}, nil
	}

	return api.DissociateServerRelease204Response{}, nil
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

func serverErrorMessage(err error, fallback string) string {
	if serverErr, ok := errors.AsType[ServerError](err); ok {
		return serverErr.Message
	}

	return fallback
}

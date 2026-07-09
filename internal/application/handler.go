package application

import (
	"context"
	"errors"

	"github.com/svetlyopet/heimdallr/internal/application/api"
)

type Handler interface {
	api.StrictServerInterface
}

type handler struct {
	service Service
}

func (h handler) ListApplications(ctx context.Context, request api.ListApplicationsRequestObject) (api.ListApplicationsResponseObject, error) {
	page, limit, ok := paginationParams(request.Params.Page, request.Params.Limit)
	if !ok {
		return api.ListApplications400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "page and limit must be positive integers"},
		}, nil
	}

	applications, total, err := h.service.GetAll(ctx, page, limit)
	if err != nil {
		return api.ListApplications500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: applicationErrorMessage(err, "failed to list applications")},
		}, nil
	}

	return api.ListApplications200JSONResponse{
		Data:       applications,
		Pagination: buildPagination(page, limit, total),
	}, nil
}

func (h handler) CreateApplication(ctx context.Context, request api.CreateApplicationRequestObject) (api.CreateApplicationResponseObject, error) {
	if request.Body == nil {
		return api.CreateApplication400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid request body"},
		}, nil
	}

	application, err := h.service.Create(ctx, *request.Body)
	if err != nil {
		if errors.Is(err, ErrApplicationAlreadyExists) {
			return api.CreateApplication409JSONResponse{
				ConflictJSONResponse: api.ConflictJSONResponse{Error: applicationErrorMessage(err, "application already exists")},
			}, nil
		}

		return api.CreateApplication500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: applicationErrorMessage(err, "failed to create application")},
		}, nil
	}

	return api.CreateApplication201JSONResponse{Data: application}, nil
}

func (h handler) GetApplication(ctx context.Context, request api.GetApplicationRequestObject) (api.GetApplicationResponseObject, error) {
	application, err := h.service.GetById(ctx, request.ApplicationId.String())
	if err != nil {
		if errors.Is(err, ErrApplicationNotFound) {
			return api.GetApplication404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: applicationErrorMessage(err, "application not found")},
			}, nil
		}

		return api.GetApplication500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: applicationErrorMessage(err, "failed to get application")},
		}, nil
	}

	return api.GetApplication200JSONResponse{Data: application}, nil
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
	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(limit) - 1) / int64(limit)
	}

	return api.Pagination{
		Page:       page,
		Limit:      limit,
		Total:      int(total),
		TotalPages: int(totalPages),
	}
}

func applicationErrorMessage(err error, fallback string) string {
	if applicationErr, ok := errors.AsType[ApplicationError](err); ok {
		return applicationErr.Message
	}

	return fallback
}

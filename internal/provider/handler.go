package provider

import (
	"context"
	"errors"

	"github.com/svetlyopet/heimdallr/internal/provider/api"
)

type Handler interface {
	api.StrictServerInterface
}

type handler struct {
	service Service
}

func (h handler) ListProviders(ctx context.Context, request api.ListProvidersRequestObject) (api.ListProvidersResponseObject, error) {
	page, limit, ok := paginationParams(request.Params.Page, request.Params.Limit)
	if !ok {
		return api.ListProviders400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "page and limit must be positive integers"},
		}, nil
	}

	providers, total, err := h.service.GetAll(ctx, page, limit)
	if err != nil {
		return api.ListProviders500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: providerErrorMessage(err, "failed to list providers")},
		}, nil
	}

	return api.ListProviders200JSONResponse{
		Data:       providers,
		Pagination: buildPagination(page, limit, total),
	}, nil
}

func (h handler) CreateProvider(ctx context.Context, request api.CreateProviderRequestObject) (api.CreateProviderResponseObject, error) {
	if request.Body == nil {
		return api.CreateProvider400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid request body"},
		}, nil
	}

	provider, err := h.service.Create(ctx, *request.Body)
	if err != nil {
		if errors.Is(err, ErrProviderAlreadyExists) {
			return api.CreateProvider409JSONResponse{
				ConflictJSONResponse: api.ConflictJSONResponse{Error: providerErrorMessage(err, "provider already exists")},
			}, nil
		}

		return api.CreateProvider500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: providerErrorMessage(err, "failed to create provider")},
		}, nil
	}

	return api.CreateProvider201JSONResponse{Data: provider}, nil
}

func (h handler) GetProvider(ctx context.Context, request api.GetProviderRequestObject) (api.GetProviderResponseObject, error) {
	provider, err := h.service.GetById(ctx, request.ProviderId.String())
	if err != nil {
		if errors.Is(err, ErrProviderNotFound) {
			return api.GetProvider404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: providerErrorMessage(err, "provider not found")},
			}, nil
		}

		return api.GetProvider500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: providerErrorMessage(err, "failed to get provider")},
		}, nil
	}

	return api.GetProvider200JSONResponse{Data: provider}, nil
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

func providerErrorMessage(err error, fallback string) string {
	if providerErr, ok := errors.AsType[ProviderError](err); ok {
		return providerErr.Message
	}

	return fallback
}

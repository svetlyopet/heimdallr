package release

import (
	"context"
	"errors"

	"github.com/svetlyopet/heimdallr/internal/application"
	"github.com/svetlyopet/heimdallr/internal/pagination"
	"github.com/svetlyopet/heimdallr/internal/release/api"
)

type Handler interface {
	api.StrictServerInterface
}

type handler struct {
	service Service
}

func (h handler) ListReleases(ctx context.Context, request api.ListReleasesRequestObject) (api.ListReleasesResponseObject, error) {
	page, limit, ok := paginationParams(request.Params.Page, request.Params.Limit)
	if !ok {
		return api.ListReleases400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "page and limit must be positive integers"},
		}, nil
	}

	releases, total, err := h.service.GetAll(ctx, request.ApplicationId.String(), page, limit)
	if err != nil {
		if errors.Is(err, application.ErrApplicationNotFound) {
			return api.ListReleases404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: releaseErrorMessage(err, "application not found")},
			}, nil
		}

		return api.ListReleases500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: releaseErrorMessage(err, "failed to list releases")},
		}, nil
	}

	return api.ListReleases200JSONResponse{
		Data:       releases,
		Pagination: buildPagination(page, limit, total),
	}, nil
}

func (h handler) CreateRelease(ctx context.Context, request api.CreateReleaseRequestObject) (api.CreateReleaseResponseObject, error) {
	if request.Body == nil {
		return api.CreateRelease400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid request body"},
		}, nil
	}

	upsert := false
	if request.Params.Upsert != nil {
		upsert = *request.Params.Upsert
	}

	release, err := h.service.Create(ctx, request.ApplicationId.String(), *request.Body, upsert)
	if err != nil {
		switch {
		case errors.Is(err, application.ErrApplicationNotFound):
			return api.CreateRelease404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: releaseErrorMessage(err, "application not found")},
			}, nil
		case errors.Is(err, ErrReleaseAlreadyExists):
			return api.CreateRelease409JSONResponse{
				ConflictJSONResponse: api.ConflictJSONResponse{Error: releaseErrorMessage(err, "release already exists")},
			}, nil
		}

		return api.CreateRelease500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: releaseErrorMessage(err, "failed to create release")},
		}, nil
	}

	return api.CreateRelease201JSONResponse{Data: release}, nil
}

func (h handler) GetRelease(ctx context.Context, request api.GetReleaseRequestObject) (api.GetReleaseResponseObject, error) {
	release, err := h.service.GetById(ctx, request.ReleaseId.String(), request.ApplicationId.String())
	if err != nil {
		if errors.Is(err, ErrReleaseNotFound) {
			return api.GetRelease404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: releaseErrorMessage(err, "release not found")},
			}, nil
		}

		return api.GetRelease500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: releaseErrorMessage(err, "failed to get release")},
		}, nil
	}

	return api.GetRelease200JSONResponse{Data: release}, nil
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

func releaseErrorMessage(err error, fallback string) string {
	if releaseErr, ok := errors.AsType[ReleaseError](err); ok {
		return releaseErr.Message
	}

	return fallback
}

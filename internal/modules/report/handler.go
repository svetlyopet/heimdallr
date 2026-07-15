package report

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/svetlyopet/heimdallr/internal/modules/report/api"
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

func (h handler) ListReleaseReports(ctx context.Context, request api.ListReleaseReportsRequestObject) (api.ListReleaseReportsResponseObject, error) {
	page, limit, ok := paginationParams(request.Params.Page, request.Params.Limit)
	if !ok {
		return api.ListReleaseReports400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "page and limit must be positive integers"},
		}, nil
	}

	reports, total, err := h.service.GetAll(ctx, request.ApplicationId.String(), request.ReleaseId.String(), page, limit)
	if err != nil {
		if errors.Is(err, ErrReportNotFound) {
			return api.ListReleaseReports404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: reportErrorMessage(err, "release not found")},
			}, nil
		}

		return api.ListReleaseReports500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: reportErrorMessage(err, "failed to list reports")},
		}, nil
	}

	return api.ListReleaseReports200JSONResponse{
		Data:       reports,
		Pagination: buildPagination(page, limit, total),
	}, nil
}

func (h handler) CreateReleaseReport(ctx context.Context, request api.CreateReleaseReportRequestObject) (api.CreateReleaseReportResponseObject, error) {
	if request.Body == nil {
		return api.CreateReleaseReport400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid request body"},
		}, nil
	}

	if request.Body.Metadata != nil {
		if _, err := json.Marshal(request.Body.Metadata); err != nil {
			return api.CreateReleaseReport400JSONResponse{
				BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid metadata"},
			}, nil
		}
	}

	if request.Body.Output != nil && validateOutput(ctx, *request.Body.Output) != nil {
		return api.CreateReleaseReport400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "invalid output"},
		}, nil
	}

	report, err := h.service.Create(ctx, request.ApplicationId.String(), request.ReleaseId.String(), *request.Body)
	if err != nil {
		if errors.Is(err, ErrReportNotFound) {
			return api.CreateReleaseReport404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: reportErrorMessage(err, "release not found")},
			}, nil
		}

		if _, ok := errors.AsType[ReportError](err); ok {
			return api.CreateReleaseReport400JSONResponse{
				BadRequestJSONResponse: api.BadRequestJSONResponse{Error: reportErrorMessage(err, "invalid request")},
			}, nil
		}

		return api.CreateReleaseReport500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: reportErrorMessage(err, "failed to create report")},
		}, nil
	}

	return api.CreateReleaseReport201JSONResponse{Data: report}, nil
}

func (h handler) GetReleaseReport(ctx context.Context, request api.GetReleaseReportRequestObject) (api.GetReleaseReportResponseObject, error) {
	report, err := h.service.GetById(ctx, request.ApplicationId.String(), request.ReleaseId.String(), request.ReportId)
	if err != nil {
		if errors.Is(err, ErrReportNotFound) {
			return api.GetReleaseReport404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: reportErrorMessage(err, "report not found")},
			}, nil
		}

		return api.GetReleaseReport500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: reportErrorMessage(err, "failed to get report")},
		}, nil
	}

	return api.GetReleaseReport200JSONResponse{Data: report}, nil
}

func (h handler) ListReportsGlobal(ctx context.Context, request api.ListReportsGlobalRequestObject) (api.ListReportsGlobalResponseObject, error) {
	page, limit, ok := paginationParams(request.Params.Page, request.Params.Limit)
	if !ok {
		return api.ListReportsGlobal400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "page and limit must be positive integers"},
		}, nil
	}

	if request.Params.Status != nil && !request.Params.Status.Valid() {
		return api.ListReportsGlobal400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "status must be one of skipped, success, failed"},
		}, nil
	}

	if request.Params.Type != nil && !request.Params.Type.Valid() {
		return api.ListReportsGlobal400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "type must be one of sast, dast, sbom, code_coverage, custom"},
		}, nil
	}

	filters := ListFilters{}
	if request.Params.ApplicationId != nil {
		filters.ApplicationID = request.Params.ApplicationId.String()
	}
	if request.Params.ReleaseId != nil {
		filters.ReleaseID = request.Params.ReleaseId.String()
	}
	if request.Params.Status != nil {
		filters.Status = string(*request.Params.Status)
	}
	if request.Params.Type != nil {
		filters.Type = string(*request.Params.Type)
	}

	reports, total, err := h.service.GetAllGlobal(ctx, filters, page, limit)
	if err != nil {
		if errors.Is(err, ErrInvalidApplicationID) || errors.Is(err, ErrInvalidReleaseID) {
			return api.ListReportsGlobal400JSONResponse{
				BadRequestJSONResponse: api.BadRequestJSONResponse{Error: reportErrorMessage(err, "invalid query param value")},
			}, nil
		}

		return api.ListReportsGlobal500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: reportErrorMessage(err, "failed to list reports")},
		}, nil
	}

	return api.ListReportsGlobal200JSONResponse{
		Data:       reports,
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

func reportErrorMessage(err error, fallback string) string {
	if reportErr, ok := errors.AsType[ReportError](err); ok {
		return reportErr.Message
	}

	return fallback
}

func validateOutput(ctx context.Context, value string) error {
	return validation.ValidateBase64Output(value, requestlimits.MaxDecodedOutputBytes(ctx))
}

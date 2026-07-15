package analytics

import (
	"context"
	"errors"

	"github.com/svetlyopet/heimdallr/internal/modules/analytics/api"
)

type Handler interface {
	api.StrictServerInterface
}

type handler struct {
	service Service
}

func (h handler) GetAutomationAnalyticsOverview(ctx context.Context, _ api.GetAutomationAnalyticsOverviewRequestObject) (api.GetAutomationAnalyticsOverviewResponseObject, error) {
	response, err := h.service.GetAutomationOverview(ctx)
	if err != nil {
		return api.GetAutomationAnalyticsOverview500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: analyticsErrorMessage(err, "failed to get automation analytics")},
		}, nil
	}

	return api.GetAutomationAnalyticsOverview200JSONResponse{Data: response}, nil
}

func (h handler) GetAutomationAnalyticsOverviewByID(ctx context.Context, request api.GetAutomationAnalyticsOverviewByIDRequestObject) (api.GetAutomationAnalyticsOverviewByIDResponseObject, error) {
	response, err := h.service.GetAutomationOverviewByID(ctx, request.AutomationId.String())
	if err != nil {
		if errors.Is(err, ErrAutomationNotFound) {
			return api.GetAutomationAnalyticsOverviewByID404JSONResponse{
				NotFoundJSONResponse: api.NotFoundJSONResponse{Error: analyticsErrorMessage(err, "automation not found")},
			}, nil
		}

		return api.GetAutomationAnalyticsOverviewByID500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: analyticsErrorMessage(err, "failed to get automation analytics")},
		}, nil
	}

	return api.GetAutomationAnalyticsOverviewByID200JSONResponse{Data: response}, nil
}

func (h handler) GetComplianceAnalyticsOverview(ctx context.Context, _ api.GetComplianceAnalyticsOverviewRequestObject) (api.GetComplianceAnalyticsOverviewResponseObject, error) {
	response, err := h.service.GetComplianceOverview(ctx)
	if err != nil {
		return api.GetComplianceAnalyticsOverview500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: analyticsErrorMessage(err, "failed to get compliance analytics")},
		}, nil
	}

	return api.GetComplianceAnalyticsOverview200JSONResponse{Data: response}, nil
}

func (h handler) GetFleetComplianceAnalyticsOverview(ctx context.Context, _ api.GetFleetComplianceAnalyticsOverviewRequestObject) (api.GetFleetComplianceAnalyticsOverviewResponseObject, error) {
	response, err := h.service.GetFleetComplianceOverview(ctx)
	if err != nil {
		return api.GetFleetComplianceAnalyticsOverview500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: analyticsErrorMessage(err, "failed to get fleet compliance analytics")},
		}, nil
	}

	return api.GetFleetComplianceAnalyticsOverview200JSONResponse{Data: response}, nil
}

func (h handler) ListNonCompliantServers(ctx context.Context, request api.ListNonCompliantServersRequestObject) (api.ListNonCompliantServersResponseObject, error) {
	page, limit, ok := paginationParams(request.Params.Page, request.Params.Limit)
	if !ok {
		return api.ListNonCompliantServers400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{Error: "page and limit must be positive integers"},
		}, nil
	}

	servers, paginationResponse, err := h.service.ListNonCompliantServers(ctx, page, limit)
	if err != nil {
		return api.ListNonCompliantServers500JSONResponse{
			InternalServerErrorJSONResponse: api.InternalServerErrorJSONResponse{Error: analyticsErrorMessage(err, "failed to list non-compliant servers")},
		}, nil
	}

	return api.ListNonCompliantServers200JSONResponse{
		Data:       servers,
		Pagination: paginationResponse,
	}, nil
}

func NewHandler(service Service) (Handler, error) {
	if service == nil {
		return nil, errors.New("analytics service is required")
	}

	return &handler{service: service}, nil
}

func analyticsErrorMessage(err error, fallback string) string {
	if analyticsErr, ok := errors.AsType[AnalyticsError](err); ok {
		return analyticsErr.Message
	}

	return fallback
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

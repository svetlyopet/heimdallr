package analytics

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler interface {
	GetAutomationOverview(ctx *gin.Context)
	GetAutomationOverviewByID(ctx *gin.Context)
}

type handler struct {
	service Service
}

func (h handler) GetAutomationOverview(ctx *gin.Context) {
	response, err := h.service.GetAutomationOverview(ctx.Request.Context())
	if err != nil {
		analyticsErr := NewGetAutomationAnalyticsError(err)
		returnErrorResponse(ctx, http.StatusInternalServerError, analyticsErr)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": response,
	})
}

func (h handler) GetAutomationOverviewByID(ctx *gin.Context) {
	automationID := strings.TrimSpace(ctx.Param("id"))
	if automationID == "" {
		analyticsErr := NewAnalyticsError("invalid automation id", errors.New("automation id is required"))
		returnErrorResponse(ctx, http.StatusBadRequest, analyticsErr)
		return
	}

	if _, err := uuid.Parse(automationID); err != nil {
		analyticsErr := NewAnalyticsError("invalid automation id", err)
		returnErrorResponse(ctx, http.StatusBadRequest, analyticsErr)
		return
	}

	response, err := h.service.GetAutomationOverviewByID(ctx.Request.Context(), automationID)
	if err != nil {
		if errors.Is(err, ErrAutomationNotFound) {
			analyticsErr := NewAnalyticsError("automation not found", err)
			returnErrorResponse(ctx, http.StatusNotFound, analyticsErr)
			return
		}

		analyticsErr := NewGetAutomationAnalyticsError(err)
		returnErrorResponse(ctx, http.StatusInternalServerError, analyticsErr)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": response,
	})
}

func NewHandler(service Service) (Handler, error) {
	if service == nil {
		return nil, errors.New("analytics service is required")
	}

	return &handler{
		service: service,
	}, nil
}

func returnErrorResponse(ctx *gin.Context, statusCode int, err error) {
	if analyticsErr, ok := errors.AsType[AnalyticsError](err); ok {
		ctx.JSON(statusCode, gin.H{
			"error": analyticsErr.Message,
		})
		return
	}

	ctx.JSON(http.StatusInternalServerError, gin.H{
		"error": http.StatusText(http.StatusInternalServerError),
	})
}

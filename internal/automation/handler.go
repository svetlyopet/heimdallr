package automation

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler interface {
	List(ctx *gin.Context)
	Get(ctx *gin.Context)
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
}

type handler struct {
	service Service
}

func (h handler) List(ctx *gin.Context) {
	page, err := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		automationErr := NewAutomationError("invalid query param value", errors.New("page must be a positive integer"))
		returnErrorResponse(ctx, http.StatusBadRequest, automationErr)
		return
	}

	limit, err := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 {
		automationErr := NewAutomationError("invalid query param value", errors.New("limit must be a positive integer"))
		returnErrorResponse(ctx, http.StatusBadRequest, automationErr)
		return
	}

	automations, total, err := h.service.GetAll(ctx.Request.Context(), page, limit)
	if err != nil {
		automationErr := NewGetAutomationsError(err)
		returnErrorResponse(ctx, http.StatusInternalServerError, automationErr)
		return
	}

	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(limit) - 1) / int64(limit)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": automations,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

func (h handler) Get(ctx *gin.Context) {
	automationID := ctx.Param("automation_id")
	if automationID == "" {
		automationErr := NewInvalidAutomationIDError(ErrInvalidAutomationID)
		returnErrorResponse(ctx, http.StatusBadRequest, automationErr)
		return
	}

	automation, err := h.service.GetById(ctx.Request.Context(), automationID)
	if err != nil {
		if errors.Is(err, ErrAutomationNotFound) {
			automationErr := NewAutomationNotFoundError(err)
			returnErrorResponse(ctx, http.StatusNotFound, automationErr)
			return
		}

		automationErr := NewGetAutomationError(err)
		returnErrorResponse(ctx, http.StatusInternalServerError, automationErr)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": automation,
	})
}

func (h handler) Create(ctx *gin.Context) {
	var req CreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		automationErr := NewAutomationError("invalid request body", err)
		returnErrorResponse(ctx, http.StatusBadRequest, automationErr)
		return
	}

	automation, err := h.service.Create(ctx.Request.Context(), req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, ErrAutomationAlreadyExists) {
			statusCode = http.StatusConflict
		}

		automationErr := NewCreateAutomationError(err)
		returnErrorResponse(ctx, statusCode, automationErr)
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"data": automation,
	})
}

func (h handler) Update(ctx *gin.Context) {
	automationID := ctx.Param("automation_id")
	if automationID == "" {
		automationErr := NewInvalidAutomationIDError(ErrInvalidAutomationID)
		returnErrorResponse(ctx, http.StatusBadRequest, automationErr)
		return
	}

	var req UpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		automationErr := NewAutomationError("invalid request body", err)
		returnErrorResponse(ctx, http.StatusBadRequest, automationErr)
		return
	}

	automation, err := h.service.Update(ctx.Request.Context(), req, automationID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, ErrAutomationNotFound) {
			statusCode = http.StatusNotFound
		}

		automationErr := NewUpdateAutomationError(err)
		returnErrorResponse(ctx, statusCode, automationErr)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": automation,
	})
}

func (h handler) Delete(ctx *gin.Context) {
	automationID := ctx.Param("automation_id")
	if automationID == "" {
		automationErr := NewInvalidAutomationIDError(ErrInvalidAutomationID)
		returnErrorResponse(ctx, http.StatusBadRequest, automationErr)
		return
	}

	if err := h.service.Delete(ctx.Request.Context(), automationID); err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, ErrAutomationNotFound) {
			statusCode = http.StatusNotFound
		}

		automationErr := NewDeleteAutomationError(err)
		returnErrorResponse(ctx, statusCode, automationErr)
		return
	}

	ctx.Status(http.StatusNoContent)
}

func NewHandler(service Service) (Handler, error) {
	return &handler{
		service: service,
	}, nil
}

func returnErrorResponse(ctx *gin.Context, statusCode int, err error) {
	if automationErr, ok := errors.AsType[AutomationError](err); ok {
		ctx.JSON(statusCode, gin.H{
			"error": automationErr.Message,
		})
		return
	}

	ctx.JSON(http.StatusInternalServerError, gin.H{
		"error": http.StatusText(http.StatusInternalServerError),
	})
}

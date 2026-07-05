package application

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
}

type handler struct {
	service Service
}

func (h handler) List(ctx *gin.Context) {
	page, err := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		applicationErr := NewApplicationError("invalid query param value", errors.New("page must be a positive integer"))
		returnErrorResponse(ctx, http.StatusBadRequest, applicationErr)
		return
	}

	limit, err := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 {
		applicationErr := NewApplicationError("invalid query param value", errors.New("limit must be a positive integer"))
		returnErrorResponse(ctx, http.StatusBadRequest, applicationErr)
		return
	}

	applications, total, err := h.service.GetAll(ctx.Request.Context(), page, limit)
	if err != nil {
		returnErrorResponse(ctx, http.StatusInternalServerError, NewGetApplicationsError(err))
		return
	}

	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(limit) - 1) / int64(limit)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": applications,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

func (h handler) Get(ctx *gin.Context) {
	applicationID := ctx.Param("application_id")
	if applicationID == "" {
		returnErrorResponse(ctx, http.StatusBadRequest, NewInvalidApplicationIDError(ErrInvalidApplicationID))
		return
	}

	application, err := h.service.GetById(ctx.Request.Context(), applicationID)
	if err != nil {
		if errors.Is(err, ErrApplicationNotFound) {
			returnErrorResponse(ctx, http.StatusNotFound, NewApplicationNotFoundError(err))
			return
		}

		returnErrorResponse(ctx, http.StatusInternalServerError, NewGetApplicationError(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": application})
}

func (h handler) Create(ctx *gin.Context) {
	var req CreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		returnErrorResponse(ctx, http.StatusBadRequest, NewApplicationError("invalid request body", err))
		return
	}

	application, err := h.service.Create(ctx.Request.Context(), req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, ErrApplicationAlreadyExists) {
			statusCode = http.StatusConflict
		}

		returnErrorResponse(ctx, statusCode, NewCreateApplicationError(err))
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"data": application})
}

func NewHandler(service Service) (Handler, error) {
	return &handler{service: service}, nil
}

func returnErrorResponse(ctx *gin.Context, statusCode int, err error) {
	if applicationErr, ok := errors.AsType[ApplicationError](err); ok {
		ctx.JSON(statusCode, gin.H{"error": applicationErr.Message})
		return
	}

	ctx.JSON(http.StatusInternalServerError, gin.H{"error": http.StatusText(http.StatusInternalServerError)})
}

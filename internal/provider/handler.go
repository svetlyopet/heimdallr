package provider

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
		providerErr := NewProviderError("invalid query param value", errors.New("page must be a positive integer"))
		returnErrorResponse(ctx, http.StatusBadRequest, providerErr)
		return
	}

	limit, err := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 {
		providerErr := NewProviderError("invalid query param value", errors.New("limit must be a positive integer"))
		returnErrorResponse(ctx, http.StatusBadRequest, providerErr)
		return
	}

	providers, total, err := h.service.GetAll(ctx.Request.Context(), page, limit)
	if err != nil {
		providerErr := NewGetProvidersError(err)
		returnErrorResponse(ctx, http.StatusInternalServerError, providerErr)
		return
	}

	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(limit) - 1) / int64(limit)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": providers,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

func (h handler) Get(ctx *gin.Context) {
	providerID := ctx.Param("provider_id")
	if providerID == "" {
		providerErr := NewInvalidProviderIDError(ErrInvalidProviderID)
		returnErrorResponse(ctx, http.StatusBadRequest, providerErr)
		return
	}

	provider, err := h.service.GetById(ctx.Request.Context(), providerID)
	if err != nil {
		if errors.Is(err, ErrProviderNotFound) {
			providerErr := NewProviderNotFoundError(err)
			returnErrorResponse(ctx, http.StatusNotFound, providerErr)
			return
		}

		providerErr := NewGetProviderError(err)
		returnErrorResponse(ctx, http.StatusInternalServerError, providerErr)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": provider,
	})
}

func (h handler) Create(ctx *gin.Context) {
	var req CreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		providerErr := NewProviderError("invalid request body", err)
		returnErrorResponse(ctx, http.StatusBadRequest, providerErr)
		return
	}

	provider, err := h.service.Create(ctx.Request.Context(), req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, ErrProviderAlreadyExists) {
			statusCode = http.StatusConflict
		}

		providerErr := NewCreateProviderError(err)
		returnErrorResponse(ctx, statusCode, providerErr)
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"data": provider,
	})
}

func NewHandler(service Service) (Handler, error) {
	return &handler{
		service: service,
	}, nil
}

func returnErrorResponse(ctx *gin.Context, statusCode int, err error) {
	if providerErr, ok := errors.AsType[ProviderError](err); ok {
		ctx.JSON(statusCode, gin.H{
			"error": providerErr.Message,
		})
		return
	}

	ctx.JSON(http.StatusInternalServerError, gin.H{
		"error": http.StatusText(http.StatusInternalServerError),
	})
}

package release

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/application"
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
	applicationID, ok := getValidApplicationID(ctx)
	if !ok {
		return
	}

	page, err := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		returnErrorResponse(ctx, http.StatusBadRequest, NewReleaseError("invalid query param value", errors.New("page must be a positive integer")))
		return
	}

	limit, err := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 {
		returnErrorResponse(ctx, http.StatusBadRequest, NewReleaseError("invalid query param value", errors.New("limit must be a positive integer")))
		return
	}

	releases, total, err := h.service.GetAll(ctx.Request.Context(), applicationID, page, limit)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, application.ErrApplicationNotFound) {
			statusCode = http.StatusNotFound
		}

		returnErrorResponse(ctx, statusCode, NewGetReleasesError(err))
		return
	}

	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(limit) - 1) / int64(limit)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": releases,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

func (h handler) Get(ctx *gin.Context) {
	applicationID, ok := getValidApplicationID(ctx)
	if !ok {
		return
	}

	releaseID, ok := getValidReleaseID(ctx)
	if !ok {
		return
	}

	release, err := h.service.GetById(ctx.Request.Context(), releaseID, applicationID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, ErrReleaseNotFound) {
			statusCode = http.StatusNotFound
		}

		returnErrorResponse(ctx, statusCode, NewGetReleaseError(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": release})
}

func (h handler) Create(ctx *gin.Context) {
	applicationID, ok := getValidApplicationID(ctx)
	if !ok {
		return
	}

	var req CreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		returnErrorResponse(ctx, http.StatusBadRequest, NewReleaseError("invalid request body", err))
		return
	}

	upsert := strings.EqualFold(ctx.DefaultQuery("upsert", "false"), "true")

	release, err := h.service.Create(ctx.Request.Context(), applicationID, req, upsert)
	if err != nil {
		statusCode := http.StatusInternalServerError
		switch {
		case errors.Is(err, application.ErrApplicationNotFound):
			statusCode = http.StatusNotFound
		case errors.Is(err, ErrReleaseAlreadyExists):
			statusCode = http.StatusConflict
		}

		returnErrorResponse(ctx, statusCode, NewCreateReleaseError(err))
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"data": release})
}

func NewHandler(service Service) (Handler, error) {
	return &handler{service: service}, nil
}

func getValidApplicationID(ctx *gin.Context) (string, bool) {
	applicationID := strings.TrimSpace(ctx.Param("application_id"))
	if applicationID == "" {
		returnErrorResponse(ctx, http.StatusBadRequest, NewInvalidApplicationIDError(ErrInvalidApplicationID))
		return "", false
	}

	if _, err := uuid.Parse(applicationID); err != nil {
		returnErrorResponse(ctx, http.StatusBadRequest, NewInvalidApplicationIDError(err))
		return "", false
	}

	return applicationID, true
}

func getValidReleaseID(ctx *gin.Context) (string, bool) {
	releaseID := strings.TrimSpace(ctx.Param("release_id"))
	if releaseID == "" {
		returnErrorResponse(ctx, http.StatusBadRequest, NewInvalidReleaseIDError(ErrInvalidReleaseID))
		return "", false
	}

	if _, err := uuid.Parse(releaseID); err != nil {
		returnErrorResponse(ctx, http.StatusBadRequest, NewInvalidReleaseIDError(err))
		return "", false
	}

	return releaseID, true
}

func returnErrorResponse(ctx *gin.Context, statusCode int, err error) {
	if releaseErr, ok := errors.AsType[ReleaseError](err); ok {
		ctx.JSON(statusCode, gin.H{"error": releaseErr.Message})
		return
	}

	ctx.JSON(http.StatusInternalServerError, gin.H{"error": http.StatusText(http.StatusInternalServerError)})
}

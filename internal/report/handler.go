package report

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler interface {
	List(ctx *gin.Context)
	Get(ctx *gin.Context)
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
}

type handler struct {
	service Service
}

func (h handler) List(ctx *gin.Context) {
	applicationID, releaseID, ok := getValidScope(ctx)
	if !ok {
		return
	}

	page, err := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		returnErrorResponse(ctx, http.StatusBadRequest, NewReportError("invalid query param value", errors.New("page must be a positive integer")))
		return
	}

	limit, err := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 {
		returnErrorResponse(ctx, http.StatusBadRequest, NewReportError("invalid query param value", errors.New("limit must be a positive integer")))
		return
	}

	reports, total, err := h.service.GetAll(ctx.Request.Context(), applicationID, releaseID, page, limit)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, ErrReportNotFound) {
			statusCode = http.StatusNotFound
		}

		returnErrorResponse(ctx, statusCode, NewGetReportsError(err))
		return
	}

	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(limit) - 1) / int64(limit)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": reports,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

func (h handler) Get(ctx *gin.Context) {
	applicationID, releaseID, ok := getValidScope(ctx)
	if !ok {
		return
	}

	reportID, ok := getValidReportID(ctx)
	if !ok {
		return
	}

	report, err := h.service.GetById(ctx.Request.Context(), applicationID, releaseID, reportID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, ErrReportNotFound) {
			statusCode = http.StatusNotFound
		}

		returnErrorResponse(ctx, statusCode, NewGetReportError(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": report})
}

func (h handler) Create(ctx *gin.Context) {
	applicationID, releaseID, ok := getValidScope(ctx)
	if !ok {
		return
	}

	var req CreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		returnErrorResponse(ctx, http.StatusBadRequest, NewReportError("invalid request body", err))
		return
	}

	if req.Metadata != nil && !json.Valid(req.Metadata) {
		returnErrorResponse(ctx, http.StatusBadRequest, NewInvalidMetadataError(errors.New("not valid json")))
		return
	}

	if req.Output != "" && !isValidBase64(req.Output) {
		returnErrorResponse(ctx, http.StatusBadRequest, NewInvalidOutputError(errors.New("not valid encoding")))
		return
	}

	report, err := h.service.Create(ctx.Request.Context(), applicationID, releaseID, req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, ErrReportNotFound) {
			statusCode = http.StatusNotFound
		}

		returnErrorResponse(ctx, statusCode, NewCreateReportError(err))
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"data": report})
}

func (h handler) Update(ctx *gin.Context) {
	applicationID, releaseID, ok := getValidScope(ctx)
	if !ok {
		return
	}

	reportID, ok := getValidReportID(ctx)
	if !ok {
		return
	}

	var req UpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		returnErrorResponse(ctx, http.StatusBadRequest, NewReportError("invalid request body", err))
		return
	}

	if req.Metadata != nil && !json.Valid(req.Metadata) {
		returnErrorResponse(ctx, http.StatusBadRequest, NewInvalidMetadataError(errors.New("not valid json")))
		return
	}

	if req.Output != "" && !isValidBase64(req.Output) {
		returnErrorResponse(ctx, http.StatusBadRequest, NewInvalidOutputError(errors.New("not valid encoding")))
		return
	}

	report, err := h.service.Update(ctx.Request.Context(), applicationID, releaseID, reportID, req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, ErrReportNotFound) {
			statusCode = http.StatusNotFound
		}

		returnErrorResponse(ctx, statusCode, NewUpdateReportError(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": report})
}

func NewHandler(service Service) (Handler, error) {
	return &handler{service: service}, nil
}

func getValidScope(ctx *gin.Context) (string, string, bool) {
	applicationID := strings.TrimSpace(ctx.Param("application_id"))
	if applicationID == "" {
		returnErrorResponse(ctx, http.StatusBadRequest, NewInvalidApplicationIDError(ErrInvalidApplicationID))
		return "", "", false
	}

	if _, err := uuid.Parse(applicationID); err != nil {
		returnErrorResponse(ctx, http.StatusBadRequest, NewInvalidApplicationIDError(err))
		return "", "", false
	}

	releaseID := strings.TrimSpace(ctx.Param("release_id"))
	if releaseID == "" {
		returnErrorResponse(ctx, http.StatusBadRequest, NewInvalidReleaseIDError(ErrInvalidReleaseID))
		return "", "", false
	}

	if _, err := uuid.Parse(releaseID); err != nil {
		returnErrorResponse(ctx, http.StatusBadRequest, NewInvalidReleaseIDError(err))
		return "", "", false
	}

	return applicationID, releaseID, true
}

func getValidReportID(ctx *gin.Context) (string, bool) {
	reportID := strings.TrimSpace(ctx.Param("report_id"))
	if reportID == "" {
		returnErrorResponse(ctx, http.StatusBadRequest, NewInvalidReportIDError(ErrInvalidReportID))
		return "", false
	}

	return reportID, true
}

func returnErrorResponse(ctx *gin.Context, statusCode int, err error) {
	if reportErr, ok := errors.AsType[ReportError](err); ok {
		ctx.JSON(statusCode, gin.H{"error": reportErr.Message})
		return
	}

	ctx.JSON(http.StatusInternalServerError, gin.H{"error": http.StatusText(http.StatusInternalServerError)})
}

func isValidBase64(value string) bool {
	if value == "" {
		return true
	}

	_, err := base64.StdEncoding.DecodeString(value)
	return err == nil
}

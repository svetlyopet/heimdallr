package job

import (
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
	automationID, ok := getValidAutomationID(ctx)
	if !ok {
		return
	}

	page, err := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		jobErr := NewJobError("invalid query param value", errors.New("page must be a positive integer"))
		returnErrorResponse(ctx, http.StatusBadRequest, jobErr)
		return
	}

	limit, err := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 {
		jobErr := NewJobError("invalid query param value", errors.New("limit must be a positive integer"))
		returnErrorResponse(ctx, http.StatusBadRequest, jobErr)
		return
	}

	jobs, total, err := h.service.GetAll(ctx.Request.Context(), automationID, page, limit)
	if err != nil {
		jobErr := NewGetJobsError(err)
		returnErrorResponse(ctx, http.StatusInternalServerError, jobErr)
		return
	}

	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(limit) - 1) / int64(limit)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": jobs,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

func (h handler) Get(ctx *gin.Context) {
	automationID, ok := getValidAutomationID(ctx)
	if !ok {
		return
	}

	jobID, ok := getValidJobID(ctx)
	if !ok {
		return
	}

	job, err := h.service.GetById(ctx.Request.Context(), jobID, automationID)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			jobErr := NewJobNotFoundError(err)
			returnErrorResponse(ctx, http.StatusNotFound, jobErr)
			return
		}

		jobErr := NewGetJobError(err)
		returnErrorResponse(ctx, http.StatusInternalServerError, jobErr)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": job,
	})
}

func (h handler) Create(ctx *gin.Context) {
	automationID, ok := getValidAutomationID(ctx)
	if !ok {
		return
	}

	var req CreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		jobErr := NewJobError("invalid request body", err)
		returnErrorResponse(ctx, http.StatusBadRequest, jobErr)
		return
	}

	job, err := h.service.Create(ctx.Request.Context(), automationID, req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, ErrJobNotFound) {
			statusCode = http.StatusNotFound
		}

		jobErr := NewCreateJobError(err)
		returnErrorResponse(ctx, statusCode, jobErr)
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"data": job,
	})
}

func (h handler) Update(ctx *gin.Context) {
	automationID, ok := getValidAutomationID(ctx)
	if !ok {
		return
	}

	jobID, ok := getValidJobID(ctx)
	if !ok {
		return
	}

	var req UpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		jobErr := NewJobError("invalid request body", err)
		returnErrorResponse(ctx, http.StatusBadRequest, jobErr)
		return
	}

	job, err := h.service.Update(ctx.Request.Context(), automationID, jobID, req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, ErrJobNotFound) {
			statusCode = http.StatusNotFound
		}

		jobErr := NewUpdateJobError(err)
		returnErrorResponse(ctx, statusCode, jobErr)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": job,
	})
}

func NewHandler(service Service) (Handler, error) {
	return &handler{
		service: service,
	}, nil
}

func getValidAutomationID(ctx *gin.Context) (string, bool) {
	automationID := strings.TrimSpace(ctx.Param("automation_id"))
	if automationID == "" {
		jobErr := NewInvalidAutomationIDError(ErrInvalidAutomationID)
		returnErrorResponse(ctx, http.StatusBadRequest, jobErr)
		return "", false
	}

	if _, err := uuid.Parse(automationID); err != nil {
		jobErr := NewInvalidAutomationIDError(err)
		returnErrorResponse(ctx, http.StatusBadRequest, jobErr)
		return "", false
	}

	return automationID, true
}

func getValidJobID(ctx *gin.Context) (string, bool) {
	jobID := strings.TrimSpace(ctx.Param("job_id"))
	if jobID == "" {
		jobErr := NewInvalidJobIDError(ErrInvalidJobID)
		returnErrorResponse(ctx, http.StatusBadRequest, jobErr)
		return "", false
	}

	return jobID, true
}

func returnErrorResponse(ctx *gin.Context, statusCode int, err error) {
	if jobErr, ok := errors.AsType[JobError](err); ok {
		ctx.JSON(statusCode, gin.H{
			"error": jobErr.Message,
		})
		return
	}

	ctx.JSON(http.StatusInternalServerError, gin.H{
		"error": http.StatusText(http.StatusInternalServerError),
	})
}

package server

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler interface {
	List(ctx *gin.Context)
	Get(ctx *gin.Context)
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)

	ListJobs(ctx *gin.Context)
	AssociateJob(ctx *gin.Context)
	DissociateJob(ctx *gin.Context)

	ListReleases(ctx *gin.Context)
	AssociateRelease(ctx *gin.Context)
	DissociateRelease(ctx *gin.Context)
}

type handler struct {
	service Service
}

func (h handler) List(ctx *gin.Context) {
	page, limit, ok := parsePagination(ctx)
	if !ok {
		return
	}

	servers, total, err := h.service.GetAll(ctx.Request.Context(), page, limit)
	if err != nil {
		returnErrorResponse(ctx, http.StatusInternalServerError, NewGetServersError(err))
		return
	}

	writePaginatedResponse(ctx, servers, page, limit, total)
}

func (h handler) Get(ctx *gin.Context) {
	serverID, ok := getValidServerID(ctx)
	if !ok {
		return
	}

	server, err := h.service.GetById(ctx.Request.Context(), serverID)
	if err != nil {
		if errors.Is(err, ErrServerNotFound) {
			returnErrorResponse(ctx, http.StatusNotFound, NewServerNotFoundError(err))
			return
		}

		returnErrorResponse(ctx, http.StatusInternalServerError, NewGetServerError(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": server})
}

func (h handler) Create(ctx *gin.Context) {
	var req CreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		returnErrorResponse(ctx, http.StatusBadRequest, NewServerError("invalid request body", err))
		return
	}

	server, err := h.service.Create(ctx.Request.Context(), req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, ErrServerAlreadyExists) {
			statusCode = http.StatusConflict
		} else if errors.Is(err, ErrAgentAlreadyAssigned) {
			statusCode = http.StatusConflict
		}

		returnErrorResponse(ctx, statusCode, NewCreateServerError(err))
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"data": server})
}

func (h handler) Update(ctx *gin.Context) {
	serverID, ok := getValidServerID(ctx)
	if !ok {
		return
	}

	var req UpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		returnErrorResponse(ctx, http.StatusBadRequest, NewServerError("invalid request body", err))
		return
	}

	server, err := h.service.Update(ctx.Request.Context(), serverID, req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrServerNotFound):
			statusCode = http.StatusNotFound
		case errors.Is(err, ErrAgentAlreadyAssigned):
			statusCode = http.StatusConflict
		}

		errToReturn := NewUpdateServerError(err)
		if errors.Is(err, ErrAgentAlreadyAssigned) {
			errToReturn = NewAgentAlreadyAssignedError(err)
		}

		returnErrorResponse(ctx, statusCode, errToReturn)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": server})
}

func (h handler) ListJobs(ctx *gin.Context) {
	serverID, ok := getValidServerID(ctx)
	if !ok {
		return
	}

	page, limit, ok := parsePagination(ctx)
	if !ok {
		return
	}

	jobs, total, err := h.service.ListJobs(ctx.Request.Context(), serverID, page, limit)
	if err != nil {
		if errors.Is(err, ErrServerNotFound) {
			returnErrorResponse(ctx, http.StatusNotFound, NewServerNotFoundError(err))
			return
		}

		returnErrorResponse(ctx, http.StatusInternalServerError, NewListJobsError(err))
		return
	}

	writePaginatedResponse(ctx, jobs, page, limit, total)
}

func (h handler) AssociateJob(ctx *gin.Context) {
	serverID, ok := getValidServerID(ctx)
	if !ok {
		return
	}

	var req JobAssociateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		returnErrorResponse(ctx, http.StatusBadRequest, NewServerError("invalid request body", err))
		return
	}

	if err := h.service.AssociateJob(ctx.Request.Context(), serverID, req); err != nil {
		statusCode := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrServerNotFound), errors.Is(err, ErrJobNotFound):
			statusCode = http.StatusNotFound
		case errors.Is(err, ErrJobAlreadyAssociated):
			statusCode = http.StatusConflict
		}

		returnErrorResponse(ctx, statusCode, NewAssociateJobError(err))
		return
	}

	ctx.Status(http.StatusCreated)
}

func (h handler) DissociateJob(ctx *gin.Context) {
	serverID, ok := getValidServerID(ctx)
	if !ok {
		return
	}

	jobID := ctx.Param("job_id")
	if jobID == "" {
		returnErrorResponse(ctx, http.StatusBadRequest, NewServerError("invalid job id", ErrJobNotFound))
		return
	}

	automationIDParam := ctx.Query("automation_id")
	if automationIDParam == "" {
		returnErrorResponse(ctx, http.StatusBadRequest, NewServerError("automation_id query param is required", ErrJobNotFound))
		return
	}

	automationID, err := uuid.Parse(automationIDParam)
	if err != nil {
		returnErrorResponse(ctx, http.StatusBadRequest, NewServerError("invalid automation id", err))
		return
	}

	if err := h.service.DissociateJob(ctx.Request.Context(), serverID, jobID, automationID); err != nil {
		statusCode := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrServerNotFound), errors.Is(err, ErrJobNotFound):
			statusCode = http.StatusNotFound
		}

		returnErrorResponse(ctx, statusCode, NewDissociateJobError(err))
		return
	}

	ctx.Status(http.StatusNoContent)
}

func (h handler) ListReleases(ctx *gin.Context) {
	serverID, ok := getValidServerID(ctx)
	if !ok {
		return
	}

	page, limit, ok := parsePagination(ctx)
	if !ok {
		return
	}

	releases, total, err := h.service.ListReleases(ctx.Request.Context(), serverID, page, limit)
	if err != nil {
		if errors.Is(err, ErrServerNotFound) {
			returnErrorResponse(ctx, http.StatusNotFound, NewServerNotFoundError(err))
			return
		}

		returnErrorResponse(ctx, http.StatusInternalServerError, NewListReleasesError(err))
		return
	}

	writePaginatedResponse(ctx, releases, page, limit, total)
}

func (h handler) AssociateRelease(ctx *gin.Context) {
	serverID, ok := getValidServerID(ctx)
	if !ok {
		return
	}

	var req ReleaseAssociateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		returnErrorResponse(ctx, http.StatusBadRequest, NewServerError("invalid request body", err))
		return
	}

	if err := h.service.AssociateRelease(ctx.Request.Context(), serverID, req); err != nil {
		statusCode := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrServerNotFound), errors.Is(err, ErrReleaseNotFound):
			statusCode = http.StatusNotFound
		case errors.Is(err, ErrReleaseAlreadyAssociated):
			statusCode = http.StatusConflict
		}

		returnErrorResponse(ctx, statusCode, NewAssociateReleaseError(err))
		return
	}

	ctx.Status(http.StatusCreated)
}

func (h handler) DissociateRelease(ctx *gin.Context) {
	serverID, ok := getValidServerID(ctx)
	if !ok {
		return
	}

	releaseIDParam := ctx.Param("release_id")
	if releaseIDParam == "" {
		returnErrorResponse(ctx, http.StatusBadRequest, NewServerError("invalid release id", ErrReleaseNotFound))
		return
	}

	releaseID, err := uuid.Parse(releaseIDParam)
	if err != nil {
		returnErrorResponse(ctx, http.StatusBadRequest, NewServerError("invalid release id", err))
		return
	}

	if err := h.service.DissociateRelease(ctx.Request.Context(), serverID, releaseID); err != nil {
		statusCode := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrServerNotFound), errors.Is(err, ErrReleaseNotFound):
			statusCode = http.StatusNotFound
		}

		returnErrorResponse(ctx, statusCode, NewDissociateReleaseError(err))
		return
	}

	ctx.Status(http.StatusNoContent)
}

func NewHandler(service Service) (Handler, error) {
	return &handler{service: service}, nil
}

func getValidServerID(ctx *gin.Context) (string, bool) {
	serverID := ctx.Param("server_id")
	if serverID == "" {
		returnErrorResponse(ctx, http.StatusBadRequest, NewInvalidServerIDError(ErrInvalidServerID))
		return "", false
	}

	if _, err := uuid.Parse(serverID); err != nil {
		returnErrorResponse(ctx, http.StatusBadRequest, NewInvalidServerIDError(err))
		return "", false
	}

	return serverID, true
}

func parsePagination(ctx *gin.Context) (int, int, bool) {
	page, err := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		returnErrorResponse(ctx, http.StatusBadRequest, NewServerError("invalid query param value", errors.New("page must be a positive integer")))
		return 0, 0, false
	}

	limit, err := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 {
		returnErrorResponse(ctx, http.StatusBadRequest, NewServerError("invalid query param value", errors.New("limit must be a positive integer")))
		return 0, 0, false
	}

	return page, limit, true
}

func writePaginatedResponse(ctx *gin.Context, data any, page int, limit int, total int64) {
	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(limit) - 1) / int64(limit)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": data,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

func returnErrorResponse(ctx *gin.Context, statusCode int, err error) {
	if serverErr, ok := errors.AsType[ServerError](err); ok {
		ctx.JSON(statusCode, gin.H{"error": serverErr.Message})
		return
	}

	ctx.JSON(http.StatusInternalServerError, gin.H{"error": http.StatusText(http.StatusInternalServerError)})
}

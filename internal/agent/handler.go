package agent

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/server"
)

type Handler interface {
	List(ctx *gin.Context)
	Get(ctx *gin.Context)
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)

	ListGlobal(ctx *gin.Context)
	GetGlobal(ctx *gin.Context)
	CreateUnassigned(ctx *gin.Context)
}

type handler struct {
	service Service
}

func (h handler) List(ctx *gin.Context) {
	serverID, ok := getValidServerID(ctx)
	if !ok {
		return
	}

	page, err := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		returnErrorResponse(ctx, http.StatusBadRequest, NewAgentError("invalid query param value", errors.New("page must be a positive integer")))
		return
	}

	limit, err := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 {
		returnErrorResponse(ctx, http.StatusBadRequest, NewAgentError("invalid query param value", errors.New("limit must be a positive integer")))
		return
	}

	agents, total, err := h.service.GetAll(ctx.Request.Context(), serverID, page, limit)
	if err != nil {
		if errors.Is(err, server.ErrServerNotFound) {
			returnErrorResponse(ctx, http.StatusNotFound, NewAgentError(server.ErrServerNotFound.Error(), err))
			return
		}

		returnErrorResponse(ctx, http.StatusInternalServerError, NewListAgentsError(err))
		return
	}

	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(limit) - 1) / int64(limit)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": agents,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

func (h handler) Get(ctx *gin.Context) {
	serverID, ok := getValidServerID(ctx)
	if !ok {
		return
	}

	agentID, ok := getValidAgentID(ctx)
	if !ok {
		return
	}

	agent, err := h.service.GetById(ctx.Request.Context(), agentID, serverID)
	if err != nil {
		if errors.Is(err, ErrAgentNotFound) {
			returnErrorResponse(ctx, http.StatusNotFound, NewAgentNotFoundError(err))
			return
		}

		returnErrorResponse(ctx, http.StatusInternalServerError, NewGetAgentError(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": agent})
}

func (h handler) Create(ctx *gin.Context) {
	serverID, ok := getValidServerID(ctx)
	if !ok {
		return
	}

	var req CreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		returnErrorResponse(ctx, http.StatusBadRequest, NewAgentError("invalid request body", err))
		return
	}

	agent, err := h.service.Create(ctx.Request.Context(), serverID, req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, server.ErrServerNotFound) {
			statusCode = http.StatusNotFound
		}

		returnErrorResponse(ctx, statusCode, NewCreateAgentError(err))
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"data": agent})
}

func (h handler) Delete(ctx *gin.Context) {
	serverID, ok := getValidServerID(ctx)
	if !ok {
		return
	}

	agentID, ok := getValidAgentID(ctx)
	if !ok {
		return
	}

	if err := h.service.Delete(ctx.Request.Context(), serverID, agentID); err != nil {
		statusCode := http.StatusInternalServerError
		switch {
		case errors.Is(err, server.ErrServerNotFound):
			statusCode = http.StatusNotFound
		case errors.Is(err, ErrAgentNotFound):
			statusCode = http.StatusNotFound
		}

		returnErrorResponse(ctx, statusCode, NewDeleteAgentError(err))
		return
	}

	ctx.Status(http.StatusNoContent)
}

func (h handler) ListGlobal(ctx *gin.Context) {
	page, err := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		returnErrorResponse(ctx, http.StatusBadRequest, NewAgentError("invalid query param value", errors.New("page must be a positive integer")))
		return
	}

	limit, err := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 {
		returnErrorResponse(ctx, http.StatusBadRequest, NewAgentError("invalid query param value", errors.New("limit must be a positive integer")))
		return
	}

	unassignedOnly := ctx.Query("unassigned") == "true"

	agents, total, err := h.service.ListGlobal(ctx.Request.Context(), unassignedOnly, page, limit)
	if err != nil {
		returnErrorResponse(ctx, http.StatusInternalServerError, NewListAgentsError(err))
		return
	}

	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + int64(limit) - 1) / int64(limit)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": agents,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

func (h handler) GetGlobal(ctx *gin.Context) {
	agentID, ok := getValidGlobalAgentID(ctx)
	if !ok {
		return
	}

	agent, err := h.service.GetByIdGlobal(ctx.Request.Context(), agentID)
	if err != nil {
		if errors.Is(err, ErrAgentNotFound) {
			returnErrorResponse(ctx, http.StatusNotFound, NewAgentNotFoundError(err))
			return
		}

		returnErrorResponse(ctx, http.StatusInternalServerError, NewGetAgentError(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": agent})
}

func (h handler) CreateUnassigned(ctx *gin.Context) {
	var req CreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		returnErrorResponse(ctx, http.StatusBadRequest, NewAgentError("invalid request body", err))
		return
	}

	agent, err := h.service.CreateUnassigned(ctx.Request.Context(), req)
	if err != nil {
		returnErrorResponse(ctx, http.StatusInternalServerError, NewCreateAgentError(err))
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"data": agent})
}

func NewHandler(service Service) (Handler, error) {
	return &handler{service: service}, nil
}

func getValidServerID(ctx *gin.Context) (string, bool) {
	serverID := strings.TrimSpace(ctx.Param("server_id"))
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

func getValidAgentID(ctx *gin.Context) (string, bool) {
	agentID := strings.TrimSpace(ctx.Param("agent_id"))
	if agentID == "" {
		returnErrorResponse(ctx, http.StatusBadRequest, NewInvalidAgentIDError(ErrInvalidAgentID))
		return "", false
	}

	if _, err := uuid.Parse(agentID); err != nil {
		returnErrorResponse(ctx, http.StatusBadRequest, NewInvalidAgentIDError(err))
		return "", false
	}

	return agentID, true
}

func getValidGlobalAgentID(ctx *gin.Context) (string, bool) {
	agentID := strings.TrimSpace(ctx.Param("agent_id"))
	if agentID == "" {
		returnErrorResponse(ctx, http.StatusBadRequest, NewInvalidAgentIDError(ErrInvalidAgentID))
		return "", false
	}

	if _, err := uuid.Parse(agentID); err != nil {
		returnErrorResponse(ctx, http.StatusBadRequest, NewInvalidAgentIDError(err))
		return "", false
	}

	return agentID, true
}

func returnErrorResponse(ctx *gin.Context, statusCode int, err error) {
	if agentErr, ok := errors.AsType[AgentError](err); ok {
		ctx.JSON(statusCode, gin.H{"error": agentErr.Message})
		return
	}

	ctx.JSON(http.StatusInternalServerError, gin.H{"error": http.StatusText(http.StatusInternalServerError)})
}

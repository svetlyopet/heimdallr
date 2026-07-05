package token

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/svetlyopet/heimdallr/internal/auth"
)

type Handler interface {
	List(ctx *gin.Context)
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
}

type handler struct {
	service Service
}

func (h handler) List(ctx *gin.Context) {
	tokens, err := h.service.List(ctx.Request.Context())
	if err != nil {
		returnErrorResponse(ctx, http.StatusInternalServerError, NewTokenError(ErrListTokens.Error(), err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": tokens})
}

func (h handler) Create(ctx *gin.Context) {
	var req CreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		returnErrorResponse(ctx, http.StatusBadRequest, NewTokenError("invalid request body", err))
		return
	}

	var createdBy *uuid.UUID
	if user, err := userFromContext(ctx); err == nil {
		if parsed, parseErr := uuid.Parse(user.ID); parseErr == nil {
			createdBy = &parsed
		}
	}

	token, err := h.service.Create(ctx.Request.Context(), req, createdBy)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, ErrInvalidScopes) {
			statusCode = http.StatusBadRequest
		}

		returnErrorResponse(ctx, statusCode, NewTokenError(ErrCreateToken.Error(), err))
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"data": token})
}

func (h handler) Delete(ctx *gin.Context) {
	tokenID := ctx.Param("token_id")
	if tokenID == "" {
		returnErrorResponse(ctx, http.StatusBadRequest, NewTokenError("invalid token id", ErrTokenNotFound))
		return
	}

	if err := h.service.Delete(ctx.Request.Context(), tokenID); err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, ErrTokenNotFound) {
			statusCode = http.StatusNotFound
		}

		returnErrorResponse(ctx, statusCode, NewTokenError(ErrDeleteToken.Error(), err))
		return
	}

	ctx.Status(http.StatusNoContent)
}

func NewHandler(service Service) (Handler, error) {
	return &handler{service: service}, nil
}

func userFromContext(ctx *gin.Context) (auth.GetResponse, error) {
	value, exists := ctx.Get("auth.user")
	if !exists {
		return auth.GetResponse{}, auth.ErrInvalidCredentials
	}

	user, ok := value.(auth.GetResponse)
	if !ok {
		return auth.GetResponse{}, auth.ErrInvalidCredentials
	}

	return user, nil
}

func returnErrorResponse(ctx *gin.Context, statusCode int, err error) {
	if tokenErr, ok := errors.AsType[TokenError](err); ok {
		ctx.JSON(statusCode, gin.H{"error": tokenErr.Message})
		return
	}

	ctx.JSON(http.StatusInternalServerError, gin.H{"error": http.StatusText(http.StatusInternalServerError)})
}

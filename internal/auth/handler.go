package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler interface {
	List(ctx *gin.Context)
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
}

type handler struct {
	service Service
}

func (h handler) List(ctx *gin.Context) {
	users, err := h.service.List(ctx.Request.Context())
	if err != nil {
		returnErrorResponse(ctx, http.StatusInternalServerError, NewListUsersError(err))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": users})
}

func (h handler) Create(ctx *gin.Context) {
	var req CreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		authErr := NewAuthError("invalid request body", err)
		returnErrorResponse(ctx, http.StatusBadRequest, authErr)
		return
	}

	user, err := h.service.Create(ctx.Request.Context(), req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrUserAlreadyExists):
			statusCode = http.StatusConflict
		case errors.Is(err, ErrInvalidRole), errors.Is(err, ErrInvalidPasswordValue), errors.Is(err, ErrInvalidCredentials):
			statusCode = http.StatusBadRequest
		}

		authErr := NewCreateUserError(err)
		returnErrorResponse(ctx, statusCode, authErr)
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"data": user})
}

func (h handler) Update(ctx *gin.Context) {
	userID := ctx.Param("user_id")
	if userID == "" {
		authErr := NewInvalidUserIDError(ErrInvalidUserID)
		returnErrorResponse(ctx, http.StatusBadRequest, authErr)
		return
	}

	var req UpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		authErr := NewAuthError("invalid request body", err)
		returnErrorResponse(ctx, http.StatusBadRequest, authErr)
		return
	}

	user, err := h.service.Update(ctx.Request.Context(), userID, req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrUserNotFound):
			statusCode = http.StatusNotFound
		case errors.Is(err, ErrInvalidRole), errors.Is(err, ErrInvalidPasswordValue), errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrInvalidUserID), errors.Is(err, ErrRootRoleForbidden):
			statusCode = http.StatusBadRequest
		}

		authErr := NewUpdateUserError(err)
		returnErrorResponse(ctx, statusCode, authErr)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": user})
}

func (h handler) Delete(ctx *gin.Context) {
	userID := ctx.Param("user_id")
	if userID == "" {
		authErr := NewInvalidUserIDError(ErrInvalidUserID)
		returnErrorResponse(ctx, http.StatusBadRequest, authErr)
		return
	}

	if err := h.service.Delete(ctx.Request.Context(), userID); err != nil {
		statusCode := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrUserNotFound):
			statusCode = http.StatusNotFound
		case errors.Is(err, ErrInvalidUserID), errors.Is(err, ErrRootDeleteForbidden):
			statusCode = http.StatusBadRequest
		}

		authErr := NewDeleteUserError(err)
		returnErrorResponse(ctx, statusCode, authErr)
		return
	}

	ctx.Status(http.StatusNoContent)
}

func NewHandler(service Service) (Handler, error) {
	return &handler{service: service}, nil
}

func returnErrorResponse(ctx *gin.Context, statusCode int, err error) {
	if authErr, ok := errors.AsType[AuthError](err); ok {
		ctx.JSON(statusCode, gin.H{"error": authErr.Message})
		return
	}

	ctx.JSON(http.StatusInternalServerError, gin.H{"error": http.StatusText(http.StatusInternalServerError)})
}

package rbac

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RequireRole(authorizer Authorizer, roles ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user, err := userFromGinContext(ctx)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": ErrUnauthorized.Error()})
			return
		}

		if !authorizer.HasAnyRole(user, roles...) {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": ErrInsufficientRole.Error()})
			return
		}

		ctx.Next()
	}
}

func RequireScope(authorizer Authorizer, scope string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user, err := userFromGinContext(ctx)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": ErrUnauthorized.Error()})
			return
		}

		if !authorizer.HasScope(user, scope) {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": ErrInsufficientScope.Error()})
			return
		}

		ctx.Next()
	}
}

func StrictHandlerErrorFunc(ctx *gin.Context, err error) {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		ctx.JSON(httpErr.Status, gin.H{"error": httpErr.Message})
		return
	}

	slog.ErrorContext(
		ctx.Request.Context(),
		"strict handler failed",
		slog.Any("error", err),
		slog.String("method", ctx.Request.Method),
		slog.String("route", ctx.FullPath()),
	)
	ctx.JSON(http.StatusInternalServerError, gin.H{"error": http.StatusText(http.StatusInternalServerError)})
}

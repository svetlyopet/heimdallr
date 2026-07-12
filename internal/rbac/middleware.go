package rbac

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/logger"
)

const OperationIDKey = "operation_id"

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

func NewStrictHandlerErrorFunc(appLogger *logger.Logger) func(*gin.Context, error) {
	if appLogger == nil {
		appLogger = logger.Default()
	}

	return func(ctx *gin.Context, err error) {
		var httpErr *HTTPError
		if errors.As(err, &httpErr) {
			ctx.JSON(httpErr.Status, gin.H{"error": httpErr.Message})
			return
		}

		attrs := []slog.Attr{
			slog.String("method", ctx.Request.Method),
			slog.String("route", ctx.FullPath()),
		}
		if operationID, ok := ctx.Get(OperationIDKey); ok {
			if id, isString := operationID.(string); isString && id != "" {
				attrs = append(attrs, slog.String("operation_id", id))
			}
		}

		appLogger.ErrorWithStack(ctx.Request.Context(), "strict handler failed", err, attrs...)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": http.StatusText(http.StatusInternalServerError)})
	}
}

func StrictHandlerErrorFunc(ctx *gin.Context, err error) {
	NewStrictHandlerErrorFunc(logger.Default())(ctx, err)
}

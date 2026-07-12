package rbac

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

func StrictScopeMiddleware(authorizer Authorizer, policies map[string]string) func(func(*gin.Context, interface{}) (interface{}, error), string) func(*gin.Context, interface{}) (interface{}, error) {
	return func(f func(*gin.Context, interface{}) (interface{}, error), operationID string) func(*gin.Context, interface{}) (interface{}, error) {
		requiredScope, ok := policies[operationID]
		if !ok {
			return func(ctx *gin.Context, _ interface{}) (interface{}, error) {
				logContext := context.Background()
				if ctx.Request != nil {
					logContext = ctx.Request.Context()
				}
				slog.ErrorContext(
					logContext,
					"authorization policy is not configured",
					slog.String("operation_id", operationID),
				)
				return nil, &HTTPError{
					Status:  http.StatusInternalServerError,
					Message: http.StatusText(http.StatusInternalServerError),
					Err:     ErrPolicyNotConfigured,
				}
			}
		}

		return func(ctx *gin.Context, request interface{}) (interface{}, error) {
			user, err := userFromGinContext(ctx)
			if err != nil {
				return nil, unauthorizedError()
			}

			if !authorizer.HasScope(user, requiredScope) {
				return nil, forbiddenScopeError()
			}

			return f(ctx, request)
		}
	}
}

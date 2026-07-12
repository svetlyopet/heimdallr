package rbac

import (
	"github.com/gin-gonic/gin"
)

func StrictScopeMiddleware(authorizer Authorizer, policies map[string]string) func(func(*gin.Context, interface{}) (interface{}, error), string) func(*gin.Context, interface{}) (interface{}, error) {
	return func(f func(*gin.Context, interface{}) (interface{}, error), operationID string) func(*gin.Context, interface{}) (interface{}, error) {
		requiredScope, ok := policies[operationID]
		if !ok {
			return f
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

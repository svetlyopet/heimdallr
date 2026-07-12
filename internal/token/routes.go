package token

import (
	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/rbac"
	"github.com/svetlyopet/heimdallr/internal/token/api"
)

var Policies = map[string]string{
	"ListTokens":  rbac.ScopeAdmin,
	"CreateToken": rbac.ScopeAdmin,
	"DeleteToken": rbac.ScopeAdmin,
}

func RegisterRoutes(rg *gin.RouterGroup, handler Handler, authorizer rbac.Authorizer, appLogger *logger.Logger) {
	adminGroup := rg.Group("")
	adminGroup.Use(rbac.RequireRole(authorizer, rbac.RoleAdmin))

	scopeMiddleware := func(next api.StrictHandlerFunc, operationID string) api.StrictHandlerFunc {
		return rbac.StrictScopeMiddleware(authorizer, Policies)(next, operationID)
	}

	strictHandler := api.NewStrictHandlerWithOptions(handler, []api.StrictMiddlewareFunc{scopeMiddleware}, api.StrictGinServerOptions{
		HandlerErrorFunc: rbac.NewStrictHandlerErrorFunc(appLogger),
	})
	api.RegisterHandlersWithOptions(adminGroup, strictHandler, api.GinServerOptions{})
}

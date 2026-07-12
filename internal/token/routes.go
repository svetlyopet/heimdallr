package token

import (
	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/rbac"
	"github.com/svetlyopet/heimdallr/internal/token/api"
)

func RegisterRoutes(rg *gin.RouterGroup, handler Handler, authorizer rbac.Authorizer) {
	adminGroup := rg.Group("")
	adminGroup.Use(rbac.RequireRole(authorizer, rbac.RoleAdmin))

	policies := map[string]string{
		"ListTokens":  rbac.ScopeAdmin,
		"CreateToken": rbac.ScopeAdmin,
		"DeleteToken": rbac.ScopeAdmin,
	}

	scopeMiddleware := func(next api.StrictHandlerFunc, operationID string) api.StrictHandlerFunc {
		return rbac.StrictScopeMiddleware(authorizer, policies)(next, operationID)
	}

	strictHandler := api.NewStrictHandlerWithOptions(handler, []api.StrictMiddlewareFunc{scopeMiddleware}, api.StrictGinServerOptions{
		HandlerErrorFunc: rbac.StrictHandlerErrorFunc,
	})
	api.RegisterHandlersWithOptions(adminGroup, strictHandler, api.GinServerOptions{})
}

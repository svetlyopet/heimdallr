package application

import (
	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/application/api"
	"github.com/svetlyopet/heimdallr/internal/rbac"
)

func RegisterRoutes(rg *gin.RouterGroup, handler Handler, authorizer rbac.Authorizer) {
	policies := map[string]string{
		"ListApplications":  rbac.ScopeRead,
		"CreateApplication": rbac.ScopeApplicationWrite,
		"GetApplication":    rbac.ScopeRead,
	}

	scopeMiddleware := func(next api.StrictHandlerFunc, operationID string) api.StrictHandlerFunc {
		return rbac.StrictScopeMiddleware(authorizer, policies)(next, operationID)
	}

	strictHandler := api.NewStrictHandlerWithOptions(handler, []api.StrictMiddlewareFunc{scopeMiddleware}, api.StrictGinServerOptions{
		HandlerErrorFunc: rbac.StrictHandlerErrorFunc,
	})
	api.RegisterHandlersWithOptions(rg, strictHandler, api.GinServerOptions{})
}

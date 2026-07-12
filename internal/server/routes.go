package server

import (
	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/rbac"
	"github.com/svetlyopet/heimdallr/internal/server/api"
)

func RegisterRoutes(rg *gin.RouterGroup, handler Handler, authorizer rbac.Authorizer) {
	policies := map[string]string{
		"ListServers":             rbac.ScopeRead,
		"CreateServer":            rbac.ScopeAutomationWrite,
		"GetServer":               rbac.ScopeRead,
		"UpdateServer":            rbac.ScopeAutomationWrite,
		"ListServerJobs":          rbac.ScopeRead,
		"AssociateServerJob":      rbac.ScopeAutomationWrite,
		"DissociateServerJob":     rbac.ScopeAutomationWrite,
		"ListServerReleases":      rbac.ScopeRead,
		"AssociateServerRelease":  rbac.ScopeAutomationWrite,
		"DissociateServerRelease": rbac.ScopeAutomationWrite,
	}

	scopeMiddleware := func(next api.StrictHandlerFunc, operationID string) api.StrictHandlerFunc {
		return rbac.StrictScopeMiddleware(authorizer, policies)(next, operationID)
	}

	strictHandler := api.NewStrictHandlerWithOptions(handler, []api.StrictMiddlewareFunc{scopeMiddleware}, api.StrictGinServerOptions{
		HandlerErrorFunc: rbac.StrictHandlerErrorFunc,
	})
	api.RegisterHandlersWithOptions(rg, strictHandler, api.GinServerOptions{})
}

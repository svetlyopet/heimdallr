package server

import (
	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/rbac"
	"github.com/svetlyopet/heimdallr/internal/server/api"
)

var Policies = map[string]string{
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

func RegisterRoutes(rg *gin.RouterGroup, handler Handler, authorizer rbac.Authorizer) {
	scopeMiddleware := func(next api.StrictHandlerFunc, operationID string) api.StrictHandlerFunc {
		return rbac.StrictScopeMiddleware(authorizer, Policies)(next, operationID)
	}

	strictHandler := api.NewStrictHandlerWithOptions(handler, []api.StrictMiddlewareFunc{scopeMiddleware}, api.StrictGinServerOptions{
		HandlerErrorFunc: rbac.StrictHandlerErrorFunc,
	})
	api.RegisterHandlersWithOptions(rg, strictHandler, api.GinServerOptions{})
}

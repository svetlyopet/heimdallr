package agent

import (
	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/agent/api"
	"github.com/svetlyopet/heimdallr/internal/rbac"
)

func RegisterRoutes(rg *gin.RouterGroup, handler Handler, authorizer rbac.Authorizer) {
	policies := map[string]string{
		"ListGlobalAgents":      rbac.ScopeRead,
		"CreateUnassignedAgent": rbac.ScopeAutomationWrite,
		"DeleteGlobalAgent":     rbac.ScopeAutomationWrite,
		"GetGlobalAgent":        rbac.ScopeRead,
		"ListAgentServers":      rbac.ScopeRead,
		"ListAgents":            rbac.ScopeRead,
		"CreateAgent":           rbac.ScopeAutomationWrite,
		"DetachAgent":           rbac.ScopeAutomationWrite,
		"GetAgent":              rbac.ScopeRead,
	}

	scopeMiddleware := func(next api.StrictHandlerFunc, operationID string) api.StrictHandlerFunc {
		return rbac.StrictScopeMiddleware(authorizer, policies)(next, operationID)
	}

	strictHandler := api.NewStrictHandlerWithOptions(handler, []api.StrictMiddlewareFunc{scopeMiddleware}, api.StrictGinServerOptions{
		HandlerErrorFunc: rbac.StrictHandlerErrorFunc,
	})
	api.RegisterHandlersWithOptions(rg, strictHandler, api.GinServerOptions{})
}

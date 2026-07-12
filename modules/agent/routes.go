package agent

import (
	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/rbac"
	"github.com/svetlyopet/heimdallr/modules/agent/api"
)

var Policies = map[string]string{
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

func RegisterRoutes(rg *gin.RouterGroup, handler Handler, authorizer rbac.Authorizer, appLogger *logger.Logger) {
	scopeMiddleware := func(next api.StrictHandlerFunc, operationID string) api.StrictHandlerFunc {
		return rbac.StrictScopeMiddleware(authorizer, Policies)(next, operationID)
	}

	strictHandler := api.NewStrictHandlerWithOptions(handler, []api.StrictMiddlewareFunc{scopeMiddleware}, api.StrictGinServerOptions{
		HandlerErrorFunc: rbac.NewStrictHandlerErrorFunc(appLogger),
	})
	api.RegisterHandlersWithOptions(rg, strictHandler, api.GinServerOptions{})
}

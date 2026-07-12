package automation

import (
	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/automation/api"
	"github.com/svetlyopet/heimdallr/internal/rbac"
)

func RegisterRoutes(rg *gin.RouterGroup, handler Handler, authorizer rbac.Authorizer) {
	policies := map[string]string{
		"ListAutomations":  rbac.ScopeRead,
		"CreateAutomation": rbac.ScopeAutomationWrite,
		"DeleteAutomation": rbac.ScopeAutomationWrite,
		"GetAutomation":    rbac.ScopeRead,
		"UpdateAutomation": rbac.ScopeAutomationWrite,
	}

	scopeMiddleware := func(next api.StrictHandlerFunc, operationID string) api.StrictHandlerFunc {
		return rbac.StrictScopeMiddleware(authorizer, policies)(next, operationID)
	}

	strictHandler := api.NewStrictHandlerWithOptions(handler, []api.StrictMiddlewareFunc{scopeMiddleware}, api.StrictGinServerOptions{
		HandlerErrorFunc: rbac.StrictHandlerErrorFunc,
	})
	api.RegisterHandlersWithOptions(rg, strictHandler, api.GinServerOptions{})
}

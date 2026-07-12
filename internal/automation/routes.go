package automation

import (
	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/automation/api"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/rbac"
)

var Policies = map[string]string{
	"ListAutomations":  rbac.ScopeRead,
	"CreateAutomation": rbac.ScopeAutomationWrite,
	"DeleteAutomation": rbac.ScopeAutomationWrite,
	"GetAutomation":    rbac.ScopeRead,
	"UpdateAutomation": rbac.ScopeAutomationWrite,
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

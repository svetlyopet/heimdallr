package job

import (
	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/job/api"
	"github.com/svetlyopet/heimdallr/internal/rbac"
)

var Policies = map[string]string{
	"ListAutomationJobs":  rbac.ScopeRead,
	"CreateAutomationJob": rbac.ScopeAutomationWrite,
	"GetAutomationJob":    rbac.ScopeRead,
	"UpdateAutomationJob": rbac.ScopeAutomationWrite,
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

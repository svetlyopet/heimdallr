package analytics

import (
	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/analytics/api"
	"github.com/svetlyopet/heimdallr/internal/rbac"
)

func RegisterRoutes(rg *gin.RouterGroup, handler Handler, authorizer rbac.Authorizer) {
	policies := map[string]string{
		"GetAutomationAnalyticsOverview":       rbac.ScopeRead,
		"GetAutomationAnalyticsOverviewByID":   rbac.ScopeRead,
		"GetComplianceAnalyticsOverview":     rbac.ScopeRead,
	}

	scopeMiddleware := func(next api.StrictHandlerFunc, operationID string) api.StrictHandlerFunc {
		return rbac.StrictScopeMiddleware(authorizer, policies)(next, operationID)
	}

	strictHandler := api.NewStrictHandlerWithOptions(handler, []api.StrictMiddlewareFunc{scopeMiddleware}, api.StrictGinServerOptions{
		HandlerErrorFunc: rbac.StrictHandlerErrorFunc,
	})
	api.RegisterHandlersWithOptions(rg, strictHandler, api.GinServerOptions{})
}

package report

import (
	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/rbac"
	"github.com/svetlyopet/heimdallr/internal/report/api"
)

func RegisterRoutes(rg *gin.RouterGroup, handler Handler, authorizer rbac.Authorizer) {
	policies := map[string]string{
		"ListReleaseReports":  rbac.ScopeRead,
		"CreateReleaseReport": rbac.ScopeApplicationWrite,
		"GetReleaseReport":    rbac.ScopeRead,
		"UpdateReleaseReport": rbac.ScopeApplicationWrite,
		"ListReportsGlobal":   rbac.ScopeRead,
	}

	scopeMiddleware := func(next api.StrictHandlerFunc, operationID string) api.StrictHandlerFunc {
		return rbac.StrictScopeMiddleware(authorizer, policies)(next, operationID)
	}

	strictHandler := api.NewStrictHandlerWithOptions(handler, []api.StrictMiddlewareFunc{scopeMiddleware}, api.StrictGinServerOptions{
		HandlerErrorFunc: rbac.StrictHandlerErrorFunc,
	})
	api.RegisterHandlersWithOptions(rg, strictHandler, api.GinServerOptions{})
}

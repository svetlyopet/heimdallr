package report

import (
	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"github.com/svetlyopet/heimdallr/internal/modules/report/api"
	"github.com/svetlyopet/heimdallr/internal/rbac"
)

var Policies = map[string]string{
	"ListReleaseReports":  rbac.ScopeRead,
	"CreateReleaseReport": rbac.ScopeApplicationWrite,
	"GetReleaseReport":    rbac.ScopeRead,
	"UpdateReleaseReport": rbac.ScopeApplicationWrite,
	"ListReportsGlobal":   rbac.ScopeRead,
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

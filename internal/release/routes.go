package release

import (
	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/rbac"
	"github.com/svetlyopet/heimdallr/internal/release/api"
)

var Policies = map[string]string{
	"ListReleases":  rbac.ScopeRead,
	"CreateRelease": rbac.ScopeApplicationWrite,
	"GetRelease":    rbac.ScopeRead,
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

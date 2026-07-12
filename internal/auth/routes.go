package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/auth/api"
	"github.com/svetlyopet/heimdallr/internal/rbac"
)

func RegisterPublicRoutes(rg *gin.RouterGroup, handler Handler, loginRateLimiter *LoginRateLimiter) {
	strictHandler := api.NewStrictHandler(handler, nil)
	wrapper := api.ServerInterfaceWrapper{Handler: strictHandler}

	loginGroup := rg.Group("")
	if loginRateLimiter != nil {
		loginGroup.Use(loginRateLimiter.Middleware())
	}

	loginGroup.POST("/v1/auth/login", wrapper.Login)
}

func RegisterProtectedRoutes(rg *gin.RouterGroup, handler Handler, authorizer rbac.Authorizer) {
	adminGroup := rg.Group("")
	adminGroup.Use(rbac.RequireRole(authorizer, rbac.RoleAdmin))

	policies := map[string]string{
		"ListUsers":  rbac.ScopeAdmin,
		"CreateUser": rbac.ScopeAdmin,
		"UpdateUser": rbac.ScopeAdmin,
		"DeleteUser": rbac.ScopeAdmin,
	}

	scopeMiddleware := func(next api.StrictHandlerFunc, operationID string) api.StrictHandlerFunc {
		return rbac.StrictScopeMiddleware(authorizer, policies)(next, operationID)
	}

	strictHandler := api.NewStrictHandlerWithOptions(handler, []api.StrictMiddlewareFunc{scopeMiddleware}, api.StrictGinServerOptions{
		HandlerErrorFunc: rbac.StrictHandlerErrorFunc,
	})
	wrapper := api.ServerInterfaceWrapper{Handler: strictHandler}

	adminGroup.GET("/v1/auth/users", wrapper.ListUsers)
	adminGroup.POST("/v1/auth/users", wrapper.CreateUser)
	adminGroup.DELETE("/v1/auth/users/:user_id", wrapper.DeleteUser)
	adminGroup.PUT("/v1/auth/users/:user_id", wrapper.UpdateUser)
}

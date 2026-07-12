package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/auth/api"
	"github.com/svetlyopet/heimdallr/internal/rbac"
)

var Policies = map[string]string{
	"ListUsers":  rbac.ScopeAdmin,
	"CreateUser": rbac.ScopeAdmin,
	"UpdateUser": rbac.ScopeAdmin,
	"DeleteUser": rbac.ScopeAdmin,
}

func RegisterPublicRoutes(rg *gin.RouterGroup, handler Handler, loginRateLimiter *LoginRateLimiter, cookieConfigs ...CookieConfig) {
	cookieConfig := CookieConfig{}
	if len(cookieConfigs) > 0 {
		cookieConfig = cookieConfigs[0]
	}

	strictHandler := api.NewStrictHandler(handler, []api.StrictMiddlewareFunc{
		SessionCookieMiddleware(cookieConfig),
	})
	wrapper := api.ServerInterfaceWrapper{Handler: strictHandler}

	loginGroup := rg.Group("")
	if loginRateLimiter != nil {
		loginGroup.Use(loginRateLimiter.Middleware())
	}

	loginGroup.POST("/v1/auth/login", wrapper.Login)
}

func RegisterProtectedRoutes(rg *gin.RouterGroup, handler Handler, authorizer rbac.Authorizer, cookieConfigs ...CookieConfig) {
	cookieConfig := CookieConfig{}
	if len(cookieConfigs) > 0 {
		cookieConfig = cookieConfigs[0]
	}

	adminGroup := rg.Group("")
	adminGroup.Use(rbac.RequireRole(authorizer, rbac.RoleAdmin))

	scopeMiddleware := func(next api.StrictHandlerFunc, operationID string) api.StrictHandlerFunc {
		return rbac.StrictScopeMiddleware(authorizer, Policies)(next, operationID)
	}

	strictHandler := api.NewStrictHandlerWithOptions(handler, []api.StrictMiddlewareFunc{scopeMiddleware}, api.StrictGinServerOptions{
		HandlerErrorFunc: rbac.StrictHandlerErrorFunc,
	})
	wrapper := api.ServerInterfaceWrapper{Handler: strictHandler}

	logoutHandler := api.NewStrictHandlerWithOptions(handler, []api.StrictMiddlewareFunc{
		SessionCookieMiddleware(cookieConfig),
	}, api.StrictGinServerOptions{
		HandlerErrorFunc: rbac.StrictHandlerErrorFunc,
	})
	logoutWrapper := api.ServerInterfaceWrapper{Handler: logoutHandler}

	rg.POST("/v1/auth/logout", logoutWrapper.Logout)
	adminGroup.GET("/v1/auth/users", wrapper.ListUsers)
	adminGroup.POST("/v1/auth/users", wrapper.CreateUser)
	adminGroup.DELETE("/v1/auth/users/:user_id", wrapper.DeleteUser)
	adminGroup.PUT("/v1/auth/users/:user_id", wrapper.UpdateUser)
}

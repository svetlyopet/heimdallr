package testutil

import (
	"github.com/gin-gonic/gin"
	authapi "github.com/svetlyopet/heimdallr/internal/auth/api"
	"github.com/svetlyopet/heimdallr/internal/rbac"
)

func AuthenticatedAdminMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Set("auth.user", authapi.AuthUser{
			Roles: []authapi.AuthRole{
				authapi.Admin,
				authapi.Reader,
				authapi.AuthRole(rbac.ScopeRead),
				authapi.AuthRole(rbac.ScopeApplicationWrite),
				authapi.AuthRole(rbac.ScopeAutomationWrite),
				authapi.AuthRole(rbac.ScopeAdmin),
			},
		})
		ctx.Next()
	}
}

func AuthenticatedReaderMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Set("auth.user", authapi.AuthUser{
			Roles: []authapi.AuthRole{
				authapi.Reader,
				authapi.AuthRole(rbac.ScopeRead),
			},
		})
		ctx.Next()
	}
}

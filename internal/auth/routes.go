package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/auth/api"
)

func RegisterPublicRoutes(rg *gin.RouterGroup, handler Handler) {
	strictHandler := api.NewStrictHandler(handler, nil)
	api.RegisterHandlersWithOptions(rg, strictHandler, api.GinServerOptions{})
}

func RequireRoles(service Service, roles ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user, err := UserFromGinContext(ctx)
		if err != nil {
			ctx.AbortWithStatusJSON(401, gin.H{"error": ErrInvalidCredentials.Error()})
			return
		}

		if !service.HasAnyRole(user, roles...) {
			ctx.AbortWithStatusJSON(403, gin.H{"error": ErrInsufficientRole.Error()})
			return
		}

		ctx.Next()
	}
}

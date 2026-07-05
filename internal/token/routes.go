package token

import (
	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/auth"
)

func RegisterRoutes(rg *gin.RouterGroup, handler Handler, authService auth.Service) {
	tokenRoutesV1 := rg.Group("/v1/auth/tokens")
	tokenRoutesV1.Use(auth.RequireRoles(authService, auth.RoleAdmin))
	{
		tokenRoutesV1.GET("", handler.List)
		tokenRoutesV1.POST("", handler.Create)
		tokenRoutesV1.DELETE("/:token_id", handler.Delete)
	}
}

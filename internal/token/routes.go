package token

import (
	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/auth"
	"github.com/svetlyopet/heimdallr/internal/token/api"
)

func RegisterRoutes(rg *gin.RouterGroup, handler Handler, authService auth.Service) {
	adminGroup := rg.Group("")
	adminGroup.Use(auth.RequireRoles(authService, auth.RoleAdmin))
	strictHandler := api.NewStrictHandler(handler, nil)
	api.RegisterHandlersWithOptions(adminGroup, strictHandler, api.GinServerOptions{})
}

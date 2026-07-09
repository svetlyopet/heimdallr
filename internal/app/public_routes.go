package app

import (
	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/auth"
)

func (a *App) RegisterPublicRoutes(rg *gin.RouterGroup) {
	auth.RegisterPublicRoutes(rg, a.authHandler)
}

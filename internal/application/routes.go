package application

import (
	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/application/api"
)

func RegisterRoutes(rg *gin.RouterGroup, handler Handler) {
	strictHandler := api.NewStrictHandler(handler, nil)
	api.RegisterHandlersWithOptions(rg, strictHandler, api.GinServerOptions{})
}

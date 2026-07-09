package release

import (
	"github.com/gin-gonic/gin"
	"github.com/svetlyopet/heimdallr/internal/release/api"
)

func RegisterRoutes(rg *gin.RouterGroup, handler Handler) {
	strictHandler := api.NewStrictHandler(handler, nil)
	api.RegisterHandlersWithOptions(rg, strictHandler, api.GinServerOptions{})
}

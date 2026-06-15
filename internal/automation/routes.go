package automation

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup, handler Handler) {
	automationRoutesV1 := rg.Group("/v1/automation")
	{
		automationRoutesV1.GET("", handler.List)
		automationRoutesV1.GET("/:automation_id", handler.Get)
		automationRoutesV1.POST("", handler.Create)
		automationRoutesV1.PUT("/:automation_id", handler.Update)
		automationRoutesV1.DELETE("/:automation_id", handler.Delete)
	}

}

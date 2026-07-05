package application

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, handler Handler) {
	applicationRoutesV1 := rg.Group("/v1/application")
	{
		applicationRoutesV1.GET("", handler.List)
		applicationRoutesV1.POST("", handler.Create)
		applicationRoutesV1.GET("/:application_id", handler.Get)
	}
}

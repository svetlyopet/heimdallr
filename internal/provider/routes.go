package provider

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, handler Handler) {
	providerRoutesV1 := rg.Group("/v1/provider")
	{
		providerRoutesV1.GET("", handler.List)
		providerRoutesV1.GET("/:provider_id", handler.Get)
		providerRoutesV1.POST("", handler.Create)
	}
}

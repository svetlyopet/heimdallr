package release

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, handler Handler) {
	releaseRoutesV1 := rg.Group("/v1/application/:application_id/release")
	{
		releaseRoutesV1.GET("", handler.List)
		releaseRoutesV1.POST("", handler.Create)
		releaseRoutesV1.GET("/:release_id", handler.Get)
	}
}

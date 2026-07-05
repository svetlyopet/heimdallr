package report

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, handler Handler) {
	reportRoutesV1 := rg.Group("/v1/application/:application_id/release/:release_id/report")
	{
		reportRoutesV1.GET("", handler.List)
		reportRoutesV1.POST("", handler.Create)
		reportRoutesV1.GET("/:report_id", handler.Get)
		reportRoutesV1.PATCH("/:report_id", handler.Update)
	}
}

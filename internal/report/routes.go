package report

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, handler Handler) {
	reportRoutesV1 := rg.Group("/v1/report")
	{
		reportRoutesV1.GET("", handler.ListAll)
	}

	applicationReportRoutesV1 := rg.Group("/v1/application/:application_id/release/:release_id/report")
	{
		applicationReportRoutesV1.GET("", handler.List)
		applicationReportRoutesV1.POST("", handler.Create)
		applicationReportRoutesV1.GET("/:report_id", handler.Get)
		applicationReportRoutesV1.PATCH("/:report_id", handler.Update)
	}
}

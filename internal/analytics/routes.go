package analytics

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, handler Handler) {
	analyticsRoutesV1 := rg.Group("/v1/analytics")
	{
		analyticsRoutesV1.GET("/automation", handler.GetAutomationOverview)
		analyticsRoutesV1.GET("/automation/:id", handler.GetAutomationOverviewByID)
	}
}

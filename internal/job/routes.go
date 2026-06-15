package job

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, handler Handler) {
	jobRoutesV1 := rg.Group("/v1/automation/:automation_id/job")
	{
		jobRoutesV1.GET("", handler.List)
		jobRoutesV1.POST("", handler.Create)
		jobRoutesV1.GET("/:job_id", handler.Get)
		jobRoutesV1.PATCH("/:job_id", handler.Update)
	}
}

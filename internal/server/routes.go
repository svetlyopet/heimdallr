package server

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, handler Handler) {
	serverRoutesV1 := rg.Group("/v1/server")
	{
		serverRoutesV1.GET("", handler.List)
		serverRoutesV1.POST("", handler.Create)
		serverRoutesV1.GET("/:server_id", handler.Get)
		serverRoutesV1.PUT("/:server_id", handler.Update)

		jobRoutes := serverRoutesV1.Group("/:server_id/job")
		{
			jobRoutes.GET("", handler.ListJobs)
			jobRoutes.POST("", handler.AssociateJob)
			jobRoutes.DELETE("/:job_id", handler.DissociateJob)
		}

		releaseRoutes := serverRoutesV1.Group("/:server_id/release")
		{
			releaseRoutes.GET("", handler.ListReleases)
			releaseRoutes.POST("", handler.AssociateRelease)
			releaseRoutes.DELETE("/:release_id", handler.DissociateRelease)
		}
	}
}

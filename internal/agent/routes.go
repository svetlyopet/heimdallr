package agent

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, handler Handler) {
	agentRoutesV1 := rg.Group("/v1/server/:server_id/agent")
	{
		agentRoutesV1.GET("", handler.List)
		agentRoutesV1.POST("", handler.Create)
		agentRoutesV1.GET("/:agent_id", handler.Get)
		agentRoutesV1.DELETE("/:agent_id", handler.Delete)
	}
}

func RegisterGlobalRoutes(rg *gin.RouterGroup, handler Handler) {
	globalRoutesV1 := rg.Group("/v1/agent")
	{
		globalRoutesV1.GET("", handler.ListGlobal)
		globalRoutesV1.POST("", handler.CreateUnassigned)
		globalRoutesV1.GET("/:agent_id", handler.GetGlobal)
	}
}

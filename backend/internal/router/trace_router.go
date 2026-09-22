package router

import (
	"fiber-otdr-fault-localization/backend/internal/constants"
	appmw "fiber-otdr-fault-localization/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

func registerTraceRoutes(api *gin.RouterGroup, deps Dependencies) {
	traces := api.Group("/traces")
	traces.GET("", deps.TraceHandler.List)
	traces.GET("/:id", deps.TraceHandler.Get)
	traces.POST("/import", appmw.RateLimitMiddleware(deps.ImportLimiter, "trace_import"), appmw.RBACMiddleware(constants.RoleAnalyst, constants.RoleAdmin), deps.TraceHandler.Import)
	traces.POST("/:id/detect", appmw.RateLimitMiddleware(deps.AnalyzeLimiter, "event_detection"), appmw.RBACMiddleware(constants.RoleAnalyst, constants.RoleAdmin), deps.EventHandler.Detect)
}

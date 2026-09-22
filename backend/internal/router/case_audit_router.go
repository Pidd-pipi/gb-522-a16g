package router

import (
	"fiber-otdr-fault-localization/backend/internal/constants"
	appmw "fiber-otdr-fault-localization/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

func registerCaseAuditRoutes(api *gin.RouterGroup, deps Dependencies) {
	cases := api.Group("/cases")
	cases.GET("", deps.CaseHandler.List)
	cases.GET("/:id", deps.CaseHandler.Get)
	cases.POST("", appmw.RBACMiddleware(constants.RoleAnalyst, constants.RoleAdmin), deps.CaseHandler.Create)
	cases.POST("/:id/analyze", appmw.RateLimitMiddleware(deps.AnalyzeLimiter, "case_analysis"), appmw.RBACMiddleware(constants.RoleAnalyst, constants.RoleAdmin), deps.CaseHandler.Analyze)
	cases.POST("/:id/confirm", appmw.RBACMiddleware(constants.RoleReviewer, constants.RoleAdmin), deps.CaseHandler.Confirm)
	cases.POST("/:id/close", appmw.RBACMiddleware(constants.RoleReviewer, constants.RoleAdmin), deps.CaseHandler.Close)
	audit := api.Group("/audit")
	audit.GET("", appmw.RBACMiddleware(constants.RoleReviewer, constants.RoleAdmin), deps.AuditHandler.List)
}

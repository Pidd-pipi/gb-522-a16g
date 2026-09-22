package router

import (
	"fiber-otdr-fault-localization/backend/internal/handler"
	appmw "fiber-otdr-fault-localization/backend/internal/middleware"
	"fiber-otdr-fault-localization/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	AuthHandler    *handler.AuthHandler
	RouteHandler   *handler.FiberRouteHandler
	TraceHandler   *handler.TraceHandler
	EventHandler   *handler.EventHandler
	CaseHandler    *handler.CaseHandler
	AuditHandler   *handler.AuditHandler
	AuthService    *service.AuthService
	LoginLimiter   *appmw.RateLimiter
	ImportLimiter  *appmw.RateLimiter
	AnalyzeLimiter *appmw.RateLimiter
}

func Register(engine *gin.Engine, deps Dependencies) {
	engine.GET("/healthz", deps.AuditHandler.Health)
	engine.GET("/readyz", deps.AuditHandler.Ready)
	api := engine.Group("/api/v1")
	api.POST("/auth/login", appmw.RateLimitMiddleware(deps.LoginLimiter, "login"), deps.AuthHandler.Login)
	protected := api.Group("")
	protected.Use(appmw.AuthMiddleware(deps.AuthService))
	registerFiberRoutes(protected, deps)
	registerTraceRoutes(protected, deps)
	registerEventRoutes(protected, deps)
	registerCaseAuditRoutes(protected, deps)
}

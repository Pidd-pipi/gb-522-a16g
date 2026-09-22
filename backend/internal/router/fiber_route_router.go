package router

import (
	"fiber-otdr-fault-localization/backend/internal/constants"
	appmw "fiber-otdr-fault-localization/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

func registerFiberRoutes(api *gin.RouterGroup, deps Dependencies) {
	routes := api.Group("/routes")
	routes.GET("", deps.RouteHandler.List)
	routes.GET("/:id", deps.RouteHandler.Get)
	routes.POST("", appmw.RBACMiddleware(constants.RoleAnalyst, constants.RoleAdmin), deps.RouteHandler.Create)
	routes.PATCH("/:id", appmw.RBACMiddleware(constants.RoleAnalyst, constants.RoleAdmin), deps.RouteHandler.Update)
	routes.POST("/:id/baseline", appmw.RBACMiddleware(constants.RoleReviewer, constants.RoleAdmin), deps.RouteHandler.SetBaseline)
}

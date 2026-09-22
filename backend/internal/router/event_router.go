package router

import (
	"fiber-otdr-fault-localization/backend/internal/constants"
	appmw "fiber-otdr-fault-localization/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

func registerEventRoutes(api *gin.RouterGroup, deps Dependencies) {
	events := api.Group("/events")
	events.GET("", deps.EventHandler.List)
	events.PATCH("/:id/review", appmw.RBACMiddleware(constants.RoleReviewer, constants.RoleAdmin), deps.EventHandler.Review)
}

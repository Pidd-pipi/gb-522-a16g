package handler

import (
	"net/http"
	"time"

	"fiber-otdr-fault-localization/backend/internal/dto"
	"fiber-otdr-fault-localization/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type AuditHandler struct{ service *service.AuditService }

func NewAuditHandler(audit *service.AuditService) *AuditHandler { return &AuditHandler{audit} }

func (h *AuditHandler) List(c *gin.Context) {
	query := dto.AuditQuery{RouteID: queryUint(c, "route_id"), ActorName: c.Query("actor"), Action: c.Query("action"), Page: queryInt(c, "page", 1), PageSize: queryInt(c, "page_size", 50)}
	if value := c.Query("from"); value != "" {
		if parsed, err := time.Parse(time.RFC3339, value); err == nil {
			query.From = &parsed
		}
	}
	if value := c.Query("to"); value != "" {
		if parsed, err := time.Parse(time.RFC3339, value); err == nil {
			query.To = &parsed
		}
	}
	items, pagination, err := h.service.List(query)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, http.StatusOK, items, pagination)
}

func (h *AuditHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "fiber-otdr-fault-localization"})
}

func (h *AuditHandler) Ready(c *gin.Context) {
	if err := h.service.Ready(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

package handler

import (
	"net/http"
	"strconv"

	"fiber-otdr-fault-localization/backend/internal/constants"
	"fiber-otdr-fault-localization/backend/internal/dto"
	"fiber-otdr-fault-localization/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type EventHandler struct {
	service  *service.EventService
	validate *validator.Validate
}

func NewEventHandler(events *service.EventService, validate *validator.Validate) *EventHandler {
	return &EventHandler{events, validate}
}

func (h *EventHandler) List(c *gin.Context) {
	query := dto.EventQuery{TraceID: queryUint(c, "trace_id"), RouteID: queryUint(c, "route_id"), Type: constants.EventType(c.Query("event_type")), Page: queryInt(c, "page", 1), PageSize: queryInt(c, "page_size", 50)}
	if raw := c.Query("reviewed"); raw != "" {
		if value, err := strconv.ParseBool(raw); err == nil {
			query.Reviewed = &value
		}
	}
	items, pagination, err := h.service.List(query)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, http.StatusOK, items, pagination)
}

func (h *EventHandler) Detect(c *gin.Context) {
	id, valid := idParam(c)
	if !valid {
		return
	}
	var request dto.DetectEventsRequest
	if !bind(c, h.validate, &request) {
		return
	}
	result, err := h.service.Detect(id, request, actor(c))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, http.StatusOK, result, nil)
}

func (h *EventHandler) Review(c *gin.Context) {
	id, valid := idParam(c)
	if !valid {
		return
	}
	var request dto.ReviewEventRequest
	if !bind(c, h.validate, &request) {
		return
	}
	item, err := h.service.Review(id, request, actor(c))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, http.StatusOK, item, nil)
}

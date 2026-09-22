package handler

import (
	"net/http"
	"time"

	"fiber-otdr-fault-localization/backend/internal/dto"
	"fiber-otdr-fault-localization/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type TraceHandler struct {
	service  *service.TraceService
	validate *validator.Validate
}

func NewTraceHandler(trace *service.TraceService, validate *validator.Validate) *TraceHandler {
	return &TraceHandler{trace, validate}
}

func (h *TraceHandler) List(c *gin.Context) {
	query := dto.TraceQuery{RouteID: queryUint(c, "route_id"), Page: queryInt(c, "page", 1), PageSize: queryInt(c, "page_size", 20)}
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

func (h *TraceHandler) Import(c *gin.Context) {
	var request dto.ImportTraceRequest
	if !bind(c, h.validate, &request) {
		return
	}
	item, err := h.service.Import(request, actor(c))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, http.StatusCreated, item, nil)
}

func (h *TraceHandler) Get(c *gin.Context) {
	id, valid := idParam(c)
	if !valid {
		return
	}
	detail, events, err := h.service.Get(id)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, http.StatusOK, gin.H{"trace": detail, "events": events}, nil)
}

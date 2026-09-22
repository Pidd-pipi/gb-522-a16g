package handler

import (
	"net/http"

	"fiber-otdr-fault-localization/backend/internal/dto"
	"fiber-otdr-fault-localization/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type FiberRouteHandler struct {
	service  *service.RouteService
	validate *validator.Validate
}

func NewFiberRouteHandler(route *service.RouteService, validate *validator.Validate) *FiberRouteHandler {
	return &FiberRouteHandler{route, validate}
}

func (h *FiberRouteHandler) List(c *gin.Context) {
	query := dto.RouteQuery{Keyword: c.Query("keyword"), Status: c.Query("status"), Page: queryInt(c, "page", 1), PageSize: queryInt(c, "page_size", 20)}
	items, pagination, err := h.service.List(query)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, http.StatusOK, items, pagination)
}

func (h *FiberRouteHandler) Create(c *gin.Context) {
	var request dto.CreateRouteRequest
	if !bind(c, h.validate, &request) {
		return
	}
	item, err := h.service.Create(request, actor(c))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, http.StatusCreated, item, nil)
}

func (h *FiberRouteHandler) Get(c *gin.Context) {
	id, valid := idParam(c)
	if !valid {
		return
	}
	route, traces, pendingCases, rejections, err := h.service.Get(id)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, http.StatusOK, gin.H{"route": route, "traces": traces, "pending_cases": pendingCases, "baseline_rejections": rejections}, nil)
}

func (h *FiberRouteHandler) Update(c *gin.Context) {
	id, valid := idParam(c)
	if !valid {
		return
	}
	var request dto.UpdateRouteRequest
	if !bind(c, h.validate, &request) {
		return
	}
	item, err := h.service.Update(id, request, actor(c))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, http.StatusOK, item, nil)
}

func (h *FiberRouteHandler) SetBaseline(c *gin.Context) {
	id, valid := idParam(c)
	if !valid {
		return
	}
	var request dto.SetBaselineRequest
	if !bind(c, h.validate, &request) {
		return
	}
	item, err := h.service.SetBaseline(id, request.TraceID, request.Version, actor(c))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, http.StatusOK, item, nil)
}

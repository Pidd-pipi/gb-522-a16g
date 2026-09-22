package handler

import (
	"net/http"

	"fiber-otdr-fault-localization/backend/internal/dto"
	"fiber-otdr-fault-localization/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type CaseHandler struct {
	service  *service.CaseService
	validate *validator.Validate
}

func NewCaseHandler(cases *service.CaseService, validate *validator.Validate) *CaseHandler {
	return &CaseHandler{cases, validate}
}

func (h *CaseHandler) List(c *gin.Context) {
	query := dto.CaseQuery{RouteID: queryUint(c, "route_id"), Status: c.Query("status"), Page: queryInt(c, "page", 1), PageSize: queryInt(c, "page_size", 20)}
	items, pagination, err := h.service.List(query)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, http.StatusOK, items, pagination)
}

func (h *CaseHandler) Create(c *gin.Context) {
	var request dto.CreateCaseRequest
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

func (h *CaseHandler) Get(c *gin.Context) {
	id, valid := idParam(c)
	if !valid {
		return
	}
	item, differences, err := h.service.Get(id)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, http.StatusOK, gin.H{"case": item, "differences": differences}, nil)
}

func (h *CaseHandler) Analyze(c *gin.Context) {
	id, valid := idParam(c)
	if !valid {
		return
	}
	var request dto.AnalyzeCaseRequest
	if !bind(c, h.validate, &request) {
		return
	}
	item, err := h.service.Analyze(id, request, actor(c))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, http.StatusOK, item, nil)
}

func (h *CaseHandler) Confirm(c *gin.Context) {
	id, valid := idParam(c)
	if !valid {
		return
	}
	var request dto.ConfirmCaseRequest
	if !bind(c, h.validate, &request) {
		return
	}
	item, err := h.service.Confirm(id, request, actor(c))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, http.StatusOK, item, nil)
}

func (h *CaseHandler) Close(c *gin.Context) {
	id, valid := idParam(c)
	if !valid {
		return
	}
	var request dto.CloseCaseRequest
	if !bind(c, h.validate, &request) {
		return
	}
	item, err := h.service.Close(id, request, actor(c))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, http.StatusOK, item, nil)
}

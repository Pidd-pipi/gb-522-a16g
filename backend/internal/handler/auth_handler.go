package handler

import (
	"errors"
	"net/http"
	"strconv"

	"fiber-otdr-fault-localization/backend/internal/dto"
	"fiber-otdr-fault-localization/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type AuthHandler struct {
	service  *service.AuthService
	validate *validator.Validate
}

func NewAuthHandler(auth *service.AuthService, validate *validator.Validate) *AuthHandler {
	return &AuthHandler{auth, validate}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var request dto.LoginRequest
	if !bind(c, h.validate, &request) {
		return
	}
	response, err := h.service.Login(request)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, http.StatusOK, response, nil)
}

func ok(c *gin.Context, status int, data any, meta any) {
	payload := gin.H{"data": data, "request_id": c.GetString("request_id")}
	if meta != nil {
		payload["meta"] = meta
	}
	c.JSON(status, payload)
}

func fail(c *gin.Context, err error) {
	var appErr *service.AppError
	if errors.As(err, &appErr) {
		payload := gin.H{"code": appErr.Code, "message": appErr.Message, "request_id": c.GetString("request_id")}
		if appErr.Details != nil {
			payload["details"] = appErr.Details
		}
		c.JSON(appErr.Status, gin.H{"error": payload})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": service.CodeInternal, "message": "unexpected server error", "request_id": c.GetString("request_id")}})
}

func bind(c *gin.Context, validate *validator.Validate, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		fail(c, &service.AppError{Code: service.CodeInvalidInput, Status: http.StatusBadRequest, Message: "request body is not valid JSON", Err: err})
		return false
	}
	if err := validate.Struct(target); err != nil {
		fail(c, &service.AppError{Code: service.CodeInvalidInput, Status: http.StatusBadRequest, Message: "request validation failed: " + err.Error(), Err: err})
		return false
	}
	return true
}

func idParam(c *gin.Context) (uint, bool) {
	value, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || value == 0 {
		fail(c, &service.AppError{Code: service.CodeInvalidInput, Status: http.StatusBadRequest, Message: "resource id must be a positive integer", Err: err})
		return 0, false
	}
	return uint(value), true
}

func actor(c *gin.Context) service.Actor {
	id, _ := c.Get("user_id")
	role, _ := c.Get("role")
	username, _ := c.Get("username")
	userID, _ := id.(uint)
	roleString, _ := role.(string)
	usernameString, _ := username.(string)
	return service.Actor{ID: userID, Username: usernameString, Role: roleString, RequestID: c.GetString("request_id")}
}

func queryUint(c *gin.Context, key string) *uint {
	if c.Query(key) == "" {
		return nil
	}
	value, err := strconv.ParseUint(c.Query(key), 10, 64)
	if err != nil || value == 0 {
		return nil
	}
	converted := uint(value)
	return &converted
}

func queryInt(c *gin.Context, key string, fallback int) int {
	value, err := strconv.Atoi(c.Query(key))
	if err != nil {
		return fallback
	}
	return value
}

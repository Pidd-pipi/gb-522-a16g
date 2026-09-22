package dto

import "time"

type LoginRequest struct {
	Username string `json:"username" validate:"required,min=3,max=60"`
	Password string `json:"password" validate:"required,min=8,max=100"`
}

type UserView struct {
	ID          uint   `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      UserView  `json:"user"`
}

type AuditQuery struct {
	RouteID   *uint
	ActorName string
	Action    string
	From      *time.Time
	To        *time.Time
	Page      int
	PageSize  int
}

type Pagination struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

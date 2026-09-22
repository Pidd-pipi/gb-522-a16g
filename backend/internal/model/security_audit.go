package model

import "time"

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"size:60;not null;uniqueIndex" json:"username"`
	PasswordHash string    `gorm:"size:100;not null" json:"-"`
	DisplayName  string    `gorm:"size:100;not null" json:"display_name"`
	Role         string    `gorm:"size:24;not null;index;check:user_role_allowed,role IN ('analyst','reviewer','admin')" json:"role"`
	Active       bool      `gorm:"not null;default:true" json:"active"`
	CreatedAt    time.Time `json:"created_at"`
}

type AuditLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ActorID      uint      `gorm:"not null;index" json:"actor_id"`
	ActorName    string    `gorm:"size:60;not null;index" json:"actor_name"`
	Action       string    `gorm:"size:80;not null;index" json:"action"`
	ResourceType string    `gorm:"size:60;not null;index" json:"resource_type"`
	ResourceID   uint      `gorm:"not null;index" json:"resource_id"`
	RouteID      *uint     `gorm:"index" json:"route_id"`
	RequestID    string    `gorm:"size:80;not null;index" json:"request_id"`
	Before       string    `gorm:"type:text;not null" json:"before"`
	After        string    `gorm:"type:text;not null" json:"after"`
	CreatedAt    time.Time `gorm:"index" json:"created_at"`
}

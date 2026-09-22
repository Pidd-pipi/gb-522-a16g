package repository

import (
	"fmt"
	"strings"

	"fiber-otdr-fault-localization/backend/internal/dto"
	"fiber-otdr-fault-localization/backend/internal/model"
	"gorm.io/gorm"
)

type AuditRepository struct{ db *gorm.DB }

func (r *AuditRepository) Create(entry *model.AuditLog) error {
	if strings.TrimSpace(entry.RequestID) == "" {
		return fmt.Errorf("audit request id is required")
	}
	if err := r.db.Create(entry).Error; err != nil {
		return fmt.Errorf("create immutable audit entry: %w", err)
	}
	return nil
}

// ListRejectedBaselineChanges returns the recorded baseline-switch refusals for
// a route, newest first, so the route detail view can read rejection reasons
// back after a refresh.
func (r *AuditRepository) ListRejectedBaselineChanges(routeID uint, limit int) ([]model.AuditLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var entries []model.AuditLog
	if err := r.db.Where("action = ? AND route_id = ?", "route.baseline_change_rejected", routeID).Order("created_at DESC").Limit(limit).Find(&entries).Error; err != nil {
		return nil, fmt.Errorf("list rejected baseline changes: %w", err)
	}
	return entries, nil
}

func (r *AuditRepository) List(query dto.AuditQuery) ([]model.AuditLog, int64, error) {
	db := r.db.Model(&model.AuditLog{})
	if query.RouteID != nil {
		db = db.Where("route_id = ?", *query.RouteID)
	}
	if query.ActorName != "" {
		db = db.Where("actor_name LIKE ?", "%"+query.ActorName+"%")
	}
	if query.Action != "" {
		db = db.Where("action = ?", query.Action)
	}
	if query.From != nil {
		db = db.Where("created_at >= ?", *query.From)
	}
	if query.To != nil {
		db = db.Where("created_at <= ?", *query.To)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}
	var entries []model.AuditLog
	if err := db.Order("created_at DESC").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&entries).Error; err != nil {
		return nil, 0, fmt.Errorf("list audit logs: %w", err)
	}
	return entries, total, nil
}

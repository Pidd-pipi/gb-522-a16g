package repository

import (
	"errors"
	"fmt"
	"strings"

	"fiber-otdr-fault-localization/backend/internal/dto"
	"fiber-otdr-fault-localization/backend/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FiberRouteRepository struct{ db *gorm.DB }

func (r *FiberRouteRepository) Create(route *model.FiberRoute) error {
	if err := r.db.Create(route).Error; err != nil {
		return fmt.Errorf("create fiber route: %w", err)
	}
	return nil
}

func (r *FiberRouteRepository) Get(id uint) (model.FiberRoute, error) {
	var route model.FiberRoute
	if err := r.db.First(&route, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return route, ErrNotFound
		}
		return route, fmt.Errorf("get fiber route: %w", err)
	}
	return route, nil
}

func (r *FiberRouteRepository) GetByCode(code string) (model.FiberRoute, error) {
	var route model.FiberRoute
	if err := r.db.Where("route_code = ?", code).First(&route).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return route, ErrNotFound
		}
		return route, fmt.Errorf("get fiber route by code: %w", err)
	}
	return route, nil
}

func (r *FiberRouteRepository) List(query dto.RouteQuery) ([]model.FiberRoute, int64, error) {
	db := r.db.Model(&model.FiberRoute{})
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("route_code LIKE ? OR name LIKE ?", like, like)
	}
	if query.Status != "" {
		db = db.Where("route_status = ?", query.Status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count fiber routes: %w", err)
	}
	var routes []model.FiberRoute
	if err := db.Order("created_at DESC").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&routes).Error; err != nil {
		return nil, 0, fmt.Errorf("list fiber routes: %w", err)
	}
	return routes, total, nil
}

func (r *FiberRouteRepository) Update(route *model.FiberRoute) error {
	result := r.db.Model(&model.FiberRoute{}).Where("id = ?", route.ID).Updates(map[string]any{"name": route.Name, "length_m": route.LengthM, "refractive_index": route.RefractiveIndex, "launch_connector": route.LaunchConnector, "route_status": route.RouteStatus})
	if result.Error != nil {
		return fmt.Errorf("update fiber route: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// GetForUpdate loads a route and takes a row lock on databases that support
// SELECT ... FOR UPDATE, serializing baseline replacement against case creation
// that snapshots the route baseline. The SQLite self-contained builds rely on
// the database write lock plus the versioned conditional update instead.
func (r *FiberRouteRepository) GetForUpdate(id uint) (model.FiberRoute, error) {
	var route model.FiberRoute
	query := r.db
	if r.db.Dialector.Name() == "postgres" {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := query.First(&route, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return route, ErrNotFound
		}
		return route, fmt.Errorf("get fiber route for update: %w", err)
	}
	return route, nil
}

// ReplaceBaseline swaps the baseline only when the row still carries the
// expected version and baseline. RowsAffected == 0 means another transaction
// changed the baseline first (concurrent change / repeated submission).
func (r *FiberRouteRepository) ReplaceBaseline(routeID uint, expectedVersion, traceID uint) error {
	result := r.db.Model(&model.FiberRoute{}).
		Where("id = ? AND version = ?", routeID, expectedVersion).
		Updates(map[string]any{"baseline_trace_id": traceID, "version": gorm.Expr("version + 1")})
	if result.Error != nil {
		return fmt.Errorf("replace route baseline: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrConcurrentChange
	}
	return nil
}

package repository

import (
	"errors"
	"fmt"

	"fiber-otdr-fault-localization/backend/internal/dto"
	"fiber-otdr-fault-localization/backend/internal/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type TraceRepository struct{ db *gorm.DB }

func (r *TraceRepository) Create(trace *model.TraceCapture) error {
	if err := r.db.Create(trace).Error; err != nil {
		return fmt.Errorf("create trace capture: %w", err)
	}
	return nil
}

func (r *TraceRepository) Get(id uint) (model.TraceCapture, error) {
	var trace model.TraceCapture
	if err := r.db.First(&trace, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return trace, ErrNotFound
		}
		return trace, fmt.Errorf("get trace capture: %w", err)
	}
	return trace, nil
}

func (r *TraceRepository) List(query dto.TraceQuery) ([]model.TraceCapture, int64, error) {
	db := r.db.Model(&model.TraceCapture{})
	if query.RouteID != nil {
		db = db.Where("route_id = ?", *query.RouteID)
	}
	if query.From != nil {
		db = db.Where("captured_at >= ?", *query.From)
	}
	if query.To != nil {
		db = db.Where("captured_at <= ?", *query.To)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count traces: %w", err)
	}
	var traces []model.TraceCapture
	if err := db.Order("captured_at DESC").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&traces).Error; err != nil {
		return nil, 0, fmt.Errorf("list traces: %w", err)
	}
	return traces, total, nil
}

func (r *TraceRepository) UpdateProcessing(id uint, processed datatypes.JSON, noise float64, window int, threshold float64, merge int) error {
	result := r.db.Model(&model.TraceCapture{}).Where("id = ?", id).Updates(map[string]any{"processed_json": processed, "noise_floor_db": noise, "denoise_window": window, "peak_threshold_db": threshold, "merge_window": merge})
	if result.Error != nil {
		return fmt.Errorf("update trace processing: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *TraceRepository) BelongsToRoute(traceID, routeID uint) (bool, error) {
	var count int64
	if err := r.db.Model(&model.TraceCapture{}).Where("id = ? AND route_id = ?", traceID, routeID).Count(&count).Error; err != nil {
		return false, fmt.Errorf("check trace route: %w", err)
	}
	return count == 1, nil
}

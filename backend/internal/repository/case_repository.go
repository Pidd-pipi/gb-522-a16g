package repository

import (
	"errors"
	"fmt"
	"time"

	"fiber-otdr-fault-localization/backend/internal/constants"
	"fiber-otdr-fault-localization/backend/internal/dto"
	"fiber-otdr-fault-localization/backend/internal/model"
	"gorm.io/gorm"
)

type CaseRepository struct{ db *gorm.DB }

func (r *CaseRepository) Create(item *model.LocalizationCase) error {
	if err := r.db.Create(item).Error; err != nil {
		return fmt.Errorf("create localization case: %w", err)
	}
	return nil
}

func (r *CaseRepository) Get(id uint) (model.LocalizationCase, error) {
	var item model.LocalizationCase
	if err := r.db.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return item, ErrNotFound
		}
		return item, fmt.Errorf("get localization case: %w", err)
	}
	return item, nil
}

func (r *CaseRepository) List(query dto.CaseQuery) ([]model.LocalizationCase, int64, error) {
	db := r.db.Model(&model.LocalizationCase{})
	if query.RouteID != nil {
		db = db.Where("route_id = ?", *query.RouteID)
	}
	if query.Status != "" {
		db = db.Where("case_status = ?", query.Status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count localization cases: %w", err)
	}
	var items []model.LocalizationCase
	if err := db.Order("created_at DESC").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list localization cases: %w", err)
	}
	return items, total, nil
}

func (r *CaseRepository) Transition(id, version uint, from, to constants.CaseStatus, updates map[string]any) error {
	updates["case_status"] = to
	updates["version"] = gorm.Expr("version + 1")
	result := r.db.Model(&model.LocalizationCase{}).Where("id = ? AND version = ? AND case_status = ?", id, version, from).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("transition localization case: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("case state or version changed: %w", gorm.ErrInvalidTransaction)
	}
	return nil
}

func (r *CaseRepository) SaveAnalysis(id uint, status constants.CaseStatus, updates map[string]any) error {
	updates["case_status"] = status
	updates["version"] = gorm.Expr("version + 1")
	result := r.db.Model(&model.LocalizationCase{}).Where("id = ? AND case_status = ?", id, constants.CaseAnalyzing).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("save case analysis: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("case is no longer analyzing: %w", gorm.ErrInvalidTransaction)
	}
	return nil
}

// RecoverStaleAnalysis releases a worker lease left behind by a crashed process.
// The timestamp predicate keeps concurrent analyzers from resetting a fresh run.
func (r *CaseRepository) RecoverStaleAnalysis(id uint, cutoff time.Time) (bool, error) {
	result := r.db.Model(&model.LocalizationCase{}).
		Where("id = ? AND case_status = ? AND updated_at < ?", id, constants.CaseAnalyzing, cutoff).
		Updates(map[string]any{"case_status": constants.CaseDraft, "analysis_error": "previous analysis worker expired; please retry", "version": gorm.Expr("version + 1")})
	if result.Error != nil {
		return false, fmt.Errorf("recover stale case analysis: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

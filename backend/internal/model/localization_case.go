package model

import (
	"time"

	"fiber-otdr-fault-localization/backend/internal/constants"
	"gorm.io/datatypes"
)

type LocalizationCase struct {
	ID                 uint                 `gorm:"primaryKey" json:"id"`
	RouteID            uint                 `gorm:"not null;index" json:"route_id"`
	BaselineTraceID    uint                 `gorm:"not null;index" json:"baseline_trace_id"`
	CurrentTraceID     uint                 `gorm:"not null;index" json:"current_trace_id"`
	CaseStatus         constants.CaseStatus `gorm:"size:24;not null;index;check:case_status_allowed,case_status IN ('draft','analyzing','pending_review','confirmed','closed')" json:"case_status"`
	EstimatedDistanceM *float64             `json:"estimated_distance_m"`
	UncertaintyM       *float64             `json:"uncertainty_m"`
	Conclusion         string               `gorm:"size:2000" json:"conclusion"`
	ReviewerID         *uint                `json:"reviewer_id"`
	ClosedAt           *time.Time           `json:"closed_at"`
	AnalysisError      string               `gorm:"size:1000" json:"analysis_error"`
	ParametersJSON     datatypes.JSON       `gorm:"type:jsonb;not null" json:"parameters_json"`
	DifferencesJSON    datatypes.JSON       `gorm:"type:jsonb" json:"differences_json"`
	Version            uint                 `gorm:"not null;default:1" json:"version"`
	CreatedBy          uint                 `gorm:"not null" json:"created_by"`
	CreatedAt          time.Time            `json:"created_at"`
	UpdatedAt          time.Time            `json:"updated_at"`
}

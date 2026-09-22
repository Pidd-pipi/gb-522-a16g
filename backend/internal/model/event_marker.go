package model

import (
	"time"

	"fiber-otdr-fault-localization/backend/internal/constants"
)

type EventMarker struct {
	ID                       uint                `gorm:"primaryKey" json:"id"`
	TraceID                  uint                `gorm:"not null;index" json:"trace_id"`
	DistanceM                float64             `gorm:"not null" json:"distance_m"`
	EventType                constants.EventType `gorm:"size:24;not null;index;check:event_type_allowed,event_type IN ('connector','splice','bend','break','end','unknown')" json:"event_type"`
	InsertionLossDB          float64             `gorm:"not null" json:"insertion_loss_db"`
	ReflectanceDB            float64             `gorm:"not null" json:"reflectance_db"`
	Confidence               float64             `gorm:"not null" json:"confidence"`
	AlgorithmEventType       constants.EventType `gorm:"size:24;not null;check:algorithm_event_type_allowed,algorithm_event_type IN ('connector','splice','bend','break','end','unknown')" json:"algorithm_event_type"`
	AlgorithmDistanceM       float64             `gorm:"not null" json:"algorithm_distance_m"`
	AlgorithmInsertionLossDB float64             `gorm:"not null" json:"algorithm_insertion_loss_db"`
	Reviewed                 bool                `gorm:"not null;default:false;index" json:"reviewed"`
	ReviewNote               string              `gorm:"size:1000" json:"review_note"`
	ReviewedBy               *uint               `json:"reviewed_by"`
	ReviewedAt               *time.Time          `json:"reviewed_at"`
	CreatedAt                time.Time           `json:"created_at"`
	UpdatedAt                time.Time           `json:"updated_at"`
}

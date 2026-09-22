package dto

import "fiber-otdr-fault-localization/backend/internal/constants"

type ReviewEventRequest struct {
	EventType  constants.EventType `json:"event_type" validate:"required"`
	DistanceM  *float64            `json:"distance_m" validate:"omitempty,gte=0"`
	ReviewNote string              `json:"review_note" validate:"required,min=3,max=1000"`
}

type EventQuery struct {
	TraceID  *uint
	RouteID  *uint
	Type     constants.EventType
	Reviewed *bool
	Page     int
	PageSize int
}

type DetectionSummary struct {
	TraceID       uint    `json:"trace_id"`
	DetectedCount int     `json:"detected_count"`
	NoiseFloorDB  float64 `json:"noise_floor_db"`
	ThresholdDB   float64 `json:"threshold_db"`
	RejectedCount int     `json:"rejected_out_of_bounds"`
}

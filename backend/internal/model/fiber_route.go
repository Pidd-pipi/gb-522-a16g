package model

import "time"

type FiberRoute struct {
	ID              uint    `gorm:"primaryKey" json:"id"`
	RouteCode       string  `gorm:"size:40;not null;uniqueIndex" json:"route_code"`
	Name            string  `gorm:"size:120;not null" json:"name"`
	LengthM         float64 `gorm:"not null;check:length_m > 0 AND length_m <= 500000" json:"length_m"`
	RefractiveIndex float64 `gorm:"not null;check:refractive_index >= 1.3 AND refractive_index <= 1.7" json:"refractive_index"`
	LaunchConnector string  `gorm:"size:80;not null" json:"launch_connector"`
	RouteStatus     string  `gorm:"size:24;not null;default:active;check:route_status_allowed,route_status IN ('active','maintenance','retired')" json:"route_status"`
	BaselineTraceID *uint   `gorm:"index" json:"baseline_trace_id"`
	// Version is the optimistic-lock token bumped on every baseline replacement,
	// so callers that held an older route snapshot cannot overwrite a baseline
	// that changed concurrently.
	Version   uint      `gorm:"not null;default:1" json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

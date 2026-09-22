package model

import "time"

type FiberRoute struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	RouteCode       string    `gorm:"size:40;not null;uniqueIndex" json:"route_code"`
	Name            string    `gorm:"size:120;not null" json:"name"`
	LengthM         float64   `gorm:"not null;check:length_m > 0 AND length_m <= 500000" json:"length_m"`
	RefractiveIndex float64   `gorm:"not null;check:refractive_index >= 1.3 AND refractive_index <= 1.7" json:"refractive_index"`
	LaunchConnector string    `gorm:"size:80;not null" json:"launch_connector"`
	RouteStatus     string    `gorm:"size:24;not null;default:active;check:route_status_allowed,route_status IN ('active','maintenance','retired')" json:"route_status"`
	BaselineTraceID *uint     `gorm:"index" json:"baseline_trace_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// BaselineChangeAttempt persists the outcome of every reviewer request to
// replace a route's review baseline. Rejected attempts are committed inside
// the same transaction that performs the checks so the route detail page can
// read back the refusal reason after a refresh, even though the baseline
// itself is never rewritten.
type BaselineChangeAttempt struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	RouteID          uint      `gorm:"not null;index" json:"route_id"`
	RequestedTraceID uint      `gorm:"not null" json:"requested_trace_id"`
	CurrentTraceID   *uint     `json:"current_trace_id"`
	Result           string    `gorm:"size:24;not null;index;check:baseline_attempt_result_allowed,result IN ('rejected','succeeded')" json:"result"`
	Reason           string    `gorm:"size:40;not null;index;check:baseline_attempt_reason_allowed,reason IN ('open_cases','trace_not_on_route','same_baseline','baseline_concurrent_change','success')" json:"reason"`
	ActorID          uint      `gorm:"not null" json:"actor_id"`
	ActorName        string    `gorm:"size:60;not null" json:"actor_name"`
	RequestID        string    `gorm:"size:80;not null;index" json:"request_id"`
	BlockingCaseIDs  string    `gorm:"size:500;not null;default:'[]'" json:"blocking_case_ids"`
	CreatedAt        time.Time `gorm:"index" json:"created_at"`
}

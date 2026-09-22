package dto

type CreateRouteRequest struct {
	RouteCode       string  `json:"route_code" validate:"required,min=2,max=40,alphanumunicode"`
	Name            string  `json:"name" validate:"required,min=2,max=120"`
	LengthM         float64 `json:"length_m" validate:"required,gt=0,lte=500000"`
	RefractiveIndex float64 `json:"refractive_index" validate:"required,gte=1.3,lte=1.7"`
	LaunchConnector string  `json:"launch_connector" validate:"required,min=2,max=80"`
	RouteStatus     string  `json:"route_status" validate:"omitempty,oneof=active maintenance retired"`
}

type UpdateRouteRequest struct {
	Name            *string  `json:"name" validate:"omitempty,min=2,max=120"`
	LengthM         *float64 `json:"length_m" validate:"omitempty,gt=0,lte=500000"`
	RefractiveIndex *float64 `json:"refractive_index" validate:"omitempty,gte=1.3,lte=1.7"`
	LaunchConnector *string  `json:"launch_connector" validate:"omitempty,min=2,max=80"`
	RouteStatus     *string  `json:"route_status" validate:"omitempty,oneof=active maintenance retired"`
}

type SetBaselineRequest struct {
	TraceID uint `json:"trace_id" validate:"required,gt=0"`
	// Version is the route version the reviewer last read. It is required so a
	// baseline that changed concurrently cannot be silently overwritten.
	Version uint `json:"version" validate:"required,gt=0"`
}

// BaselineBlocker describes one still-open case that pins the current baseline.
type BaselineBlocker struct {
	CaseID          uint   `json:"case_id"`
	CaseStatus      string `json:"case_status"`
	BaselineTraceID uint   `json:"baseline_trace_id"`
	CurrentTraceID  uint   `json:"current_trace_id"`
	CreatedAt       string `json:"created_at"`
}

// BaselineRejection is a persisted refusal of a baseline switch, read back in
// the route detail view after refresh.
type BaselineRejection struct {
	RequestID      string            `json:"request_id"`
	ActorName      string            `json:"actor_name"`
	Reason         string            `json:"reason"`
	Message        string            `json:"message"`
	RequestedTrace uint              `json:"requested_trace_id,omitempty"`
	Blockers       []BaselineBlocker `json:"blockers,omitempty"`
	CreatedAt      string            `json:"created_at"`
}

type RouteQuery struct {
	Keyword  string
	Status   string
	Page     int
	PageSize int
}

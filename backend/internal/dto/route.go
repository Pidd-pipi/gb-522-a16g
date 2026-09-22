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
}

// BlockingCaseView summarizes an open case that blocks a baseline replacement.
type BlockingCaseView struct {
	ID             uint   `json:"id"`
	CaseStatus     string `json:"case_status"`
	CurrentTraceID uint   `json:"current_trace_id"`
	CreatedAt      string `json:"created_at"`
}

// PendingCaseView summarizes an open case on a route for the detail panel.
type PendingCaseView struct {
	ID              uint   `json:"id"`
	BaselineTraceID uint   `json:"baseline_trace_id"`
	CurrentTraceID  uint   `json:"current_trace_id"`
	CaseStatus      string `json:"case_status"`
	Conclusion      string `json:"conclusion"`
	CreatedAt       string `json:"created_at"`
}

// BaselineRejectionView is the persisted outcome of a baseline change attempt.
type BaselineRejectionView struct {
	ID               uint   `json:"id"`
	RequestedTraceID uint   `json:"requested_trace_id"`
	CurrentTraceID   *uint  `json:"current_trace_id"`
	Result           string `json:"result"`
	Reason           string `json:"reason"`
	ActorName        string `json:"actor_name"`
	BlockingCaseIDs  []uint `json:"blocking_case_ids"`
	CreatedAt        string `json:"created_at"`
}

type RouteQuery struct {
	Keyword  string
	Status   string
	Page     int
	PageSize int
}

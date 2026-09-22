package dto

type CreateCaseRequest struct {
	RouteID            uint    `json:"route_id" validate:"required,gt=0"`
	BaselineTraceID    uint    `json:"baseline_trace_id" validate:"required,gt=0"`
	CurrentTraceID     uint    `json:"current_trace_id" validate:"required,gt=0,nefield=BaselineTraceID"`
	DistanceToleranceM float64 `json:"distance_tolerance_m" validate:"omitempty,gt=0,lte=1000"`
	LossIncreaseDB     float64 `json:"loss_increase_db" validate:"omitempty,gt=0,lte=20"`
}

type AnalyzeCaseRequest struct {
	DistanceToleranceM float64 `json:"distance_tolerance_m" validate:"omitempty,gt=0,lte=1000"`
	LossIncreaseDB     float64 `json:"loss_increase_db" validate:"omitempty,gt=0,lte=20"`
}

type ConfirmCaseRequest struct {
	Conclusion         string  `json:"conclusion" validate:"required,min=10,max=2000"`
	EstimatedDistanceM float64 `json:"estimated_distance_m" validate:"gte=0"`
	UncertaintyM       float64 `json:"uncertainty_m" validate:"gte=0,lte=5000"`
	Version            uint    `json:"version" validate:"required,gt=0"`
}

type CloseCaseRequest struct {
	Version uint `json:"version" validate:"required,gt=0"`
}

type CaseQuery struct {
	RouteID  *uint
	Status   string
	Page     int
	PageSize int
}

type CaseParameters struct {
	DistanceToleranceM float64 `json:"distance_tolerance_m"`
	LossIncreaseDB     float64 `json:"loss_increase_db"`
}

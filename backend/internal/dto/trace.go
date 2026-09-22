package dto

import "time"

type ImportTraceRequest struct {
	RouteID          uint      `json:"route_id" validate:"required,gt=0"`
	WavelengthNM     int       `json:"wavelength_nm" validate:"required,oneof=1310 1490 1550 1625"`
	PulseWidthNS     float64   `json:"pulse_width_ns" validate:"required,gt=0,lte=20000"`
	SampleIntervalNS float64   `json:"sample_interval_ns" validate:"required,gt=0,lte=10000"`
	Points           []float64 `json:"points" validate:"required,min=16"`
	CapturedAt       time.Time `json:"captured_at" validate:"required"`
	DenoiseWindow    int       `json:"denoise_window" validate:"omitempty,min=1,max=31"`
	PeakThresholdDB  float64   `json:"peak_threshold_db" validate:"omitempty,gt=0,lte=20"`
	MergeWindow      int       `json:"merge_window" validate:"omitempty,min=1,max=50"`
}

type DetectEventsRequest struct {
	DenoiseWindow   int     `json:"denoise_window" validate:"omitempty,min=1,max=31"`
	PeakThresholdDB float64 `json:"peak_threshold_db" validate:"omitempty,gt=0,lte=20"`
	MergeWindow     int     `json:"merge_window" validate:"omitempty,min=1,max=50"`
}

type TraceQuery struct {
	RouteID  *uint
	From     *time.Time
	To       *time.Time
	Page     int
	PageSize int
}

type TraceDetail struct {
	ID               uint      `json:"id"`
	RouteID          uint      `json:"route_id"`
	WavelengthNM     int       `json:"wavelength_nm"`
	PulseWidthNS     float64   `json:"pulse_width_ns"`
	SampleIntervalNS float64   `json:"sample_interval_ns"`
	Points           []float64 `json:"points"`
	ProcessedPoints  []float64 `json:"processed_points"`
	NoiseFloorDB     float64   `json:"noise_floor_db"`
	DenoiseWindow    int       `json:"denoise_window"`
	PeakThresholdDB  float64   `json:"peak_threshold_db"`
	MergeWindow      int       `json:"merge_window"`
	CapturedAt       time.Time `json:"captured_at"`
	UploadedBy       uint      `json:"uploaded_by"`
}

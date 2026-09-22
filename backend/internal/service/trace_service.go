package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"fiber-otdr-fault-localization/backend/internal/algorithm"
	"fiber-otdr-fault-localization/backend/internal/dto"
	"fiber-otdr-fault-localization/backend/internal/model"
	"fiber-otdr-fault-localization/backend/internal/repository"
	"gorm.io/datatypes"
)

type TraceService struct {
	store     *repository.Store
	maxPoints int
}

func NewTraceService(store *repository.Store, maxPoints int) *TraceService {
	return &TraceService{store, maxPoints}
}

func (s *TraceService) Import(request dto.ImportTraceRequest, actor Actor) (model.TraceCapture, error) {
	if len(request.Points) > s.maxPoints {
		return model.TraceCapture{}, invalid(fmt.Sprintf("trace exceeds the %d-point limit", s.maxPoints), nil)
	}
	for i, point := range request.Points {
		if math.IsNaN(point) || math.IsInf(point, 0) || point < -200 || point > 100 {
			return model.TraceCapture{}, invalid(fmt.Sprintf("sample %d is outside the supported dB range", i), nil)
		}
	}
	route, err := s.store.Routes.Get(request.RouteID)
	if errors.Is(err, repository.ErrNotFound) {
		return model.TraceCapture{}, notFound("route")
	}
	if err != nil {
		return model.TraceCapture{}, internal("load route failed", err)
	}
	window := request.DenoiseWindow
	if window == 0 {
		window = 5
	}
	threshold := request.PeakThresholdDB
	if threshold == 0 {
		threshold = 0.8
	}
	merge := request.MergeWindow
	if merge == 0 {
		merge = 3
	}
	filtered, err := algorithm.MovingMedian(request.Points, window)
	if err != nil {
		return model.TraceCapture{}, invalid("trace denoising failed", err)
	}
	noise, err := algorithm.EstimateNoiseFloor(filtered)
	if err != nil {
		return model.TraceCapture{}, &AppError{Code: CodeAlgorithmInput, Status: 422, Message: "not enough trace samples for noise estimation", Err: err}
	}
	lastDistance, err := algorithm.SampleDistance(len(request.Points)-1, request.SampleIntervalNS, route.RefractiveIndex)
	if err != nil {
		return model.TraceCapture{}, invalid("distance conversion failed", err)
	}
	if lastDistance < route.LengthM*0.05 {
		return model.TraceCapture{}, invalid("trace sampling range covers less than five percent of the route", nil)
	}
	raw, _ := json.Marshal(request.Points)
	processed, _ := json.Marshal(filtered)
	trace := model.TraceCapture{RouteID: request.RouteID, WavelengthNM: request.WavelengthNM, PulseWidthNS: request.PulseWidthNS, SampleIntervalNS: request.SampleIntervalNS, RawPointsJSON: datatypes.JSON(raw), ProcessedJSON: datatypes.JSON(processed), NoiseFloorDB: noise, CapturedAt: request.CapturedAt, UploadedBy: actor.ID, DenoiseWindow: window, PeakThresholdDB: threshold, MergeWindow: merge}
	err = s.store.Transaction(func(tx *repository.Store) error {
		if err := tx.Traces.Create(&trace); err != nil {
			return err
		}
		params := map[string]any{"point_count": len(request.Points), "wavelength_nm": request.WavelengthNM, "denoise_window": window, "peak_threshold_db": threshold, "merge_window": merge}
		return tx.Audits.Create(audit(actor, "trace.imported", "TraceCapture", trace.ID, &route.ID, "{}", snapshot(params)))
	})
	if err != nil {
		return trace, internal("import trace failed", err)
	}
	return trace, nil
}

func (s *TraceService) List(query dto.TraceQuery) ([]model.TraceCapture, dto.Pagination, error) {
	normalizePage(&query.Page, &query.PageSize)
	items, total, err := s.store.Traces.List(query)
	if err != nil {
		return nil, dto.Pagination{}, internal("list traces failed", err)
	}
	return items, dto.Pagination{Page: query.Page, PageSize: query.PageSize, Total: total}, nil
}

func (s *TraceService) Get(id uint) (dto.TraceDetail, []model.EventMarker, error) {
	trace, err := s.store.Traces.Get(id)
	if errors.Is(err, repository.ErrNotFound) {
		return dto.TraceDetail{}, nil, notFound("trace")
	}
	if err != nil {
		return dto.TraceDetail{}, nil, internal("get trace failed", err)
	}
	var raw, processed []float64
	if err := json.Unmarshal(trace.RawPointsJSON, &raw); err != nil {
		return dto.TraceDetail{}, nil, internal("decode raw trace points failed", err)
	}
	if len(trace.ProcessedJSON) > 0 {
		if err := json.Unmarshal(trace.ProcessedJSON, &processed); err != nil {
			return dto.TraceDetail{}, nil, internal("decode processed trace points failed", err)
		}
	}
	events, err := s.store.Events.ForTrace(id)
	if err != nil {
		return dto.TraceDetail{}, nil, internal("list trace events failed", err)
	}
	detail := dto.TraceDetail{ID: trace.ID, RouteID: trace.RouteID, WavelengthNM: trace.WavelengthNM, PulseWidthNS: trace.PulseWidthNS, SampleIntervalNS: trace.SampleIntervalNS, Points: raw, ProcessedPoints: processed, NoiseFloorDB: trace.NoiseFloorDB, DenoiseWindow: trace.DenoiseWindow, PeakThresholdDB: trace.PeakThresholdDB, MergeWindow: trace.MergeWindow, CapturedAt: trace.CapturedAt, UploadedBy: trace.UploadedBy}
	return detail, events, nil
}

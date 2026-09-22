package service

import (
	"encoding/json"
	"errors"
	"time"

	"fiber-otdr-fault-localization/backend/internal/algorithm"
	"fiber-otdr-fault-localization/backend/internal/dto"
	"fiber-otdr-fault-localization/backend/internal/model"
	"fiber-otdr-fault-localization/backend/internal/repository"
	"gorm.io/datatypes"
)

type EventService struct{ store *repository.Store }

func NewEventService(store *repository.Store) *EventService { return &EventService{store} }

func (s *EventService) Detect(traceID uint, request dto.DetectEventsRequest, actor Actor) (dto.DetectionSummary, error) {
	trace, err := s.store.Traces.Get(traceID)
	if errors.Is(err, repository.ErrNotFound) {
		return dto.DetectionSummary{}, notFound("trace")
	}
	if err != nil {
		return dto.DetectionSummary{}, internal("get trace failed", err)
	}
	route, err := s.store.Routes.Get(trace.RouteID)
	if err != nil {
		return dto.DetectionSummary{}, internal("get trace route failed", err)
	}
	var raw []float64
	if err := json.Unmarshal(trace.RawPointsJSON, &raw); err != nil {
		return dto.DetectionSummary{}, internal("decode raw trace failed", err)
	}
	window := request.DenoiseWindow
	if window == 0 {
		window = trace.DenoiseWindow
	}
	if window == 0 {
		window = 5
	}
	threshold := request.PeakThresholdDB
	if threshold == 0 {
		threshold = trace.PeakThresholdDB
	}
	if threshold == 0 {
		threshold = 0.8
	}
	merge := request.MergeWindow
	if merge == 0 {
		merge = trace.MergeWindow
	}
	if merge == 0 {
		merge = 3
	}
	filtered, err := algorithm.MovingMedian(raw, window)
	if err != nil {
		return dto.DetectionSummary{}, &AppError{CodeAlgorithmInput, 422, "trace denoising failed", err}
	}
	noise, err := algorithm.EstimateNoiseFloor(filtered)
	if err != nil {
		return dto.DetectionSummary{}, &AppError{CodeAlgorithmInput, 422, "noise floor estimation failed", err}
	}
	detected, rejected, err := algorithm.Detect(filtered, threshold, merge, trace.SampleIntervalNS, route.RefractiveIndex, route.LengthM)
	if err != nil {
		return dto.DetectionSummary{}, &AppError{CodeAlgorithmInput, 422, "event detection failed", err}
	}
	events := make([]model.EventMarker, 0, len(detected))
	for _, item := range detected {
		events = append(events, model.EventMarker{TraceID: trace.ID, DistanceM: item.DistanceM, EventType: item.Type, InsertionLossDB: item.InsertionLossDB, ReflectanceDB: item.ReflectanceDB, Confidence: item.Confidence, AlgorithmEventType: item.Type, AlgorithmDistanceM: item.DistanceM, AlgorithmInsertionLossDB: item.InsertionLossDB})
	}
	processed, _ := json.Marshal(filtered)
	err = s.store.Transaction(func(tx *repository.Store) error {
		if err := tx.Traces.UpdateProcessing(trace.ID, datatypes.JSON(processed), noise, window, threshold, merge); err != nil {
			return err
		}
		if err := tx.Events.ReplaceForTrace(trace.ID, events); err != nil {
			return err
		}
		params := map[string]any{"denoise_window": window, "peak_threshold_db": threshold, "merge_window": merge, "noise_floor_db": noise, "detected": len(events), "rejected_out_of_bounds": rejected}
		return tx.Audits.Create(audit(actor, "trace.events_detected", "TraceCapture", trace.ID, &route.ID, "{}", snapshot(params)))
	})
	if err != nil {
		return dto.DetectionSummary{}, internal("save detected events failed", err)
	}
	return dto.DetectionSummary{TraceID: trace.ID, DetectedCount: len(events), NoiseFloorDB: noise, ThresholdDB: threshold, RejectedCount: rejected}, nil
}

func (s *EventService) List(query dto.EventQuery) ([]model.EventMarker, dto.Pagination, error) {
	normalizePage(&query.Page, &query.PageSize)
	items, total, err := s.store.Events.List(query)
	if err != nil {
		return nil, dto.Pagination{}, internal("list events failed", err)
	}
	return items, dto.Pagination{Page: query.Page, PageSize: query.PageSize, Total: total}, nil
}

func (s *EventService) Review(id uint, request dto.ReviewEventRequest, actor Actor) (model.EventMarker, error) {
	if !request.EventType.Valid() {
		return model.EventMarker{}, invalid("event_type is not supported", nil)
	}
	event, err := s.store.Events.Get(id)
	if errors.Is(err, repository.ErrNotFound) {
		return event, notFound("event")
	}
	if err != nil {
		return event, internal("get event failed", err)
	}
	trace, err := s.store.Traces.Get(event.TraceID)
	if err != nil {
		return event, internal("get event trace failed", err)
	}
	route, err := s.store.Routes.Get(trace.RouteID)
	if err != nil {
		return event, internal("get event route failed", err)
	}
	before := event
	event.EventType = request.EventType
	if request.DistanceM != nil {
		event.DistanceM = *request.DistanceM
	}
	if event.DistanceM > route.LengthM {
		return event, invalid("reviewed distance exceeds route length", nil)
	}
	now := time.Now()
	event.Reviewed = true
	event.ReviewNote = request.ReviewNote
	event.ReviewedBy = &actor.ID
	event.ReviewedAt = &now
	err = s.store.Transaction(func(tx *repository.Store) error {
		if err := tx.Events.Review(&event); err != nil {
			return err
		}
		return tx.Audits.Create(audit(actor, "event.reviewed", "EventMarker", event.ID, &route.ID, snapshot(before), snapshot(event)))
	})
	if err != nil {
		return event, internal("review event failed", err)
	}
	return event, nil
}

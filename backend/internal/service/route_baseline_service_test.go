package service

import (
	"errors"
	"strings"
	"testing"

	"fiber-otdr-fault-localization/backend/internal/constants"
	"fiber-otdr-fault-localization/backend/internal/dto"
	"fiber-otdr-fault-localization/backend/internal/model"
	"fiber-otdr-fault-localization/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func baselineTestStore(t *testing.T) *repository.Store {
	t.Helper()
	dsn := "file:" + strings.NewReplacer("/", "_", " ", "_").Replace(t.Name()) + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.FiberRoute{}, &model.TraceCapture{}, &model.EventMarker{}, &model.LocalizationCase{}, &model.AuditLog{}); err != nil {
		t.Fatal(err)
	}
	return repository.NewStore(db)
}

func seedRouteWithTraces(t *testing.T, store *repository.Store) (model.FiberRoute, model.TraceCapture, model.TraceCapture) {
	t.Helper()
	route := model.FiberRoute{RouteCode: "BASE01", Name: "Baseline Route", LengthM: 12000, RefractiveIndex: 1.468, LaunchConnector: "SC/APC", RouteStatus: "active"}
	if err := store.Routes.Create(&route); err != nil {
		t.Fatal(err)
	}
	traceA := model.TraceCapture{RouteID: route.ID, WavelengthNM: 1550, PulseWidthNS: 100, SampleIntervalNS: 1, RawPointsJSON: []byte(`[1,2,3]`), NoiseFloorDB: -40, CapturedAt: route.CreatedAt, UploadedBy: 1}
	traceB := model.TraceCapture{RouteID: route.ID, WavelengthNM: 1550, PulseWidthNS: 100, SampleIntervalNS: 1, RawPointsJSON: []byte(`[1,2,3]`), NoiseFloorDB: -40, CapturedAt: route.CreatedAt, UploadedBy: 1}
	if err := store.Traces.Create(&traceA); err != nil {
		t.Fatal(err)
	}
	if err := store.Traces.Create(&traceB); err != nil {
		t.Fatal(err)
	}
	return route, traceA, traceB
}

func baselineActor() Actor {
	return Actor{ID: 7, Username: "reviewer", Role: constants.RoleReviewer, RequestID: "req-baseline"}
}

func openCase(t *testing.T, store *repository.Store, routeID, baselineTraceID, currentTraceID uint, status constants.CaseStatus) {
	t.Helper()
	item := model.LocalizationCase{RouteID: routeID, BaselineTraceID: baselineTraceID, CurrentTraceID: currentTraceID, CaseStatus: status, ParametersJSON: []byte(`{"distance_tolerance_m":25,"loss_increase_db":0.5}`), DifferencesJSON: []byte(`[]`), Version: 1, CreatedBy: 1}
	if err := store.Cases.Create(&item); err != nil {
		t.Fatal(err)
	}
}

func TestSetBaselineRefusedWhenOpenCaseReferencesIt(t *testing.T) {
	store := baselineTestStore(t)
	svc := NewRouteService(store)
	route, traceA, traceB := seedRouteWithTraces(t, store)
	if err := store.Routes.ReplaceBaseline(route.ID, route.Version, traceA.ID); err != nil {
		t.Fatal(err)
	}
	openCase(t, store, route.ID, traceA.ID, traceB.ID, constants.CasePendingReview)

	_, err := svc.SetBaseline(route.ID, traceB.ID, 2, baselineActor())
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Code != CodeBaselineBlocked {
		t.Fatalf("expected BASELINE_CASES_OPEN, got %v", err)
	}
	blockers, _ := appErr.Details["blockers"].([]dto.BaselineBlocker)
	if len(blockers) != 1 {
		t.Fatalf("expected one blocking case in details, got %#v", appErr.Details)
	}

	reloaded, err := store.Routes.Get(route.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.BaselineTraceID == nil || *reloaded.BaselineTraceID != traceA.ID || reloaded.Version != 2 {
		t.Fatalf("baseline must stay unchanged after refusal: %+v", reloaded)
	}
	_, _, _, rejections, err := svc.Get(route.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rejections) != 1 || rejections[0].Reason != "open_cases_referencing_baseline" || len(rejections[0].Blockers) != 1 {
		t.Fatalf("rejection reason not persisted for read-back: %#v", rejections)
	}
}

func TestClosedCaseDoesNotBlockBaselineReplacement(t *testing.T) {
	store := baselineTestStore(t)
	svc := NewRouteService(store)
	route, traceA, traceB := seedRouteWithTraces(t, store)
	if err := store.Routes.ReplaceBaseline(route.ID, route.Version, traceA.ID); err != nil {
		t.Fatal(err)
	}
	openCase(t, store, route.ID, traceA.ID, traceB.ID, constants.CaseClosed)

	updated, err := svc.SetBaseline(route.ID, traceB.ID, 2, baselineActor())
	if err != nil {
		t.Fatalf("closed historical cases keep their baseline and must not block: %v", err)
	}
	if *updated.BaselineTraceID != traceB.ID || updated.Version != 3 {
		t.Fatalf("baseline not replaced: %+v", updated)
	}
}

func TestSetBaselineIsIdempotentForRepeat(t *testing.T) {
	store := baselineTestStore(t)
	svc := NewRouteService(store)
	route, traceA, _ := seedRouteWithTraces(t, store)

	first, err := svc.SetBaseline(route.ID, traceA.ID, 1, baselineActor())
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.SetBaseline(route.ID, traceA.ID, first.Version, baselineActor())
	if err != nil {
		t.Fatalf("repeat of the same switch must succeed: %v", err)
	}
	if second.Version != first.Version || *second.BaselineTraceID != traceA.ID {
		t.Fatalf("repeat must not move version: first=%d second=%d", first.Version, second.Version)
	}
	var audits []model.AuditLog
	if err := store.DB.Find(&audits).Error; err != nil {
		t.Fatal(err)
	}
	changed := 0
	for _, entry := range audits {
		if entry.Action == "route.baseline_changed" {
			changed++
		}
	}
	if changed != 1 {
		t.Fatalf("exactly one successful switch audit expected, got %d", changed)
	}
}

func TestSetBaselineRejectsStaleVersionAfterConcurrentChange(t *testing.T) {
	store := baselineTestStore(t)
	svc := NewRouteService(store)
	route, traceA, traceB := seedRouteWithTraces(t, store)
	if err := store.Routes.ReplaceBaseline(route.ID, route.Version, traceA.ID); err != nil {
		t.Fatal(err)
	}

	// Two reviewers both hold version 2 (baseline = traceA).
	if _, err := svc.SetBaseline(route.ID, traceB.ID, 2, baselineActor()); err != nil {
		t.Fatal(err)
	}
	// The second reviewer submits on the same stale version 2.
	_, err := svc.SetBaseline(route.ID, traceA.ID, 2, baselineActor())
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Code != CodeConflict {
		t.Fatalf("stale version must be STATE_CONFLICT, got %v", err)
	}
	reloaded, err := store.Routes.Get(route.ID)
	if err != nil {
		t.Fatal(err)
	}
	if *reloaded.BaselineTraceID != traceB.ID || reloaded.Version != 3 {
		t.Fatalf("stale request must not overwrite baseline: %+v", reloaded)
	}
	_, _, _, rejections, err := svc.Get(route.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rejections) != 1 || rejections[0].Reason != "baseline_changed_concurrently" {
		t.Fatalf("concurrent-change rejection not recorded: %#v", rejections)
	}
}

func TestSetBaselineRejectsTraceFromAnotherRoute(t *testing.T) {
	store := baselineTestStore(t)
	svc := NewRouteService(store)
	route, _, _ := seedRouteWithTraces(t, store)
	other := model.FiberRoute{RouteCode: "OTHER9", Name: "Other Route", LengthM: 9000, RefractiveIndex: 1.468, LaunchConnector: "SC/APC", RouteStatus: "active"}
	if err := store.Routes.Create(&other); err != nil {
		t.Fatal(err)
	}
	foreign := model.TraceCapture{RouteID: other.ID, WavelengthNM: 1550, PulseWidthNS: 100, SampleIntervalNS: 1, RawPointsJSON: []byte(`[1,2,3]`), NoiseFloorDB: -40, CapturedAt: route.CreatedAt, UploadedBy: 1}
	if err := store.Traces.Create(&foreign); err != nil {
		t.Fatal(err)
	}

	_, err := svc.SetBaseline(route.ID, foreign.ID, 1, baselineActor())
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Code != CodeInvalidInput {
		t.Fatalf("foreign trace must be INVALID_INPUT, got %v", err)
	}
	reloaded, err := store.Routes.Get(route.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.BaselineTraceID != nil {
		t.Fatalf("foreign trace must never become the baseline: %+v", reloaded)
	}
}

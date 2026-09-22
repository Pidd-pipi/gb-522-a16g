package service

import (
	"errors"
	"testing"
	"time"

	"fiber-otdr-fault-localization/backend/internal/constants"
	"fiber-otdr-fault-localization/backend/internal/model"
	"fiber-otdr-fault-localization/backend/internal/repository"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newBaselineTestStore(t *testing.T) *repository.Store {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:baseline-service-"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.FiberRoute{}, &model.TraceCapture{}, &model.EventMarker{}, &model.LocalizationCase{}, &model.AuditLog{}, &model.BaselineChangeAttempt{}); err != nil {
		t.Fatal(err)
	}
	return repository.NewStore(db)
}

func seedBaselineFixture(t *testing.T, store *repository.Store) (model.FiberRoute, model.TraceCapture, model.TraceCapture) {
	t.Helper()
	route := model.FiberRoute{RouteCode: "BL-" + t.Name()[5:9], Name: "基线测试线路", LengthM: 12000, RefractiveIndex: 1.468, LaunchConnector: "SC/APC", RouteStatus: "active"}
	if err := store.Routes.Create(&route); err != nil {
		t.Fatal(err)
	}
	mkTrace := func() model.TraceCapture {
		return model.TraceCapture{RouteID: route.ID, WavelengthNM: 1550, PulseWidthNS: 100, SampleIntervalNS: 10, RawPointsJSON: datatypes.JSON([]byte(`[1,2,3]`)), NoiseFloorDB: -40, CapturedAt: time.Now(), UploadedBy: 1}
	}
	oldTrace, newTrace := mkTrace(), mkTrace()
	if err := store.Traces.Create(&oldTrace); err != nil {
		t.Fatal(err)
	}
	if err := store.Traces.Create(&newTrace); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Routes.ReplaceBaseline(route.ID, 0, oldTrace.ID); err != nil {
		t.Fatal(err)
	}
	route.BaselineTraceID = &oldTrace.ID
	return route, oldTrace, newTrace
}

func mkCase(t *testing.T, store *repository.Store, routeID, baselineID, currentID uint, status constants.CaseStatus) model.LocalizationCase {
	t.Helper()
	item := model.LocalizationCase{RouteID: routeID, BaselineTraceID: baselineID, CurrentTraceID: currentID, CaseStatus: status, ParametersJSON: datatypes.JSON([]byte(`{"distance_tolerance_m":25,"loss_increase_db":0.5}`)), DifferencesJSON: datatypes.JSON([]byte(`[]`)), Version: 1, CreatedBy: 1}
	if err := store.Cases.Create(&item); err != nil {
		t.Fatal(err)
	}
	return item
}

func baselineActor() Actor {
	return Actor{ID: 2, Username: "reviewer", Role: constants.RoleReviewer, RequestID: "req-baseline"}
}

func asRejection(t *testing.T, err error) *BaselineRejectionError {
	t.Helper()
	var rejected *BaselineRejectionError
	if !errors.As(err, &rejected) {
		t.Fatalf("expected BaselineRejectionError, got %T: %v", err, err)
	}
	return rejected
}

func TestSetBaselineBlockedByOpenCaseAndListsIt(t *testing.T) {
	store := newBaselineTestStore(t)
	svc := NewRouteService(store)
	route, oldTrace, newTrace := seedBaselineFixture(t, store)
	openCase := mkCase(t, store, route.ID, oldTrace.ID, newTrace.ID, constants.CasePendingReview)

	_, err := svc.SetBaseline(route.ID, newTrace.ID, baselineActor())
	rejected := asRejection(t, err)
	if rejected.Code != CodeBaselineBlocked || rejected.Reason != baselineReasonOpenCases {
		t.Fatalf("unexpected rejection: code=%s reason=%s", rejected.Code, rejected.Reason)
	}
	if len(rejected.BlockingCases) != 1 || rejected.BlockingCases[0].ID != openCase.ID {
		t.Fatalf("expected blocking case %d, got %+v", openCase.ID, rejected.BlockingCases)
	}

	// The route must still point at the old baseline.
	reloaded, err := store.Routes.Get(route.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.BaselineTraceID == nil || *reloaded.BaselineTraceID != oldTrace.ID {
		t.Fatalf("baseline was rewritten despite blocking case: %+v", reloaded.BaselineTraceID)
	}

	// The rejection reason is readable back through the route detail.
	_, _, _, rejections, err := svc.Get(route.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rejections) != 1 || rejections[0].Reason != baselineReasonOpenCases || len(rejections[0].BlockingCaseIDs) != 1 || rejections[0].BlockingCaseIDs[0] != openCase.ID {
		t.Fatalf("unexpected persisted rejections: %+v", rejections)
	}

	var audits []model.AuditLog
	if err := store.DB.Find(&audits).Error; err != nil {
		t.Fatal(err)
	}
	for _, entry := range audits {
		if entry.Action == "route.baseline_changed" {
			t.Fatal("successful baseline audit must not be written when blocked")
		}
	}
}

func TestSetBaselineSucceedsOnceThenRejectsRepeat(t *testing.T) {
	store := newBaselineTestStore(t)
	svc := NewRouteService(store)
	route, oldTrace, newTrace := seedBaselineFixture(t, store)

	updated, err := svc.SetBaseline(route.ID, newTrace.ID, baselineActor())
	if err != nil {
		t.Fatalf("first replacement should succeed: %v", err)
	}
	if updated.BaselineTraceID == nil || *updated.BaselineTraceID != newTrace.ID {
		t.Fatalf("baseline not updated: %+v", updated.BaselineTraceID)
	}

	// Repeating the same operation must not succeed or rewrite anything.
	_, err = svc.SetBaseline(route.ID, newTrace.ID, baselineActor())
	rejected := asRejection(t, err)
	if rejected.Reason != baselineReasonSameBaseline {
		t.Fatalf("repeat should be same_baseline, got %s", rejected.Reason)
	}

	var changedAudits int64
	if err := store.DB.Model(&model.AuditLog{}).Where("action = ?", "route.baseline_changed").Count(&changedAudits).Error; err != nil {
		t.Fatal(err)
	}
	if changedAudits != 1 {
		t.Fatalf("baseline change must be audited exactly once, got %d", changedAudits)
	}

	// The historical case keeps the old baseline snapshot.
	historical := mkCase(t, store, route.ID, oldTrace.ID, newTrace.ID, constants.CaseClosed)
	if historical.BaselineTraceID != oldTrace.ID {
		t.Fatal("historical case must retain its original baseline")
	}
}

func TestSetBaselineClosedCaseDoesNotBlock(t *testing.T) {
	store := newBaselineTestStore(t)
	svc := NewRouteService(store)
	route, oldTrace, newTrace := seedBaselineFixture(t, store)
	mkCase(t, store, route.ID, oldTrace.ID, newTrace.ID, constants.CaseClosed)

	if _, err := svc.SetBaseline(route.ID, newTrace.ID, baselineActor()); err != nil {
		t.Fatalf("closed cases must not block replacement: %v", err)
	}
}

func TestSetBaselineConcurrentChangeIsRejected(t *testing.T) {
	store := newBaselineTestStore(t)
	svc := NewRouteService(store)
	route, oldTrace, newTrace := seedBaselineFixture(t, store)

	third := model.TraceCapture{RouteID: route.ID, WavelengthNM: 1625, PulseWidthNS: 100, SampleIntervalNS: 10, RawPointsJSON: datatypes.JSON([]byte(`[1,2,3]`)), NoiseFloorDB: -41, CapturedAt: time.Now(), UploadedBy: 1}
	if err := store.Traces.Create(&third); err != nil {
		t.Fatal(err)
	}

	// Simulate a competing committed change between request-entry checks and
	// the guarded transaction; the in-transaction re-read then observes a
	// baseline different from the one validated at request entry.
	svc.beforeBaselineTx = func(routeID uint) {
		if _, err := store.Routes.ReplaceBaseline(routeID, oldTrace.ID, third.ID); err != nil {
			t.Fatal(err)
		}
	}

	_, err := svc.SetBaseline(route.ID, newTrace.ID, baselineActor())
	rejected := asRejection(t, err)
	if rejected.Reason != baselineReasonConcurrentChange {
		t.Fatalf("expected concurrent change rejection, got reason=%s", rejected.Reason)
	}
	reloaded, err := store.Routes.Get(route.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.BaselineTraceID == nil || *reloaded.BaselineTraceID != third.ID {
		t.Fatal("concurrent winner's baseline must be preserved")
	}
}

func TestSetBaselineRollsBackWhenCaseAppearsDuringUpdate(t *testing.T) {
	store := newBaselineTestStore(t)
	svc := NewRouteService(store)
	route, oldTrace, newTrace := seedBaselineFixture(t, store)

	// A case referencing the current baseline is committed right after the
	// guarded update, before the second in-transaction check. The change must
	// roll back and be reported as an open-cases block.
	svc.afterBaselineUpdate = func(tx *repository.Store, routeID, oldBaselineID uint) {
		item := model.LocalizationCase{RouteID: routeID, BaselineTraceID: oldBaselineID, CurrentTraceID: newTrace.ID, CaseStatus: constants.CaseDraft, ParametersJSON: datatypes.JSON([]byte(`{"distance_tolerance_m":25,"loss_increase_db":0.5}`)), DifferencesJSON: datatypes.JSON([]byte(`[]`)), Version: 1, CreatedBy: 1}
		if err := tx.Cases.Create(&item); err != nil {
			t.Fatal(err)
		}
	}

	_, err := svc.SetBaseline(route.ID, newTrace.ID, baselineActor())
	rejected := asRejection(t, err)
	if rejected.Reason != baselineReasonOpenCases {
		t.Fatalf("expected open_cases rejection, got reason=%s", rejected.Reason)
	}
	if len(rejected.BlockingCases) != 1 {
		t.Fatalf("expected one blocking case, got %d", len(rejected.BlockingCases))
	}
	reloaded, err := store.Routes.Get(route.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.BaselineTraceID == nil || *reloaded.BaselineTraceID != oldTrace.ID {
		t.Fatal("baseline update must roll back when a case appears mid-transaction")
	}
	_, _, _, rejections, err := svc.Get(route.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rejections) != 1 || rejections[0].Reason != baselineReasonOpenCases {
		t.Fatalf("expected a persisted rejection, got %+v", rejections)
	}
}

func TestSetBaselineRejectsTraceFromAnotherRoute(t *testing.T) {
	store := newBaselineTestStore(t)
	svc := NewRouteService(store)
	route, oldTrace, _ := seedBaselineFixture(t, store)
	other := model.FiberRoute{RouteCode: "BL-OTHER", Name: "其他线路", LengthM: 9000, RefractiveIndex: 1.468, LaunchConnector: "SC/APC", RouteStatus: "active"}
	if err := store.Routes.Create(&other); err != nil {
		t.Fatal(err)
	}
	foreign := model.TraceCapture{RouteID: other.ID, WavelengthNM: 1550, PulseWidthNS: 100, SampleIntervalNS: 10, RawPointsJSON: datatypes.JSON([]byte(`[1,2,3]`)), NoiseFloorDB: -40, CapturedAt: time.Now(), UploadedBy: 1}
	if err := store.Traces.Create(&foreign); err != nil {
		t.Fatal(err)
	}

	_, err := svc.SetBaseline(route.ID, foreign.ID, baselineActor())
	rejected := asRejection(t, err)
	if rejected.Code != CodeInvalidInput || rejected.Reason != baselineReasonTraceNotOnRoute {
		t.Fatalf("unexpected rejection: code=%s reason=%s", rejected.Code, rejected.Reason)
	}
	reloaded, _ := store.Routes.Get(route.ID)
	if reloaded.BaselineTraceID == nil || *reloaded.BaselineTraceID != oldTrace.ID {
		t.Fatal("foreign trace must never become the baseline")
	}
}

package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"fiber-otdr-fault-localization/backend/internal/dto"
	"fiber-otdr-fault-localization/backend/internal/model"
	"fiber-otdr-fault-localization/backend/internal/repository"
)

type RouteService struct {
	store *repository.Store
	// beforeBaselineTx and afterBaselineUpdate are deterministic test seams;
	// they let tests simulate concurrent changes around the guarded transaction
	// (a competing baseline change, or a case row present at the second check).
	beforeBaselineTx    func(routeID uint)
	afterBaselineUpdate func(tx *repository.Store, routeID, oldBaselineID uint)
}

func NewRouteService(store *repository.Store) *RouteService { return &RouteService{store: store} }

func (s *RouteService) Create(request dto.CreateRouteRequest, actor Actor) (model.FiberRoute, error) {
	status := request.RouteStatus
	if status == "" {
		status = "active"
	}
	route := model.FiberRoute{RouteCode: strings.ToUpper(strings.TrimSpace(request.RouteCode)), Name: strings.TrimSpace(request.Name), LengthM: request.LengthM, RefractiveIndex: request.RefractiveIndex, LaunchConnector: strings.TrimSpace(request.LaunchConnector), RouteStatus: status}
	err := s.store.Transaction(func(tx *repository.Store) error {
		if _, err := tx.Routes.GetByCode(route.RouteCode); err == nil {
			return conflict("route code already exists", err)
		} else if !errors.Is(err, repository.ErrNotFound) {
			return err
		}
		if err := tx.Routes.Create(&route); err != nil {
			return err
		}
		return tx.Audits.Create(audit(actor, "route.created", "FiberRoute", route.ID, &route.ID, "{}", snapshot(route)))
	})
	if err != nil {
		var appErr *AppError
		if errors.As(err, &appErr) {
			return route, err
		}
		return route, internal("create route failed", err)
	}
	return route, nil
}

func (s *RouteService) List(query dto.RouteQuery) ([]model.FiberRoute, dto.Pagination, error) {
	normalizePage(&query.Page, &query.PageSize)
	items, total, err := s.store.Routes.List(query)
	if err != nil {
		return nil, dto.Pagination{}, internal("list routes failed", err)
	}
	return items, dto.Pagination{Page: query.Page, PageSize: query.PageSize, Total: total}, nil
}

func (s *RouteService) Get(id uint) (model.FiberRoute, []model.TraceCapture, []dto.PendingCaseView, []dto.BaselineRejectionView, error) {
	route, err := s.store.Routes.Get(id)
	if errors.Is(err, repository.ErrNotFound) {
		return route, nil, nil, nil, notFound("route")
	}
	if err != nil {
		return route, nil, nil, nil, internal("get route failed", err)
	}
	traces, _, err := s.store.Traces.List(dto.TraceQuery{RouteID: &id, Page: 1, PageSize: 100})
	if err != nil {
		return route, nil, nil, nil, internal("list route traces failed", err)
	}
	openCases, err := s.store.Cases.OpenByRoute(id)
	if err != nil {
		return route, nil, nil, nil, internal("list pending cases failed", err)
	}
	pending := make([]dto.PendingCaseView, 0, len(openCases))
	for _, item := range openCases {
		pending = append(pending, dto.PendingCaseView{ID: item.ID, BaselineTraceID: item.BaselineTraceID, CurrentTraceID: item.CurrentTraceID, CaseStatus: string(item.CaseStatus), Conclusion: item.Conclusion, CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339)})
	}
	attempts, err := s.store.BaselineAttempts.ListByRoute(id, 10)
	if err != nil {
		return route, nil, nil, nil, internal("list baseline rejections failed", err)
	}
	rejections := make([]dto.BaselineRejectionView, 0, len(attempts))
	for _, attempt := range attempts {
		rejections = append(rejections, toRejectionView(attempt))
	}
	return route, traces, pending, rejections, nil
}

func (s *RouteService) Update(id uint, request dto.UpdateRouteRequest, actor Actor) (model.FiberRoute, error) {
	before, err := s.store.Routes.Get(id)
	if errors.Is(err, repository.ErrNotFound) {
		return before, notFound("route")
	}
	if err != nil {
		return before, internal("get route failed", err)
	}
	beforeSnapshot := snapshot(before)
	after := before
	if request.Name != nil {
		after.Name = strings.TrimSpace(*request.Name)
	}
	if request.LengthM != nil {
		after.LengthM = *request.LengthM
	}
	if request.RefractiveIndex != nil {
		after.RefractiveIndex = *request.RefractiveIndex
	}
	if request.LaunchConnector != nil {
		after.LaunchConnector = strings.TrimSpace(*request.LaunchConnector)
	}
	if request.RouteStatus != nil {
		after.RouteStatus = *request.RouteStatus
	}
	err = s.store.Transaction(func(tx *repository.Store) error {
		if err := tx.Routes.Update(&after); err != nil {
			return err
		}
		return tx.Audits.Create(audit(actor, "route.updated", "FiberRoute", id, &id, beforeSnapshot, snapshot(after)))
	})
	if err != nil {
		return after, internal("update route failed", err)
	}
	return after, nil
}

// BaselineRejectionError explains a refused baseline replacement. The blocked
// cases are listed so reviewers can resolve them before retrying.
type BaselineRejectionError struct {
	AppError
	Reason        string                     `json:"reason"`
	BlockingCases []dto.BlockingCaseView     `json:"blocking_cases,omitempty"`
	Rejection     *dto.BaselineRejectionView `json:"rejection,omitempty"`
}

// Unwrap lets errors.As find the embedded AppError so generic error handling
// still sees the HTTP status and code.
func (e *BaselineRejectionError) Unwrap() error { return &e.AppError }

const (
	CodeBaselineBlocked = "BASELINE_REPLACEMENT_BLOCKED"

	baselineReasonOpenCases        = "open_cases"
	baselineReasonTraceNotOnRoute  = "trace_not_on_route"
	baselineReasonSameBaseline     = "same_baseline"
	baselineReasonConcurrentChange = "baseline_concurrent_change"
	baselineResultRejected         = "rejected"
	baselineResultSucceeded        = "succeeded"
)

// SetBaseline replaces a route's review baseline. Validation, the guarded
// update and the audit record run in one transaction; refused attempts commit
// a rejection record in that same transaction so the reason survives a refresh.
// Repeating a completed replacement (same trace) is refused, so it succeeds at
// most once.
func (s *RouteService) SetBaseline(routeID, traceID uint, actor Actor) (model.FiberRoute, error) {
	route, err := s.store.Routes.Get(routeID)
	if errors.Is(err, repository.ErrNotFound) {
		return route, notFound("route")
	}
	if err != nil {
		return route, internal("get route failed", err)
	}
	targetBelongs, err := s.store.Traces.BelongsToRoute(traceID, routeID)
	if err != nil {
		return route, internal("validate baseline trace failed", err)
	}
	currentID := uint(0)
	if route.BaselineTraceID != nil {
		currentID = *route.BaselineTraceID
	}

	if !targetBelongs {
		attempt := newRejection(routeID, traceID, route.BaselineTraceID, baselineReasonTraceNotOnRoute, actor, nil)
		if txErr := s.store.Transaction(func(tx *repository.Store) error {
			if err := tx.BaselineAttempts.Create(attempt); err != nil {
				return err
			}
			return tx.Audits.Create(audit(actor, "route.baseline_rejected", "FiberRoute", routeID, &routeID, snapshot(map[string]any{"baseline_trace_id": route.BaselineTraceID}), snapshot(map[string]any{"requested_trace_id": traceID, "reason": baselineReasonTraceNotOnRoute})))
		}); txErr != nil {
			return route, internal("record baseline rejection failed", txErr)
		}
		return route, rejectionError(&rejectDecision{status: http.StatusBadRequest, code: CodeInvalidInput, message: "baseline trace must belong to the route", attempt: *attempt})
	}
	if currentID == traceID {
		attempt := newRejection(routeID, traceID, route.BaselineTraceID, baselineReasonSameBaseline, actor, nil)
		if txErr := s.store.Transaction(func(tx *repository.Store) error {
			if err := tx.BaselineAttempts.Create(attempt); err != nil {
				return err
			}
			return tx.Audits.Create(audit(actor, "route.baseline_rejected", "FiberRoute", routeID, &routeID, snapshot(map[string]any{"baseline_trace_id": route.BaselineTraceID}), snapshot(map[string]any{"requested_trace_id": traceID, "reason": baselineReasonSameBaseline})))
		}); txErr != nil {
			return route, internal("record baseline rejection failed", txErr)
		}
		return route, rejectionError(&rejectDecision{status: http.StatusConflict, code: CodeConflict, message: "the selected trace is already the current baseline", attempt: *attempt})
	}

	var updated model.FiberRoute
	var decision *rejectDecision
	if s.beforeBaselineTx != nil {
		s.beforeBaselineTx(routeID)
	}
	txErr := s.store.Transaction(func(tx *repository.Store) error {
		// Re-read inside the transaction so both the open-case check and the
		// guarded update observe the same committed baseline.
		fresh, err := tx.Routes.Get(routeID)
		if err != nil {
			return err
		}
		freshCurrent := uint(0)
		if fresh.BaselineTraceID != nil {
			freshCurrent = *fresh.BaselineTraceID
		}
		if freshCurrent == traceID {
			attempt := newRejection(routeID, traceID, fresh.BaselineTraceID, baselineReasonSameBaseline, actor, nil)
			if err := tx.BaselineAttempts.Create(attempt); err != nil {
				return err
			}
			if err := tx.Audits.Create(rejectedAudit(actor, routeID, traceID, fresh.BaselineTraceID, baselineReasonSameBaseline)); err != nil {
				return err
			}
			decision = &rejectDecision{status: http.StatusConflict, code: CodeConflict, message: "the selected trace is already the current baseline", attempt: *attempt}
			return nil
		}
		if freshCurrent != currentID {
			attempt := newRejection(routeID, traceID, fresh.BaselineTraceID, baselineReasonConcurrentChange, actor, nil)
			if err := tx.BaselineAttempts.Create(attempt); err != nil {
				return err
			}
			if err := tx.Audits.Create(rejectedAudit(actor, routeID, traceID, fresh.BaselineTraceID, baselineReasonConcurrentChange)); err != nil {
				return err
			}
			decision = &rejectDecision{status: http.StatusConflict, code: CodeConflict, message: "the route baseline changed concurrently; reload the route and retry", attempt: *attempt}
			return nil
		}
		openCases, err := tx.Cases.OpenCasesByBaseline(routeID, currentID)
		if err != nil {
			return err
		}
		if len(openCases) > 0 {
			decision, err = commitOpenCasesRejection(tx, actor, routeID, traceID, fresh.BaselineTraceID, openCases)
			if err != nil {
				return err
			}
			return nil
		}
		changed, err := tx.Routes.ReplaceBaseline(routeID, currentID, traceID)
		if err != nil {
			return err
		}
		if !changed {
			attempt := newRejection(routeID, traceID, fresh.BaselineTraceID, baselineReasonConcurrentChange, actor, nil)
			if err := tx.BaselineAttempts.Create(attempt); err != nil {
				return err
			}
			if err := tx.Audits.Create(rejectedAudit(actor, routeID, traceID, fresh.BaselineTraceID, baselineReasonConcurrentChange)); err != nil {
				return err
			}
			decision = &rejectDecision{status: http.StatusConflict, code: CodeConflict, message: "the route baseline changed concurrently; reload the route and retry", attempt: *attempt}
			return nil
		}
		// Second check after the guarded update: under read-committed isolation
		// a case may have been committed between the first check and the update.
		// Returning an error rolls the whole transaction (including the baseline
		// update) back, so the route keeps its old baseline.
		if s.afterBaselineUpdate != nil {
			s.afterBaselineUpdate(tx, routeID, currentID)
		}
		lateCases, err := tx.Cases.OpenCasesByBaseline(routeID, currentID)
		if err != nil {
			return err
		}
		if len(lateCases) > 0 {
			return &rolledBackRejection{decision: buildOpenCasesDecision(actor, routeID, traceID, fresh.BaselineTraceID, lateCases)}
		}
		attempt := &model.BaselineChangeAttempt{RouteID: routeID, RequestedTraceID: traceID, CurrentTraceID: fresh.BaselineTraceID, Result: baselineResultSucceeded, Reason: "success", ActorID: actor.ID, ActorName: actor.Username, RequestID: actor.RequestID, BlockingCaseIDs: "[]"}
		if err := tx.BaselineAttempts.Create(attempt); err != nil {
			return err
		}
		before := snapshot(map[string]any{"baseline_trace_id": fresh.BaselineTraceID})
		if err := tx.Audits.Create(audit(actor, "route.baseline_changed", "FiberRoute", routeID, &routeID, before, snapshot(map[string]any{"baseline_trace_id": traceID}))); err != nil {
			return err
		}
		updated, err = tx.Routes.Get(routeID)
		return err
	})
	if txErr != nil {
		var rolledBack *rolledBackRejection
		if errors.As(txErr, &rolledBack) {
			// The guarded transaction rolled back; record its rejection reason
			// in a fresh transaction so it can be read after a refresh.
			if persistErr := s.persistRejection(rolledBack.decision); persistErr != nil {
				return route, internal("record baseline rejection failed", persistErr)
			}
			return route, rejectionError(rolledBack.decision)
		}
		return route, internal("set baseline failed", txErr)
	}
	if decision != nil {
		return route, rejectionError(decision)
	}
	return updated, nil
}

// rolledBackRejection aborts the guarded transaction while carrying the
// rejection decision the caller persists afterwards.
type rolledBackRejection struct{ decision *rejectDecision }

func (e *rolledBackRejection) Error() string { return e.decision.message }

// buildOpenCasesDecision assembles the rejection decision (without writing)
// for a request blocked by open cases referencing the current baseline.
func buildOpenCasesDecision(actor Actor, routeID, traceID uint, current *uint, openCases []model.LocalizationCase) *rejectDecision {
	blocking := make([]dto.BlockingCaseView, 0, len(openCases))
	for _, item := range openCases {
		blocking = append(blocking, dto.BlockingCaseView{ID: item.ID, CaseStatus: string(item.CaseStatus), CurrentTraceID: item.CurrentTraceID, CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339)})
	}
	attempt := newRejection(routeID, traceID, current, baselineReasonOpenCases, actor, blocking)
	return &rejectDecision{status: http.StatusConflict, code: CodeBaselineBlocked, message: "open localization cases still reference the current baseline; close or confirm them before replacing it", attempt: *attempt, blocking: blocking}
}

// commitOpenCasesRejection stages the rejection record and its audit entry in
// the guarded transaction for the pre-update check branch.
func commitOpenCasesRejection(tx *repository.Store, actor Actor, routeID, traceID uint, current *uint, openCases []model.LocalizationCase) (*rejectDecision, error) {
	decision := buildOpenCasesDecision(actor, routeID, traceID, current, openCases)
	attempt := decision.attempt
	if err := tx.BaselineAttempts.Create(&attempt); err != nil {
		return nil, err
	}
	decision.attempt = attempt
	if err := tx.Audits.Create(rejectedAudit(actor, routeID, traceID, current, baselineReasonOpenCases)); err != nil {
		return nil, err
	}
	return decision, nil
}

// persistRejection writes a rejection that had to roll back the guarded
// transaction, in a separate transaction.
func (s *RouteService) persistRejection(decision *rejectDecision) error {
	return s.store.Transaction(func(tx *repository.Store) error {
		attempt := decision.attempt
		if err := tx.BaselineAttempts.Create(&attempt); err != nil {
			return err
		}
		decision.attempt = attempt
		return tx.Audits.Create(rejectedAudit(Actor{ID: attempt.ActorID, Username: attempt.ActorName, RequestID: attempt.RequestID}, attempt.RouteID, attempt.RequestedTraceID, attempt.CurrentTraceID, attempt.Reason))
	})
}

type rejectDecision struct {
	status   int
	code     string
	message  string
	attempt  model.BaselineChangeAttempt
	blocking []dto.BlockingCaseView
}

func rejectedAudit(actor Actor, routeID, requestedTraceID uint, current *uint, reason string) *model.AuditLog {
	return audit(actor, "route.baseline_rejected", "FiberRoute", routeID, &routeID,
		snapshot(map[string]any{"baseline_trace_id": current}),
		snapshot(map[string]any{"requested_trace_id": requestedTraceID, "reason": reason}))
}

func newRejection(routeID, traceID uint, current *uint, reason string, actor Actor, blocking []dto.BlockingCaseView) *model.BaselineChangeAttempt {
	ids := make([]uint, 0, len(blocking))
	for _, item := range blocking {
		ids = append(ids, item.ID)
	}
	return &model.BaselineChangeAttempt{RouteID: routeID, RequestedTraceID: traceID, CurrentTraceID: current, Result: baselineResultRejected, Reason: reason, ActorID: actor.ID, ActorName: actor.Username, RequestID: actor.RequestID, BlockingCaseIDs: snapshot(ids)}
}

func rejectionError(decision *rejectDecision) error {
	view := toRejectionView(decision.attempt)
	return &BaselineRejectionError{
		AppError:      AppError{Code: decision.code, Status: decision.status, Message: decision.message},
		Reason:        decision.attempt.Reason,
		BlockingCases: decision.blocking,
		Rejection:     &view,
	}
}

func toRejectionView(attempt model.BaselineChangeAttempt) dto.BaselineRejectionView {
	var ids []uint
	if err := json.Unmarshal([]byte(attempt.BlockingCaseIDs), &ids); err != nil {
		ids = []uint{}
	}
	return dto.BaselineRejectionView{ID: attempt.ID, RequestedTraceID: attempt.RequestedTraceID, CurrentTraceID: attempt.CurrentTraceID, Result: attempt.Result, Reason: attempt.Reason, ActorName: attempt.ActorName, BlockingCaseIDs: ids, CreatedAt: attempt.CreatedAt.UTC().Format(time.RFC3339)}
}

func normalizePage(page, size *int) {
	if *page < 1 {
		*page = 1
	}
	if *size < 1 {
		*size = 20
	}
	if *size > 100 {
		*size = 100
	}
}
func snapshot(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("{\"summary_error\":%q}", err.Error())
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return string(encoded)
	}
	scrubSnapshot(decoded)
	clean, err := json.Marshal(decoded)
	if err != nil {
		return string(encoded)
	}
	return string(clean)
}

func scrubSnapshot(value any) {
	const redacted = "[REDACTED]"
	sensitive := func(key string) bool {
		key = strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "-", "_"), " ", "_"))
		for _, token := range []string{"password", "passwd", "token", "authorization", "secret", "credential", "private_key", "apikey", "api_key"} {
			if strings.Contains(key, token) {
				return true
			}
		}
		return false
	}
	var walk func(any)
	walk = func(node any) {
		switch current := node.(type) {
		case map[string]any:
			for key, child := range current {
				if sensitive(key) {
					current[key] = redacted
					continue
				}
				walk(child)
			}
		case []any:
			for _, child := range current {
				walk(child)
			}
		}
	}
	walk(value)
}
func audit(actor Actor, action, resource string, resourceID uint, routeID *uint, before, after string) *model.AuditLog {
	return &model.AuditLog{ActorID: actor.ID, ActorName: actor.Username, Action: action, ResourceType: resource, ResourceID: resourceID, RouteID: routeID, RequestID: actor.RequestID, Before: before, After: after}
}

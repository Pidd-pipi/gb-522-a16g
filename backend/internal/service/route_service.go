package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"fiber-otdr-fault-localization/backend/internal/dto"
	"fiber-otdr-fault-localization/backend/internal/model"
	"fiber-otdr-fault-localization/backend/internal/repository"
)

type RouteService struct{ store *repository.Store }

func NewRouteService(store *repository.Store) *RouteService { return &RouteService{store} }

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

func (s *RouteService) Get(id uint) (model.FiberRoute, []model.TraceCapture, []model.LocalizationCase, []dto.BaselineRejection, error) {
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
	pending, err := s.store.Cases.OpenByRoute(id)
	if err != nil {
		return route, nil, nil, nil, internal("list pending route cases failed", err)
	}
	rejectionEntries, err := s.store.Audits.ListRejectedBaselineChanges(id, 20)
	if err != nil {
		return route, nil, nil, nil, internal("list baseline rejections failed", err)
	}
	rejections := decodeBaselineRejections(rejectionEntries)
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

// SetBaseline replaces the review baseline of a route. Blocking-case check,
// conditional update and audit run inside a single transaction, so either the
// baseline changes with its audit entry or nothing changes at all.
func (s *RouteService) SetBaseline(routeID, traceID, version uint, actor Actor) (model.FiberRoute, error) {
	belongs, err := s.store.Traces.BelongsToRoute(traceID, routeID)
	if err != nil {
		return model.FiberRoute{}, internal("validate baseline trace failed", err)
	}
	if !belongs {
		// A trace from another route must never be written onto this route;
		// persist the refusal so reviewers can read the reason back later.
		rejection := dto.BaselineRejection{Reason: "trace_not_on_route", Message: "baseline trace must belong to the route", RequestedTrace: traceID}
		if auditErr := s.recordBaselineRejection(routeID, actor, rejection); auditErr != nil {
			return model.FiberRoute{}, internal("record baseline rejection failed", auditErr)
		}
		return model.FiberRoute{}, invalid("baseline trace must belong to the route", nil)
	}

	var result model.FiberRoute
	var decision error
	err = s.store.Transaction(func(tx *repository.Store) error {
		route, err := tx.Routes.GetForUpdate(routeID)
		if errors.Is(err, repository.ErrNotFound) {
			return notFound("route")
		}
		if err != nil {
			return internal("get route failed", err)
		}
		// Idempotency: repeating the same successful request changes nothing
		// and writes no audit row, so a double click succeeds only once.
		if route.BaselineTraceID != nil && *route.BaselineTraceID == traceID {
			result = route
			return errBaselineUnchanged
		}
		// Stale route snapshot: another transaction already moved the baseline.
		// The refusal is audited inside this transaction and committed with it;
		// returning the decision after commit keeps the audit row durable.
		if route.Version != version {
			rejection := dto.BaselineRejection{Reason: "baseline_changed_concurrently", Message: "route baseline changed concurrently; reload the route and retry", RequestedTrace: traceID}
			if err := createBaselineRejectionAudit(tx, routeID, actor, route.Version, rejection); err != nil {
				return err
			}
			decision = conflict("route baseline changed concurrently; reload the route and retry", nil)
			return nil
		}
		referenced := []uint{}
		if route.BaselineTraceID != nil {
			referenced = append(referenced, *route.BaselineTraceID)
		}
		blocking, err := tx.Cases.OpenCasesReferencingTraces(routeID, referenced)
		if err != nil {
			return internal("check open baseline cases failed", err)
		}
		if len(blocking) > 0 {
			blockers := make([]dto.BaselineBlocker, 0, len(blocking))
			for _, item := range blocking {
				blockers = append(blockers, dto.BaselineBlocker{CaseID: item.ID, CaseStatus: string(item.CaseStatus), BaselineTraceID: item.BaselineTraceID, CurrentTraceID: item.CurrentTraceID, CreatedAt: item.CreatedAt.Format(time.RFC3339)})
			}
			rejection := dto.BaselineRejection{Reason: "open_cases_referencing_baseline", Message: fmt.Sprintf("%d open case(s) still reference the current baseline", len(blockers)), RequestedTrace: traceID, Blockers: blockers}
			if err := createBaselineRejectionAudit(tx, routeID, actor, route.Version, rejection); err != nil {
				return err
			}
			decision = baselineBlocked("open cases still reference the current baseline", blockers)
			return nil
		}
		before := snapshot(map[string]any{"baseline_trace_id": route.BaselineTraceID, "version": route.Version})
		if err := tx.Routes.ReplaceBaseline(routeID, version, traceID); err != nil {
			if errors.Is(err, repository.ErrConcurrentChange) {
				return conflict("route baseline changed concurrently; reload the route and retry", err)
			}
			return internal("set baseline failed", err)
		}
		if err := tx.Audits.Create(audit(actor, "route.baseline_changed", "FiberRoute", routeID, &routeID, before, snapshot(map[string]any{"baseline_trace_id": traceID, "version": version + 1}))); err != nil {
			return err
		}
		route.BaselineTraceID = &traceID
		route.Version = version + 1
		result = route
		return nil
	})
	if errors.Is(err, errBaselineUnchanged) {
		return result, nil
	}
	if err != nil {
		return model.FiberRoute{}, err
	}
	if decision != nil {
		return model.FiberRoute{}, decision
	}
	return result, nil
}

var errBaselineUnchanged = errors.New("baseline already points at the requested trace")

func (s *RouteService) recordBaselineRejection(routeID uint, actor Actor, rejection dto.BaselineRejection) error {
	return s.store.Transaction(func(tx *repository.Store) error {
		route, err := tx.Routes.Get(routeID)
		if errors.Is(err, repository.ErrNotFound) {
			return notFound("route")
		}
		if err != nil {
			return internal("get route failed", err)
		}
		return createBaselineRejectionAudit(tx, routeID, actor, route.Version, rejection)
	})
}

func createBaselineRejectionAudit(tx *repository.Store, routeID uint, actor Actor, routeVersion uint, rejection dto.BaselineRejection) error {
	// RequestID/ActorName live in dedicated audit columns; the snapshot keeps
	// only the decision payload so the id columns stay verbatim.
	entry := audit(actor, "route.baseline_change_rejected", "FiberRoute", routeID, &routeID,
		snapshot(map[string]any{"version": routeVersion}),
		snapshot(rejection))
	return tx.Audits.Create(entry)
}

func decodeBaselineRejections(entries []model.AuditLog) []dto.BaselineRejection {
	rejections := make([]dto.BaselineRejection, 0, len(entries))
	for _, entry := range entries {
		var rejection dto.BaselineRejection
		if err := json.Unmarshal([]byte(entry.After), &rejection); err != nil {
			continue
		}
		if rejection.RequestID == "" {
			rejection.RequestID = entry.RequestID
		}
		if rejection.ActorName == "" {
			rejection.ActorName = entry.ActorName
		}
		rejection.CreatedAt = entry.CreatedAt.Format(time.RFC3339)
		rejections = append(rejections, rejection)
	}
	return rejections
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

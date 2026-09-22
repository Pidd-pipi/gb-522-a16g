package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"fiber-otdr-fault-localization/backend/internal/algorithm"
	"fiber-otdr-fault-localization/backend/internal/constants"
	"fiber-otdr-fault-localization/backend/internal/dto"
	"fiber-otdr-fault-localization/backend/internal/model"
	"fiber-otdr-fault-localization/backend/internal/repository"
	"gorm.io/datatypes"
)

type CaseService struct{ store *repository.Store }

const analysisLeaseTimeout = 10 * time.Minute

func NewCaseService(store *repository.Store) *CaseService { return &CaseService{store} }

func (s *CaseService) Create(request dto.CreateCaseRequest, actor Actor) (model.LocalizationCase, error) {
	if request.BaselineTraceID == request.CurrentTraceID {
		return model.LocalizationCase{}, invalid("baseline and current traces must differ", nil)
	}
	for _, traceID := range []uint{request.BaselineTraceID, request.CurrentTraceID} {
		belongs, err := s.store.Traces.BelongsToRoute(traceID, request.RouteID)
		if err != nil {
			return model.LocalizationCase{}, internal("validate case traces failed", err)
		}
		if !belongs {
			return model.LocalizationCase{}, invalid("both traces must belong to the selected route", nil)
		}
	}
	tolerance := request.DistanceToleranceM
	if tolerance == 0 {
		tolerance = 25
	}
	loss := request.LossIncreaseDB
	if loss == 0 {
		loss = 0.5
	}
	params, _ := json.Marshal(dto.CaseParameters{DistanceToleranceM: tolerance, LossIncreaseDB: loss})
	item := model.LocalizationCase{RouteID: request.RouteID, BaselineTraceID: request.BaselineTraceID, CurrentTraceID: request.CurrentTraceID, CaseStatus: constants.CaseDraft, ParametersJSON: datatypes.JSON(params), DifferencesJSON: datatypes.JSON([]byte("[]")), Version: 1, CreatedBy: actor.ID}
	err := s.store.Transaction(func(tx *repository.Store) error {
		if err := tx.Cases.Create(&item); err != nil {
			return err
		}
		return tx.Audits.Create(audit(actor, "case.created", "LocalizationCase", item.ID, &item.RouteID, "{}", snapshot(item)))
	})
	if err != nil {
		return item, internal("create case failed", err)
	}
	return item, nil
}

func (s *CaseService) List(query dto.CaseQuery) ([]model.LocalizationCase, dto.Pagination, error) {
	normalizePage(&query.Page, &query.PageSize)
	items, total, err := s.store.Cases.List(query)
	if err != nil {
		return nil, dto.Pagination{}, internal("list cases failed", err)
	}
	return items, dto.Pagination{Page: query.Page, PageSize: query.PageSize, Total: total}, nil
}

func (s *CaseService) Get(id uint) (model.LocalizationCase, []algorithm.Difference, error) {
	item, err := s.store.Cases.Get(id)
	if errors.Is(err, repository.ErrNotFound) {
		return item, nil, notFound("case")
	}
	if err != nil {
		return item, nil, internal("get case failed", err)
	}
	var differences []algorithm.Difference
	if len(item.DifferencesJSON) > 0 {
		if err := json.Unmarshal(item.DifferencesJSON, &differences); err != nil {
			return item, nil, internal("decode case differences failed", err)
		}
	}
	return item, differences, nil
}

func (s *CaseService) Analyze(id uint, request dto.AnalyzeCaseRequest, actor Actor) (model.LocalizationCase, error) {
	item, err := s.store.Cases.Get(id)
	if errors.Is(err, repository.ErrNotFound) {
		return item, notFound("case")
	}
	if err != nil {
		return item, internal("get case failed", err)
	}
	if item.CaseStatus == constants.CaseAnalyzing {
		var recovered bool
		recoverErr := s.store.Transaction(func(tx *repository.Store) error {
			var recoverErr error
			recovered, recoverErr = tx.Cases.RecoverStaleAnalysis(id, time.Now().Add(-analysisLeaseTimeout))
			if recoverErr != nil || !recovered {
				return recoverErr
			}
			return tx.Audits.Create(audit(actor, "case.analysis_recovered", "LocalizationCase", id, &item.RouteID, snapshot(map[string]any{"status": constants.CaseAnalyzing}), snapshot(map[string]any{"status": constants.CaseDraft, "reason": "worker_expired"})))
		})
		if recoverErr != nil {
			return item, internal("recover interrupted analysis failed", recoverErr)
		}
		if recovered {
			item, err = s.store.Cases.Get(id)
			if err != nil {
				return item, internal("reload recovered case failed", err)
			}
		}
	}
	if item.CaseStatus == constants.CaseClosed {
		return item, conflict("closed cases cannot be analyzed", nil)
	}
	if !constants.CanTransition(item.CaseStatus, constants.CaseAnalyzing) {
		return item, conflict("case must be in draft before analysis", nil)
	}
	if err := s.store.Transaction(func(tx *repository.Store) error {
		if err := tx.Cases.Transition(item.ID, item.Version, constants.CaseDraft, constants.CaseAnalyzing, map[string]any{"analysis_error": ""}); err != nil {
			return err
		}
		return tx.Audits.Create(audit(actor, "case.analysis_started", "LocalizationCase", item.ID, &item.RouteID, snapshot(map[string]any{"status": item.CaseStatus}), snapshot(map[string]any{"status": constants.CaseAnalyzing})))
	}); err != nil {
		return item, conflict("case changed while analysis was starting", err)
	}
	item.CaseStatus = constants.CaseAnalyzing
	item.Version++
	tolerance, loss := request.DistanceToleranceM, request.LossIncreaseDB
	var saved dto.CaseParameters
	_ = json.Unmarshal(item.ParametersJSON, &saved)
	if tolerance == 0 {
		tolerance = saved.DistanceToleranceM
	}
	if tolerance == 0 {
		tolerance = 25
	}
	if loss == 0 {
		loss = saved.LossIncreaseDB
	}
	if loss == 0 {
		loss = 0.5
	}
	differences, analysisErr := s.compareEvents(item, tolerance, loss)
	if analysisErr != nil {
		_ = s.store.Transaction(func(tx *repository.Store) error {
			if err := tx.Cases.SaveAnalysis(item.ID, constants.CaseDraft, map[string]any{"analysis_error": analysisErr.Error()}); err != nil {
				return err
			}
			return tx.Audits.Create(audit(actor, "case.analysis_failed", "LocalizationCase", item.ID, &item.RouteID, "{}", snapshot(map[string]any{"error": analysisErr.Error()})))
		})
		return item, &AppError{Code: CodeAlgorithmInput, Status: http.StatusUnprocessableEntity, Message: "case analysis could not be completed", Err: analysisErr}
	}
	encoded, _ := json.Marshal(differences)
	params, _ := json.Marshal(dto.CaseParameters{DistanceToleranceM: tolerance, LossIncreaseDB: loss})
	distance, uncertainty, found := algorithm.PrimaryDifference(differences)
	updates := map[string]any{"differences_json": datatypes.JSON(encoded), "parameters_json": datatypes.JSON(params), "analysis_error": ""}
	if found {
		updates["estimated_distance_m"] = distance
		updates["uncertainty_m"] = uncertainty
	}
	err = s.store.Transaction(func(tx *repository.Store) error {
		if err := tx.Cases.SaveAnalysis(item.ID, constants.CasePendingReview, updates); err != nil {
			return err
		}
		return tx.Audits.Create(audit(actor, "case.analysis_completed", "LocalizationCase", item.ID, &item.RouteID, "{}", snapshot(map[string]any{"differences": len(differences), "parameters": json.RawMessage(params)})))
	})
	if err != nil {
		return item, conflict("case changed while analysis was saved", err)
	}
	return s.store.Cases.Get(id)
}

func (s *CaseService) compareEvents(item model.LocalizationCase, tolerance, loss float64) ([]algorithm.Difference, error) {
	baseline, err := s.store.Events.ForTrace(item.BaselineTraceID)
	if err != nil {
		return nil, fmt.Errorf("load baseline events: %w", err)
	}
	current, err := s.store.Events.ForTrace(item.CurrentTraceID)
	if err != nil {
		return nil, fmt.Errorf("load current events: %w", err)
	}
	if len(baseline) == 0 || len(current) == 0 {
		return nil, fmt.Errorf("both traces require detected events")
	}
	convert := func(events []model.EventMarker) []algorithm.ComparableEvent {
		out := make([]algorithm.ComparableEvent, 0, len(events))
		for _, event := range events {
			out = append(out, algorithm.ComparableEvent{ID: event.ID, DistanceM: event.DistanceM, InsertionLossDB: event.InsertionLossDB, Confidence: event.Confidence})
		}
		return out
	}
	return algorithm.CompareBaseline(convert(baseline), convert(current), tolerance, loss)
}

func (s *CaseService) Confirm(id uint, request dto.ConfirmCaseRequest, actor Actor) (model.LocalizationCase, error) {
	if actor.Role != constants.RoleReviewer && actor.Role != constants.RoleAdmin {
		return model.LocalizationCase{}, &AppError{Code: CodeForbidden, Status: http.StatusForbidden, Message: "reviewer role is required to confirm a case"}
	}
	item, err := s.store.Cases.Get(id)
	if errors.Is(err, repository.ErrNotFound) {
		return item, notFound("case")
	}
	if err != nil {
		return item, internal("get case failed", err)
	}
	if item.CaseStatus == constants.CaseClosed {
		return item, conflict("closed cases cannot be modified", nil)
	}
	before := snapshot(item)
	err = s.store.Transaction(func(tx *repository.Store) error {
		if err := tx.Cases.Transition(id, request.Version, constants.CasePendingReview, constants.CaseConfirmed, map[string]any{"conclusion": request.Conclusion, "estimated_distance_m": request.EstimatedDistanceM, "uncertainty_m": request.UncertaintyM, "reviewer_id": actor.ID}); err != nil {
			return err
		}
		return tx.Audits.Create(audit(actor, "case.confirmed", "LocalizationCase", id, &item.RouteID, before, snapshot(request)))
	})
	if err != nil {
		return item, conflict("case is not pending review or its version changed", err)
	}
	return s.store.Cases.Get(id)
}

func (s *CaseService) Close(id uint, request dto.CloseCaseRequest, actor Actor) (model.LocalizationCase, error) {
	item, err := s.store.Cases.Get(id)
	if errors.Is(err, repository.ErrNotFound) {
		return item, notFound("case")
	}
	if err != nil {
		return item, internal("get case failed", err)
	}
	now := time.Now()
	err = s.store.Transaction(func(tx *repository.Store) error {
		if err := tx.Cases.Transition(id, request.Version, constants.CaseConfirmed, constants.CaseClosed, map[string]any{"closed_at": now}); err != nil {
			return err
		}
		return tx.Audits.Create(audit(actor, "case.closed", "LocalizationCase", id, &item.RouteID, snapshot(item), snapshot(map[string]any{"status": constants.CaseClosed, "closed_at": now})))
	})
	if err != nil {
		return item, conflict("only a confirmed case with the current version can be closed", err)
	}
	return s.store.Cases.Get(id)
}

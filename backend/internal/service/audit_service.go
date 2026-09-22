package service

import (
	"context"
	"fiber-otdr-fault-localization/backend/internal/dto"
	"fiber-otdr-fault-localization/backend/internal/model"
	"fiber-otdr-fault-localization/backend/internal/repository"
)

type AuditService struct{ store *repository.Store }

func NewAuditService(store *repository.Store) *AuditService { return &AuditService{store} }

func (s *AuditService) List(query dto.AuditQuery) ([]model.AuditLog, dto.Pagination, error) {
	normalizePage(&query.Page, &query.PageSize)
	entries, total, err := s.store.Audits.List(query)
	if err != nil {
		return nil, dto.Pagination{}, internal("list audit entries failed", err)
	}
	return entries, dto.Pagination{Page: query.Page, PageSize: query.PageSize, Total: total}, nil
}

func (s *AuditService) Ready(ctx context.Context) error {
	return s.store.Ping(ctx)
}

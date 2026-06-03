package service

import (
	"context"
	"fmt"
	"test-stake-backend/internal/models"
	"test-stake-backend/internal/repository"
)

// StakedEventListResult 是 StakedEvent 分页查询结果。
type StakedEventListResult struct {
	Items    []models.StakedEvent `json:"items"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
}

// StakedEventService 处理 StakedEvent 的业务编排。
type StakedEventService struct {
	repo *repository.StakedEventRepository
}

// NewStakedEventService 创建 StakedEventService。
func NewStakedEventService(repo *repository.StakedEventRepository) (*StakedEventService, error) {
	if repo == nil {
		return nil, fmt.Errorf("create staked event service: repository is nil")
	}

	return &StakedEventService{repo: repo}, nil
}

// Create 创建 StakedEvent。
func (s *StakedEventService) Create(ctx context.Context, event *models.StakedEvent) (*models.StakedEvent, error) {
	if err := s.repo.Create(ctx, event); err != nil {
		return nil, err
	}

	return event, nil
}

// GetByID 按 ID 查询 StakedEvent。
func (s *StakedEventService) GetByID(ctx context.Context, id int64) (*models.StakedEvent, error) {
	return s.repo.GetByID(ctx, id)
}

// List 按条件分页查询 StakedEvent。
func (s *StakedEventService) List(ctx context.Context, query repository.StakedEventQuery) (*StakedEventListResult, error) {
	events, total, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, err
	}

	page, pageSize := normalizePagination(query.Page, query.PageSize)
	return &StakedEventListResult{
		Items:    events,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func normalizePagination(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	return page, pageSize
}

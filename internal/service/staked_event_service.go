package service

import (
	"context"
	"fmt"
	"test-stake-backend/internal/models"
	"test-stake-backend/internal/repository"
)

type StakedEventListResult struct {
	Items    []models.StakedEvent `json:"items"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
}

type StakedEventService struct {
	repo *repository.StakedEventRepository
}

func NewStakedEventService(repo *repository.StakedEventRepository) (*StakedEventService, error) {
	if repo == nil {
		return nil, fmt.Errorf("create staked event service: repository is nil")
	}

	return &StakedEventService{repo: repo}, nil
}

func (s *StakedEventService) GetByID(ctx context.Context, id int64) (*models.StakedEvent, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *StakedEventService) List(ctx context.Context, query repository.StakedEventQuery) (*StakedEventListResult, error) {
	events, total, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, err
	}

	return &StakedEventListResult{
		Items:    events,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

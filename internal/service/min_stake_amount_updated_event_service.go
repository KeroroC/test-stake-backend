package service

import (
	"context"
	"fmt"
	"test-stake-backend/internal/models"
	"test-stake-backend/internal/repository"
)

type MinStakeAmountUpdatedListResult struct {
	Items    []models.MinStakeAmountUpdatedEvent `json:"items"`
	Total    int64                               `json:"total"`
	Page     int                                 `json:"page"`
	PageSize int                                 `json:"page_size"`
}

type MinStakeAmountUpdatedEventService struct {
	repo *repository.MinStakeAmountUpdatedEventRepository
}

func NewMinStakeAmountUpdatedEventService(repo *repository.MinStakeAmountUpdatedEventRepository) (*MinStakeAmountUpdatedEventService, error) {
	if repo == nil {
		return nil, fmt.Errorf("create min stake amount updated event service: repository is nil")
	}

	return &MinStakeAmountUpdatedEventService{repo: repo}, nil
}

func (s *MinStakeAmountUpdatedEventService) GetByID(ctx context.Context, id int64) (*models.MinStakeAmountUpdatedEvent, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *MinStakeAmountUpdatedEventService) List(ctx context.Context, query repository.MinStakeAmountUpdatedEventQuery) (*MinStakeAmountUpdatedListResult, error) {
	events, total, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, err
	}

	return &MinStakeAmountUpdatedListResult{
		Items:    events,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

package service

import (
	"context"
	"fmt"
	"test-stake-backend/internal/models"
	"test-stake-backend/internal/repository"
)

type RewardRateUpdatedListResult struct {
	Items    []models.RewardRateUpdatedEvent `json:"items"`
	Total    int64                           `json:"total"`
	Page     int                             `json:"page"`
	PageSize int                             `json:"page_size"`
}

type RewardRateUpdatedEventService struct {
	repo *repository.RewardRateUpdatedEventRepository
}

func NewRewardRateUpdatedEventService(repo *repository.RewardRateUpdatedEventRepository) (*RewardRateUpdatedEventService, error) {
	if repo == nil {
		return nil, fmt.Errorf("create reward rate updated event service: repository is nil")
	}

	return &RewardRateUpdatedEventService{repo: repo}, nil
}

func (s *RewardRateUpdatedEventService) GetByID(ctx context.Context, id int64) (*models.RewardRateUpdatedEvent, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *RewardRateUpdatedEventService) List(ctx context.Context, query repository.RewardRateUpdatedEventQuery) (*RewardRateUpdatedListResult, error) {
	events, total, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, err
	}

	return &RewardRateUpdatedListResult{
		Items:    events,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

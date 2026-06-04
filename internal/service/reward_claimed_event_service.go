package service

import (
	"context"
	"fmt"
	"test-stake-backend/internal/models"
	"test-stake-backend/internal/repository"
)

type RewardClaimedListResult struct {
	Items    []models.RewardClaimedEvent `json:"items"`
	Total    int64                       `json:"total"`
	Page     int                         `json:"page"`
	PageSize int                         `json:"page_size"`
}

type RewardClaimedEventService struct {
	repo *repository.RewardClaimedEventRepository
}

func NewRewardClaimedEventService(repo *repository.RewardClaimedEventRepository) (*RewardClaimedEventService, error) {
	if repo == nil {
		return nil, fmt.Errorf("create reward claimed event service: repository is nil")
	}

	return &RewardClaimedEventService{repo: repo}, nil
}

func (s *RewardClaimedEventService) GetByID(ctx context.Context, id int64) (*models.RewardClaimedEvent, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *RewardClaimedEventService) List(ctx context.Context, query repository.RewardClaimedEventQuery) (*RewardClaimedListResult, error) {
	events, total, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, err
	}

	return &RewardClaimedListResult{
		Items:    events,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

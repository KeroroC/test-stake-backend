package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"test-stake-backend/internal/cache"
	"test-stake-backend/internal/models"
	"test-stake-backend/internal/repository"

	"github.com/redis/go-redis/v9"
)

type RewardRateUpdatedListResult struct {
	Items    []models.RewardRateUpdatedEvent `json:"items"`
	Total    int64                           `json:"total"`
	Page     int                             `json:"page"`
	PageSize int                             `json:"page_size"`
}

type RewardRateUpdatedEventService struct {
	repo     *repository.RewardRateUpdatedEventRepository
	rdb      *redis.Client
	cacheTTL time.Duration
}

func NewRewardRateUpdatedEventService(repo *repository.RewardRateUpdatedEventRepository, rdb *redis.Client, cacheTTL time.Duration) (*RewardRateUpdatedEventService, error) {
	if repo == nil {
		return nil, fmt.Errorf("create reward rate updated event service: repository is nil")
	}

	return &RewardRateUpdatedEventService{repo: repo, rdb: rdb, cacheTTL: cacheTTL}, nil
}

func (s *RewardRateUpdatedEventService) GetByID(ctx context.Context, id int64) (*models.RewardRateUpdatedEvent, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *RewardRateUpdatedEventService) List(ctx context.Context, query repository.RewardRateUpdatedEventQuery) (*RewardRateUpdatedListResult, error) {
	key := cache.BuildListKey("reward-rate-updated", query)

	if result, ok, err := cache.Get[RewardRateUpdatedListResult](ctx, s.rdb, key); err == nil && ok {
		return &result, nil
	} else if err != nil {
		log.Printf("cache get reward-rate-updated list: %v", err)
	}

	events, total, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, err
	}

	result := &RewardRateUpdatedListResult{
		Items:    events,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}

	if err := cache.Set(ctx, s.rdb, key, result, s.cacheTTL); err != nil {
		log.Printf("cache set reward-rate-updated list: %v", err)
	}

	return result, nil
}

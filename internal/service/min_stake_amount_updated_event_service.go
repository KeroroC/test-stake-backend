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

type MinStakeAmountUpdatedListResult struct {
	Items    []models.MinStakeAmountUpdatedEvent `json:"items"`
	Total    int64                               `json:"total"`
	Page     int                                 `json:"page"`
	PageSize int                                 `json:"page_size"`
}

type MinStakeAmountUpdatedEventService struct {
	repo     *repository.MinStakeAmountUpdatedEventRepository
	rdb      *redis.Client
	cacheTTL time.Duration
}

func NewMinStakeAmountUpdatedEventService(repo *repository.MinStakeAmountUpdatedEventRepository, rdb *redis.Client, cacheTTL time.Duration) (*MinStakeAmountUpdatedEventService, error) {
	if repo == nil {
		return nil, fmt.Errorf("create min stake amount updated event service: repository is nil")
	}

	return &MinStakeAmountUpdatedEventService{repo: repo, rdb: rdb, cacheTTL: cacheTTL}, nil
}

func (s *MinStakeAmountUpdatedEventService) GetByID(ctx context.Context, id int64) (*models.MinStakeAmountUpdatedEvent, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *MinStakeAmountUpdatedEventService) List(ctx context.Context, query repository.MinStakeAmountUpdatedEventQuery) (*MinStakeAmountUpdatedListResult, error) {
	key := cache.BuildListKey("min-stake-amount-updated", query)

	if result, ok, err := cache.Get[MinStakeAmountUpdatedListResult](ctx, s.rdb, key); err == nil && ok {
		return &result, nil
	} else if err != nil {
		log.Printf("cache get min-stake-amount-updated list: %v", err)
	}

	events, total, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, err
	}

	result := &MinStakeAmountUpdatedListResult{
		Items:    events,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}

	if err := cache.Set(ctx, s.rdb, key, result, s.cacheTTL); err != nil {
		log.Printf("cache set min-stake-amount-updated list: %v", err)
	}

	return result, nil
}

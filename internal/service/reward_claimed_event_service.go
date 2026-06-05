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

type RewardClaimedListResult struct {
	Items    []models.RewardClaimedEvent `json:"items"`
	Total    int64                       `json:"total"`
	Page     int                         `json:"page"`
	PageSize int                         `json:"page_size"`
}

type RewardClaimedEventService struct {
	repo     *repository.RewardClaimedEventRepository
	rdb      *redis.Client
	cacheTTL time.Duration
}

func NewRewardClaimedEventService(repo *repository.RewardClaimedEventRepository, rdb *redis.Client, cacheTTL time.Duration) (*RewardClaimedEventService, error) {
	if repo == nil {
		return nil, fmt.Errorf("create reward claimed event service: repository is nil")
	}

	return &RewardClaimedEventService{repo: repo, rdb: rdb, cacheTTL: cacheTTL}, nil
}

func (s *RewardClaimedEventService) GetByID(ctx context.Context, id int64) (*models.RewardClaimedEvent, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *RewardClaimedEventService) List(ctx context.Context, query repository.RewardClaimedEventQuery) (*RewardClaimedListResult, error) {
	key := cache.BuildListKey("reward-claimed", query)

	if result, ok, err := cache.Get[RewardClaimedListResult](ctx, s.rdb, key); err == nil && ok {
		return &result, nil
	} else if err != nil {
		log.Printf("cache get reward-claimed list: %v", err)
	}

	events, total, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, err
	}

	result := &RewardClaimedListResult{
		Items:    events,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}

	if err := cache.Set(ctx, s.rdb, key, result, s.cacheTTL); err != nil {
		log.Printf("cache set reward-claimed list: %v", err)
	}

	return result, nil
}

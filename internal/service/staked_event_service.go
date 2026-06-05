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

type StakedEventListResult struct {
	Items    []models.StakedEvent `json:"items"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
}

type StakedEventService struct {
	repo     *repository.StakedEventRepository
	rdb      *redis.Client
	cacheTTL time.Duration
}

func NewStakedEventService(repo *repository.StakedEventRepository, rdb *redis.Client, cacheTTL time.Duration) (*StakedEventService, error) {
	if repo == nil {
		return nil, fmt.Errorf("create staked event service: repository is nil")
	}

	return &StakedEventService{repo: repo, rdb: rdb, cacheTTL: cacheTTL}, nil
}

func (s *StakedEventService) GetByID(ctx context.Context, id int64) (*models.StakedEvent, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *StakedEventService) List(ctx context.Context, query repository.StakedEventQuery) (*StakedEventListResult, error) {
	key := cache.BuildListKey("staked", query)

	// 尝试读缓存
	if result, ok, err := cache.Get[StakedEventListResult](ctx, s.rdb, key); err == nil && ok {
		return &result, nil
	} else if err != nil {
		log.Printf("cache get staked list: %v", err)
	}

	// 查数据库
	events, total, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, err
	}

	result := &StakedEventListResult{
		Items:    events,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}

	// 写缓存
	if err := cache.Set(ctx, s.rdb, key, result, s.cacheTTL); err != nil {
		log.Printf("cache set staked list: %v", err)
	}

	return result, nil
}

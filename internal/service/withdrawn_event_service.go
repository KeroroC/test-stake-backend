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

type WithdrawnListResult struct {
	Items    []models.WithdrawnEvent `json:"items"`
	Total    int64                   `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"page_size"`
}

type WithdrawnEventService struct {
	repo     *repository.WithdrawnEventRepository
	rdb      *redis.Client
	cacheTTL time.Duration
}

func NewWithdrawnEventService(repo *repository.WithdrawnEventRepository, rdb *redis.Client, cacheTTL time.Duration) (*WithdrawnEventService, error) {
	if repo == nil {
		return nil, fmt.Errorf("create withdrawn event service: repository is nil")
	}

	return &WithdrawnEventService{repo: repo, rdb: rdb, cacheTTL: cacheTTL}, nil
}

func (s *WithdrawnEventService) GetByID(ctx context.Context, id int64) (*models.WithdrawnEvent, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *WithdrawnEventService) List(ctx context.Context, query repository.WithdrawnEventQuery) (*WithdrawnListResult, error) {
	key := cache.BuildListKey("withdrawn", query)

	if result, ok, err := cache.Get[WithdrawnListResult](ctx, s.rdb, key); err == nil && ok {
		return &result, nil
	} else if err != nil {
		log.Printf("cache get withdrawn list: %v", err)
	}

	events, total, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, err
	}

	result := &WithdrawnListResult{
		Items:    events,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}

	if err := cache.Set(ctx, s.rdb, key, result, s.cacheTTL); err != nil {
		log.Printf("cache set withdrawn list: %v", err)
	}

	return result, nil
}

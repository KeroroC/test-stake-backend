package service

import (
	"context"
	"fmt"
	"test-stake-backend/internal/models"
	"test-stake-backend/internal/repository"
)

type WithdrawnListResult struct {
	Items    []models.WithdrawnEvent `json:"items"`
	Total    int64                   `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"page_size"`
}

type WithdrawnEventService struct {
	repo *repository.WithdrawnEventRepository
}

func NewWithdrawnEventService(repo *repository.WithdrawnEventRepository) (*WithdrawnEventService, error) {
	if repo == nil {
		return nil, fmt.Errorf("create withdrawn event service: repository is nil")
	}

	return &WithdrawnEventService{repo: repo}, nil
}

func (s *WithdrawnEventService) GetByID(ctx context.Context, id int64) (*models.WithdrawnEvent, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *WithdrawnEventService) List(ctx context.Context, query repository.WithdrawnEventQuery) (*WithdrawnListResult, error) {
	events, total, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, err
	}

	return &WithdrawnListResult{
		Items:    events,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

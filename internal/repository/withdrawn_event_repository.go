package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"test-stake-backend/internal/models"

	"gorm.io/gorm"
)

var (
	ErrInvalidWithdrawnEvent  = errors.New("invalid withdrawn event")
	ErrWithdrawnEventNotFound = errors.New("withdrawn event not found")
)

type WithdrawnEventRepository struct {
	db *gorm.DB
}

func NewWithdrawnEventRepository(db *gorm.DB) (*WithdrawnEventRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("create withdrawn event repository: db is nil")
	}

	return &WithdrawnEventRepository{db: db}, nil
}

type WithdrawnEventQuery struct {
	BaseQuery
	User string
}

func (r *WithdrawnEventRepository) GetByID(ctx context.Context, id int64) (*models.WithdrawnEvent, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be greater than 0", ErrInvalidWithdrawnEvent)
	}

	var event models.WithdrawnEvent
	if err := r.db.WithContext(ctx).First(&event, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: id=%d", ErrWithdrawnEventNotFound, id)
		}
		return nil, fmt.Errorf("get withdrawn event by id %d: %w", id, err)
	}

	return &event, nil
}

func (r *WithdrawnEventRepository) List(ctx context.Context, query WithdrawnEventQuery) ([]models.WithdrawnEvent, int64, error) {
	if err := validateWithdrawnEventQuery(query); err != nil {
		return nil, 0, err
	}

	page, pageSize := normalizePagination(query.Page, query.PageSize)
	db := r.applyQuery(r.db.WithContext(ctx).Model(&models.WithdrawnEvent{}), query)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count withdrawn events: %w", err)
	}

	var events []models.WithdrawnEvent
	err := db.Order("block_number DESC, log_index DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&events).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list withdrawn events: %w", err)
	}

	return events, total, nil
}

func (r *WithdrawnEventRepository) Create(ctx context.Context, event *models.WithdrawnEvent) error {
	if event == nil {
		return fmt.Errorf("%w: event is nil", ErrInvalidWithdrawnEvent)
	}
	if err := validateWithdrawnEvent(*event); err != nil {
		return err
	}
	normalizeStrings(&event.ContractAddress, &event.User, &event.TxHash, &event.BlockHash)

	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		return fmt.Errorf("create withdrawn event tx_hash=%s log_index=%d: %w", event.TxHash, event.LogIndex, err)
	}

	return nil
}

func (r *WithdrawnEventRepository) applyQuery(db *gorm.DB, query WithdrawnEventQuery) *gorm.DB {
	db = applyBaseQuery(db, query.BaseQuery)
	if query.User != "" {
		db = db.Where("user = ?", strings.ToLower(query.User))
	}

	return db
}

func validateWithdrawnEvent(event models.WithdrawnEvent) error {
	s := ErrInvalidWithdrawnEvent
	if err := validateAddress(s, "contract_address", event.ContractAddress); err != nil {
		return err
	}
	if err := validateAddress(s, "user", event.User); err != nil {
		return err
	}
	if err := validateUint256Amount(s, event.Amount); err != nil {
		return err
	}
	if err := validateHash(s, "tx_hash", event.TxHash); err != nil {
		return err
	}
	if err := validateHash(s, "block_hash", event.BlockHash); err != nil {
		return err
	}

	return nil
}

func validateWithdrawnEventQuery(query WithdrawnEventQuery) error {
	if err := validateBaseQuery(ErrInvalidWithdrawnEvent, query.BaseQuery); err != nil {
		return err
	}
	if query.User != "" {
		if err := validateAddress(ErrInvalidWithdrawnEvent, "user", query.User); err != nil {
			return err
		}
	}

	return nil
}

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
	ErrInvalidStakedEvent  = errors.New("invalid staked event")
	ErrStakedEventNotFound = errors.New("staked event not found")
)

type StakedEventRepository struct {
	db *gorm.DB
}

func NewStakedEventRepository(db *gorm.DB) (*StakedEventRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("create staked event repository: db is nil")
	}

	return &StakedEventRepository{db: db}, nil
}

type StakedEventQuery struct {
	BaseQuery
	User string
}

func (r *StakedEventRepository) GetByID(ctx context.Context, id int64) (*models.StakedEvent, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be greater than 0", ErrInvalidStakedEvent)
	}

	var event models.StakedEvent
	if err := r.db.WithContext(ctx).First(&event, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: id=%d", ErrStakedEventNotFound, id)
		}
		return nil, fmt.Errorf("get staked event by id %d: %w", id, err)
	}

	return &event, nil
}

func (r *StakedEventRepository) List(ctx context.Context, query StakedEventQuery) ([]models.StakedEvent, int64, error) {
	if err := validateStakedEventQuery(query); err != nil {
		return nil, 0, err
	}

	page, pageSize := normalizePagination(query.Page, query.PageSize)
	db := r.applyQuery(r.db.WithContext(ctx).Model(&models.StakedEvent{}), query)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count staked events: %w", err)
	}

	var events []models.StakedEvent
	err := db.Order("block_number DESC, log_index DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&events).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list staked events: %w", err)
	}

	return events, total, nil
}

func (r *StakedEventRepository) Create(ctx context.Context, event *models.StakedEvent) error {
	if event == nil {
		return fmt.Errorf("%w: event is nil", ErrInvalidStakedEvent)
	}
	if err := validateStakedEvent(*event); err != nil {
		return err
	}
	normalizeStrings(&event.ContractAddress, &event.User, &event.TxHash, &event.BlockHash)

	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		return fmt.Errorf("create staked event tx_hash=%s log_index=%d: %w", event.TxHash, event.LogIndex, err)
	}

	return nil
}

func (r *StakedEventRepository) applyQuery(db *gorm.DB, query StakedEventQuery) *gorm.DB {
	db = applyBaseQuery(db, query.BaseQuery)
	if query.User != "" {
		db = db.Where("user = ?", strings.ToLower(query.User))
	}

	return db
}

func validateStakedEvent(event models.StakedEvent) error {
	s := ErrInvalidStakedEvent
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

func validateStakedEventQuery(query StakedEventQuery) error {
	if err := validateBaseQuery(ErrInvalidStakedEvent, query.BaseQuery); err != nil {
		return err
	}
	if query.User != "" {
		if err := validateAddress(ErrInvalidStakedEvent, "user", query.User); err != nil {
			return err
		}
	}

	return nil
}

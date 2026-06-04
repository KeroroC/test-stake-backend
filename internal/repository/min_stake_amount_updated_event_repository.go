package repository

import (
	"context"
	"errors"
	"fmt"
	"test-stake-backend/internal/models"

	"gorm.io/gorm"
)

var (
	ErrInvalidMinStakeAmountUpdatedEvent  = errors.New("invalid min stake amount updated event")
	ErrMinStakeAmountUpdatedEventNotFound = errors.New("min stake amount updated event not found")
)

type MinStakeAmountUpdatedEventRepository struct {
	db *gorm.DB
}

func NewMinStakeAmountUpdatedEventRepository(db *gorm.DB) (*MinStakeAmountUpdatedEventRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("create min stake amount updated event repository: db is nil")
	}

	return &MinStakeAmountUpdatedEventRepository{db: db}, nil
}

type MinStakeAmountUpdatedEventQuery struct {
	BaseQuery
}

func (r *MinStakeAmountUpdatedEventRepository) GetByID(ctx context.Context, id int64) (*models.MinStakeAmountUpdatedEvent, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be greater than 0", ErrInvalidMinStakeAmountUpdatedEvent)
	}

	var event models.MinStakeAmountUpdatedEvent
	if err := r.db.WithContext(ctx).First(&event, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: id=%d", ErrMinStakeAmountUpdatedEventNotFound, id)
		}
		return nil, fmt.Errorf("get min stake amount updated event by id %d: %w", id, err)
	}

	return &event, nil
}

func (r *MinStakeAmountUpdatedEventRepository) List(ctx context.Context, query MinStakeAmountUpdatedEventQuery) ([]models.MinStakeAmountUpdatedEvent, int64, error) {
	if err := validateMinStakeAmountUpdatedEventQuery(query); err != nil {
		return nil, 0, err
	}

	page, pageSize := normalizePagination(query.Page, query.PageSize)
	db := applyBaseQuery(r.db.WithContext(ctx).Model(&models.MinStakeAmountUpdatedEvent{}), query.BaseQuery)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count min stake amount updated events: %w", err)
	}

	var events []models.MinStakeAmountUpdatedEvent
	err := db.Order("block_number DESC, log_index DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&events).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list min stake amount updated events: %w", err)
	}

	return events, total, nil
}

func (r *MinStakeAmountUpdatedEventRepository) Create(ctx context.Context, event *models.MinStakeAmountUpdatedEvent) error {
	if event == nil {
		return fmt.Errorf("%w: event is nil", ErrInvalidMinStakeAmountUpdatedEvent)
	}
	if err := validateMinStakeAmountUpdatedEvent(*event); err != nil {
		return err
	}
	normalizeStrings(&event.ContractAddress, &event.TxHash, &event.BlockHash)

	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		return fmt.Errorf("create min stake amount updated event tx_hash=%s log_index=%d: %w", event.TxHash, event.LogIndex, err)
	}

	return nil
}

func validateMinStakeAmountUpdatedEvent(event models.MinStakeAmountUpdatedEvent) error {
	s := ErrInvalidMinStakeAmountUpdatedEvent
	if err := validateAddress(s, "contract_address", event.ContractAddress); err != nil {
		return err
	}
	if err := validateUint256Amount(s, event.OldAmount); err != nil {
		return err
	}
	if err := validateUint256Amount(s, event.NewAmount); err != nil {
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

func validateMinStakeAmountUpdatedEventQuery(query MinStakeAmountUpdatedEventQuery) error {
	return validateBaseQuery(ErrInvalidMinStakeAmountUpdatedEvent, query.BaseQuery)
}

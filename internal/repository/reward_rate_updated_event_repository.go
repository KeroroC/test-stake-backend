package repository

import (
	"context"
	"errors"
	"fmt"
	"test-stake-backend/internal/models"

	"gorm.io/gorm"
)

var (
	ErrInvalidRewardRateUpdatedEvent  = errors.New("invalid reward rate updated event")
	ErrRewardRateUpdatedEventNotFound = errors.New("reward rate updated event not found")
)

type RewardRateUpdatedEventRepository struct {
	db *gorm.DB
}

func NewRewardRateUpdatedEventRepository(db *gorm.DB) (*RewardRateUpdatedEventRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("create reward rate updated event repository: db is nil")
	}

	return &RewardRateUpdatedEventRepository{db: db}, nil
}

type RewardRateUpdatedEventQuery struct {
	BaseQuery
}

func (r *RewardRateUpdatedEventRepository) GetByID(ctx context.Context, id int64) (*models.RewardRateUpdatedEvent, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be greater than 0", ErrInvalidRewardRateUpdatedEvent)
	}

	var event models.RewardRateUpdatedEvent
	if err := r.db.WithContext(ctx).First(&event, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: id=%d", ErrRewardRateUpdatedEventNotFound, id)
		}
		return nil, fmt.Errorf("get reward rate updated event by id %d: %w", id, err)
	}

	return &event, nil
}

func (r *RewardRateUpdatedEventRepository) List(ctx context.Context, query RewardRateUpdatedEventQuery) ([]models.RewardRateUpdatedEvent, int64, error) {
	if err := validateRewardRateUpdatedEventQuery(query); err != nil {
		return nil, 0, err
	}

	page, pageSize := normalizePagination(query.Page, query.PageSize)
	db := applyBaseQuery(r.db.WithContext(ctx).Model(&models.RewardRateUpdatedEvent{}), query.BaseQuery)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count reward rate updated events: %w", err)
	}

	var events []models.RewardRateUpdatedEvent
	err := db.Order("block_number DESC, log_index DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&events).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list reward rate updated events: %w", err)
	}

	return events, total, nil
}

func (r *RewardRateUpdatedEventRepository) Create(ctx context.Context, event *models.RewardRateUpdatedEvent) error {
	if event == nil {
		return fmt.Errorf("%w: event is nil", ErrInvalidRewardRateUpdatedEvent)
	}
	if err := validateRewardRateUpdatedEvent(*event); err != nil {
		return err
	}
	normalizeStrings(&event.ContractAddress, &event.TxHash, &event.BlockHash)

	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		if isDuplicateKeyError(err) {
			return nil
		}
		return fmt.Errorf("create reward rate updated event tx_hash=%s log_index=%d: %w", event.TxHash, event.LogIndex, err)
	}

	return nil
}

func validateRewardRateUpdatedEvent(event models.RewardRateUpdatedEvent) error {
	s := ErrInvalidRewardRateUpdatedEvent
	if err := validateAddress(s, "contract_address", event.ContractAddress); err != nil {
		return err
	}
	if err := validateUint256Amount(s, event.OldRate); err != nil {
		return err
	}
	if err := validateUint256Amount(s, event.NewRate); err != nil {
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

func validateRewardRateUpdatedEventQuery(query RewardRateUpdatedEventQuery) error {
	return validateBaseQuery(ErrInvalidRewardRateUpdatedEvent, query.BaseQuery)
}

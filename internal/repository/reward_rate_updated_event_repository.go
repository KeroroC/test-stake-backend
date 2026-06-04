package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"test-stake-backend/internal/models"

	"gorm.io/gorm"
)

const (
	defaultRewardRateUpdatedEventPage     = 1
	defaultRewardRateUpdatedEventPageSize = 20
	maxRewardRateUpdatedEventPageSize     = 100
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
	ID              int64
	ContractAddress string
	TxHash          string
	BlockNumberFrom *uint64
	BlockNumberTo   *uint64
	Page            int
	PageSize        int
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

	page, pageSize := normalizeRewardRateUpdatedEventPagination(query.Page, query.PageSize)
	db := r.applyQuery(r.db.WithContext(ctx).Model(&models.RewardRateUpdatedEvent{}), query)

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
	normalizeRewardRateUpdatedEvent(event)

	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		return fmt.Errorf("create reward rate updated event tx_hash=%s log_index=%d: %w", event.TxHash, event.LogIndex, err)
	}

	return nil
}

func (r *RewardRateUpdatedEventRepository) applyQuery(db *gorm.DB, query RewardRateUpdatedEventQuery) *gorm.DB {
	if query.ID > 0 {
		db = db.Where("id = ?", query.ID)
	}
	if query.ContractAddress != "" {
		db = db.Where("contract_address = ?", strings.ToLower(query.ContractAddress))
	}
	if query.TxHash != "" {
		db = db.Where("tx_hash = ?", strings.ToLower(query.TxHash))
	}
	if query.BlockNumberFrom != nil {
		db = db.Where("block_number >= ?", *query.BlockNumberFrom)
	}
	if query.BlockNumberTo != nil {
		db = db.Where("block_number <= ?", *query.BlockNumberTo)
	}

	return db
}

func normalizeRewardRateUpdatedEventPagination(page, pageSize int) (int, int) {
	if page <= 0 {
		page = defaultRewardRateUpdatedEventPage
	}
	if pageSize <= 0 {
		pageSize = defaultRewardRateUpdatedEventPageSize
	}
	if pageSize > maxRewardRateUpdatedEventPageSize {
		pageSize = maxRewardRateUpdatedEventPageSize
	}

	return page, pageSize
}

func validateRewardRateUpdatedEvent(event models.RewardRateUpdatedEvent) error {
	if err := validateAddress("contract_address", event.ContractAddress); err != nil {
		return err
	}
	if err := validateUint256Amount(event.OldRate); err != nil {
		return err
	}
	if err := validateUint256Amount(event.NewRate); err != nil {
		return err
	}
	if err := validateHash("tx_hash", event.TxHash); err != nil {
		return err
	}
	if err := validateHash("block_hash", event.BlockHash); err != nil {
		return err
	}

	return nil
}

func validateRewardRateUpdatedEventQuery(query RewardRateUpdatedEventQuery) error {
	if query.ID < 0 {
		return fmt.Errorf("%w: id must not be negative", ErrInvalidRewardRateUpdatedEvent)
	}
	if query.ContractAddress != "" {
		if err := validateAddress("contract_address", query.ContractAddress); err != nil {
			return err
		}
	}
	if query.TxHash != "" {
		if err := validateHash("tx_hash", query.TxHash); err != nil {
			return err
		}
	}
	if query.BlockNumberFrom != nil && query.BlockNumberTo != nil && *query.BlockNumberFrom > *query.BlockNumberTo {
		return fmt.Errorf("%w: block_number_from must not be greater than block_number_to", ErrInvalidRewardRateUpdatedEvent)
	}

	return nil
}

func normalizeRewardRateUpdatedEvent(event *models.RewardRateUpdatedEvent) {
	event.ContractAddress = strings.ToLower(event.ContractAddress)
	event.TxHash = strings.ToLower(event.TxHash)
	event.BlockHash = strings.ToLower(event.BlockHash)
}

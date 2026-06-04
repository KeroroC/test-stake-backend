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
	defaultMinStakeAmountUpdatedEventPage     = 1
	defaultMinStakeAmountUpdatedEventPageSize = 20
	maxMinStakeAmountUpdatedEventPageSize     = 100
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
	ID              int64
	ContractAddress string
	TxHash          string
	BlockNumberFrom *uint64
	BlockNumberTo   *uint64
	Page            int
	PageSize        int
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

	page, pageSize := normalizeMinStakeAmountUpdatedEventPagination(query.Page, query.PageSize)
	db := r.applyQuery(r.db.WithContext(ctx).Model(&models.MinStakeAmountUpdatedEvent{}), query)

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
	normalizeMinStakeAmountUpdatedEvent(event)

	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		return fmt.Errorf("create min stake amount updated event tx_hash=%s log_index=%d: %w", event.TxHash, event.LogIndex, err)
	}

	return nil
}

func (r *MinStakeAmountUpdatedEventRepository) applyQuery(db *gorm.DB, query MinStakeAmountUpdatedEventQuery) *gorm.DB {
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

func normalizeMinStakeAmountUpdatedEventPagination(page, pageSize int) (int, int) {
	if page <= 0 {
		page = defaultMinStakeAmountUpdatedEventPage
	}
	if pageSize <= 0 {
		pageSize = defaultMinStakeAmountUpdatedEventPageSize
	}
	if pageSize > maxMinStakeAmountUpdatedEventPageSize {
		pageSize = maxMinStakeAmountUpdatedEventPageSize
	}

	return page, pageSize
}

func validateMinStakeAmountUpdatedEvent(event models.MinStakeAmountUpdatedEvent) error {
	if err := validateAddress("contract_address", event.ContractAddress); err != nil {
		return err
	}
	if err := validateUint256Amount(event.OldAmount); err != nil {
		return err
	}
	if err := validateUint256Amount(event.NewAmount); err != nil {
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

func validateMinStakeAmountUpdatedEventQuery(query MinStakeAmountUpdatedEventQuery) error {
	if query.ID < 0 {
		return fmt.Errorf("%w: id must not be negative", ErrInvalidMinStakeAmountUpdatedEvent)
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
		return fmt.Errorf("%w: block_number_from must not be greater than block_number_to", ErrInvalidMinStakeAmountUpdatedEvent)
	}

	return nil
}

func normalizeMinStakeAmountUpdatedEvent(event *models.MinStakeAmountUpdatedEvent) {
	event.ContractAddress = strings.ToLower(event.ContractAddress)
	event.TxHash = strings.ToLower(event.TxHash)
	event.BlockHash = strings.ToLower(event.BlockHash)
}

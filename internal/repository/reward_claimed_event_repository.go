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
	defaultRewardClaimedEventPage     = 1
	defaultRewardClaimedEventPageSize = 20
	maxRewardClaimedEventPageSize     = 100
)

var (
	ErrInvalidRewardClaimedEvent  = errors.New("invalid reward claimed event")
	ErrRewardClaimedEventNotFound = errors.New("reward claimed event not found")
)

type RewardClaimedEventRepository struct {
	db *gorm.DB
}

func NewRewardClaimedEventRepository(db *gorm.DB) (*RewardClaimedEventRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("create reward claimed event repository: db is nil")
	}

	return &RewardClaimedEventRepository{db: db}, nil
}

type RewardClaimedEventQuery struct {
	ID              int64
	ContractAddress string
	User            string
	TxHash          string
	BlockNumberFrom *uint64
	BlockNumberTo   *uint64
	Page            int
	PageSize        int
}

func (r *RewardClaimedEventRepository) GetByID(ctx context.Context, id int64) (*models.RewardClaimedEvent, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be greater than 0", ErrInvalidRewardClaimedEvent)
	}

	var event models.RewardClaimedEvent
	if err := r.db.WithContext(ctx).First(&event, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: id=%d", ErrRewardClaimedEventNotFound, id)
		}
		return nil, fmt.Errorf("get reward claimed event by id %d: %w", id, err)
	}

	return &event, nil
}

func (r *RewardClaimedEventRepository) List(ctx context.Context, query RewardClaimedEventQuery) ([]models.RewardClaimedEvent, int64, error) {
	if err := validateRewardClaimedEventQuery(query); err != nil {
		return nil, 0, err
	}

	page, pageSize := normalizeRewardClaimedEventPagination(query.Page, query.PageSize)
	db := r.applyQuery(r.db.WithContext(ctx).Model(&models.RewardClaimedEvent{}), query)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count reward claimed events: %w", err)
	}

	var events []models.RewardClaimedEvent
	err := db.Order("block_number DESC, log_index DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&events).Error
	if err != nil {
		return nil, 0, fmt.Errorf("list reward claimed events: %w", err)
	}

	return events, total, nil
}

func (r *RewardClaimedEventRepository) Create(ctx context.Context, event *models.RewardClaimedEvent) error {
	if event == nil {
		return fmt.Errorf("%w: event is nil", ErrInvalidRewardClaimedEvent)
	}
	if err := validateRewardClaimedEvent(*event); err != nil {
		return err
	}
	normalizeRewardClaimedEvent(event)

	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		return fmt.Errorf("create reward claimed event tx_hash=%s log_index=%d: %w", event.TxHash, event.LogIndex, err)
	}

	return nil
}

func (r *RewardClaimedEventRepository) applyQuery(db *gorm.DB, query RewardClaimedEventQuery) *gorm.DB {
	if query.ID > 0 {
		db = db.Where("id = ?", query.ID)
	}
	if query.ContractAddress != "" {
		db = db.Where("contract_address = ?", strings.ToLower(query.ContractAddress))
	}
	if query.User != "" {
		db = db.Where("user = ?", strings.ToLower(query.User))
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

func normalizeRewardClaimedEventPagination(page, pageSize int) (int, int) {
	if page <= 0 {
		page = defaultRewardClaimedEventPage
	}
	if pageSize <= 0 {
		pageSize = defaultRewardClaimedEventPageSize
	}
	if pageSize > maxRewardClaimedEventPageSize {
		pageSize = maxRewardClaimedEventPageSize
	}

	return page, pageSize
}

func validateRewardClaimedEvent(event models.RewardClaimedEvent) error {
	if err := validateAddress("contract_address", event.ContractAddress); err != nil {
		return err
	}
	if err := validateAddress("user", event.User); err != nil {
		return err
	}
	if err := validateUint256Amount(event.Amount); err != nil {
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

func validateRewardClaimedEventQuery(query RewardClaimedEventQuery) error {
	if query.ID < 0 {
		return fmt.Errorf("%w: id must not be negative", ErrInvalidRewardClaimedEvent)
	}
	if query.ContractAddress != "" {
		if err := validateAddress("contract_address", query.ContractAddress); err != nil {
			return err
		}
	}
	if query.User != "" {
		if err := validateAddress("user", query.User); err != nil {
			return err
		}
	}
	if query.TxHash != "" {
		if err := validateHash("tx_hash", query.TxHash); err != nil {
			return err
		}
	}
	if query.BlockNumberFrom != nil && query.BlockNumberTo != nil && *query.BlockNumberFrom > *query.BlockNumberTo {
		return fmt.Errorf("%w: block_number_from must not be greater than block_number_to", ErrInvalidRewardClaimedEvent)
	}

	return nil
}

func normalizeRewardClaimedEvent(event *models.RewardClaimedEvent) {
	event.ContractAddress = strings.ToLower(event.ContractAddress)
	event.User = strings.ToLower(event.User)
	event.TxHash = strings.ToLower(event.TxHash)
	event.BlockHash = strings.ToLower(event.BlockHash)
}

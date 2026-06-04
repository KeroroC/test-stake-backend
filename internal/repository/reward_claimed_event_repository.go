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
	BaseQuery
	User string
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

	page, pageSize := normalizePagination(query.Page, query.PageSize)
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
	normalizeStrings(&event.ContractAddress, &event.User, &event.TxHash, &event.BlockHash)

	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		return fmt.Errorf("create reward claimed event tx_hash=%s log_index=%d: %w", event.TxHash, event.LogIndex, err)
	}

	return nil
}

func (r *RewardClaimedEventRepository) applyQuery(db *gorm.DB, query RewardClaimedEventQuery) *gorm.DB {
	db = applyBaseQuery(db, query.BaseQuery)
	if query.User != "" {
		db = db.Where("user = ?", strings.ToLower(query.User))
	}

	return db
}

func validateRewardClaimedEvent(event models.RewardClaimedEvent) error {
	s := ErrInvalidRewardClaimedEvent
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

func validateRewardClaimedEventQuery(query RewardClaimedEventQuery) error {
	if err := validateBaseQuery(ErrInvalidRewardClaimedEvent, query.BaseQuery); err != nil {
		return err
	}
	if query.User != "" {
		if err := validateAddress(ErrInvalidRewardClaimedEvent, "user", query.User); err != nil {
			return err
		}
	}

	return nil
}

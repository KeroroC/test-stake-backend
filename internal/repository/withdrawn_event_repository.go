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
	defaultWithdrawnEventPage     = 1
	defaultWithdrawnEventPageSize = 20
	maxWithdrawnEventPageSize     = 100
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
	ID              int64
	ContractAddress string
	User            string
	TxHash          string
	BlockNumberFrom *uint64
	BlockNumberTo   *uint64
	Page            int
	PageSize        int
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

	page, pageSize := normalizeWithdrawnEventPagination(query.Page, query.PageSize)
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
	normalizeWithdrawnEvent(event)

	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		return fmt.Errorf("create withdrawn event tx_hash=%s log_index=%d: %w", event.TxHash, event.LogIndex, err)
	}

	return nil
}

func (r *WithdrawnEventRepository) applyQuery(db *gorm.DB, query WithdrawnEventQuery) *gorm.DB {
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

func normalizeWithdrawnEventPagination(page, pageSize int) (int, int) {
	if page <= 0 {
		page = defaultWithdrawnEventPage
	}
	if pageSize <= 0 {
		pageSize = defaultWithdrawnEventPageSize
	}
	if pageSize > maxWithdrawnEventPageSize {
		pageSize = maxWithdrawnEventPageSize
	}

	return page, pageSize
}

func validateWithdrawnEvent(event models.WithdrawnEvent) error {
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

func validateWithdrawnEventQuery(query WithdrawnEventQuery) error {
	if query.ID < 0 {
		return fmt.Errorf("%w: id must not be negative", ErrInvalidWithdrawnEvent)
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
		return fmt.Errorf("%w: block_number_from must not be greater than block_number_to", ErrInvalidWithdrawnEvent)
	}

	return nil
}

func normalizeWithdrawnEvent(event *models.WithdrawnEvent) {
	event.ContractAddress = strings.ToLower(event.ContractAddress)
	event.User = strings.ToLower(event.User)
	event.TxHash = strings.ToLower(event.TxHash)
	event.BlockHash = strings.ToLower(event.BlockHash)
}

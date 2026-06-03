package repository

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"test-stake-backend/internal/models"

	"github.com/ethereum/go-ethereum/common"
	"gorm.io/gorm"
)

const (
	defaultStakedEventPage     = 1
	defaultStakedEventPageSize = 20
	maxStakedEventPageSize     = 100
)

var (
	ErrInvalidStakedEvent  = errors.New("invalid staked event")
	ErrStakedEventNotFound = errors.New("staked event not found")
)

// StakedEventRepository 封装 StakedEvent 表的插入和查询操作。
type StakedEventRepository struct {
	db *gorm.DB
}

// NewStakedEventRepository 创建 StakedEventRepository。
func NewStakedEventRepository(db *gorm.DB) (*StakedEventRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("create staked event repository: db is nil")
	}

	return &StakedEventRepository{db: db}, nil
}

// StakedEventQuery 是分页查询 StakedEvent 的过滤条件。
type StakedEventQuery struct {
	ID              int64
	ContractAddress string
	User            string
	TxHash          string
	BlockNumberFrom *uint64
	BlockNumberTo   *uint64
	Page            int
	PageSize        int
}

// GetByID 按主键查询单条 StakedEvent。
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

// List 按条件分页查询 StakedEvent，并返回总数。
func (r *StakedEventRepository) List(ctx context.Context, query StakedEventQuery) ([]models.StakedEvent, int64, error) {
	if err := validateStakedEventQuery(query); err != nil {
		return nil, 0, err
	}

	page, pageSize := normalizeStakedEventPagination(query.Page, query.PageSize)
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

// Create 创建一条 StakedEvent，写入前会校验必要字段。
func (r *StakedEventRepository) Create(ctx context.Context, event *models.StakedEvent) error {
	if event == nil {
		return fmt.Errorf("%w: event is nil", ErrInvalidStakedEvent)
	}
	if err := validateStakedEvent(*event); err != nil {
		return err
	}
	normalizeStakedEvent(event)

	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		return fmt.Errorf("create staked event tx_hash=%s log_index=%d: %w", event.TxHash, event.LogIndex, err)
	}

	return nil
}

func (r *StakedEventRepository) applyQuery(db *gorm.DB, query StakedEventQuery) *gorm.DB {
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

func normalizeStakedEventPagination(page, pageSize int) (int, int) {
	if page <= 0 {
		page = defaultStakedEventPage
	}
	if pageSize <= 0 {
		pageSize = defaultStakedEventPageSize
	}
	if pageSize > maxStakedEventPageSize {
		pageSize = maxStakedEventPageSize
	}

	return page, pageSize
}

func validateStakedEvent(event models.StakedEvent) error {
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

func validateStakedEventQuery(query StakedEventQuery) error {
	if query.ID < 0 {
		return fmt.Errorf("%w: id must not be negative", ErrInvalidStakedEvent)
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
		return fmt.Errorf("%w: block_number_from must not be greater than block_number_to", ErrInvalidStakedEvent)
	}

	return nil
}

func normalizeStakedEvent(event *models.StakedEvent) {
	event.ContractAddress = strings.ToLower(event.ContractAddress)
	event.User = strings.ToLower(event.User)
	event.TxHash = strings.ToLower(event.TxHash)
	event.BlockHash = strings.ToLower(event.BlockHash)
}

func validateAddress(field, value string) error {
	if !common.IsHexAddress(value) {
		return fmt.Errorf("%w: %s must be a valid hex address", ErrInvalidStakedEvent, field)
	}

	return nil
}

func validateHash(field, value string) error {
	if len(value) != 66 || !strings.HasPrefix(value, "0x") || !isHex(value[2:]) {
		return fmt.Errorf("%w: %s must be a 32-byte hex string", ErrInvalidStakedEvent, field)
	}

	return nil
}

func validateUint256Amount(value string) error {
	amount, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return fmt.Errorf("%w: amount must be a decimal uint256 string", ErrInvalidStakedEvent)
	}
	if amount.Sign() <= 0 {
		return fmt.Errorf("%w: amount must be greater than 0", ErrInvalidStakedEvent)
	}
	if amount.BitLen() > 256 {
		return fmt.Errorf("%w: amount exceeds uint256", ErrInvalidStakedEvent)
	}

	return nil
}

func isHex(value string) bool {
	for _, char := range value {
		switch {
		case char >= '0' && char <= '9':
		case char >= 'a' && char <= 'f':
		case char >= 'A' && char <= 'F':
		default:
			return false
		}
	}

	return true
}

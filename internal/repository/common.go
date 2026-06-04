package repository

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"gorm.io/gorm"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

// normalizePagination 归一化分页参数。
func normalizePagination(page, pageSize int) (int, int) {
	if page <= 0 {
		page = defaultPage
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	return page, pageSize
}

// BaseQuery 所有事件查询的公共过滤条件。
type BaseQuery struct {
	ID              int64
	ContractAddress string
	TxHash          string
	BlockNumberFrom *uint64
	BlockNumberTo   *uint64
	Page            int
	PageSize        int
}

// validateBaseQuery 校验公共查询条件。
func validateBaseQuery(sentinel error, q BaseQuery) error {
	if q.ID < 0 {
		return fmt.Errorf("%w: id must not be negative", sentinel)
	}
	if q.ContractAddress != "" {
		if err := validateAddress(sentinel, "contract_address", q.ContractAddress); err != nil {
			return err
		}
	}
	if q.TxHash != "" {
		if err := validateHash(sentinel, "tx_hash", q.TxHash); err != nil {
			return err
		}
	}
	if q.BlockNumberFrom != nil && q.BlockNumberTo != nil && *q.BlockNumberFrom > *q.BlockNumberTo {
		return fmt.Errorf("%w: block_number_from must not be greater than block_number_to", sentinel)
	}

	return nil
}

// applyBaseQuery 在 gorm.DB 上叠加公共查询条件。
func applyBaseQuery(db *gorm.DB, q BaseQuery) *gorm.DB {
	if q.ID > 0 {
		db = db.Where("id = ?", q.ID)
	}
	if q.ContractAddress != "" {
		db = db.Where("contract_address = ?", strings.ToLower(q.ContractAddress))
	}
	if q.TxHash != "" {
		db = db.Where("tx_hash = ?", strings.ToLower(q.TxHash))
	}
	if q.BlockNumberFrom != nil {
		db = db.Where("block_number >= ?", *q.BlockNumberFrom)
	}
	if q.BlockNumberTo != nil {
		db = db.Where("block_number <= ?", *q.BlockNumberTo)
	}

	return db
}

// validateAddress 校验 hex 地址。
func validateAddress(sentinel error, field, value string) error {
	if !common.IsHexAddress(value) {
		return fmt.Errorf("%w: %s must be a valid hex address", sentinel, field)
	}

	return nil
}

// validateHash 校验 32 字节 hex 哈希。
func validateHash(sentinel error, field, value string) error {
	if len(value) != 66 || !strings.HasPrefix(value, "0x") || !isHex(value[2:]) {
		return fmt.Errorf("%w: %s must be a 32-byte hex string", sentinel, field)
	}

	return nil
}

// validateUint256Amount 校验 uint256 十进制字符串。
func validateUint256Amount(sentinel error, value string) error {
	amount, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return fmt.Errorf("%w: amount must be a decimal uint256 string", sentinel)
	}
	if amount.Sign() < 0 {
		return fmt.Errorf("%w: amount must not be negative", sentinel)
	}
	if amount.BitLen() > 256 {
		return fmt.Errorf("%w: amount exceeds uint256", sentinel)
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

// normalizeStrings 将传入的字符串指针值转为小写。
func normalizeStrings(ptrs ...*string) {
	for _, p := range ptrs {
		*p = strings.ToLower(*p)
	}
}

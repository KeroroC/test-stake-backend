package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"test-stake-backend/internal/models"

	"gorm.io/gorm"
)

type ContractRepository struct {
	db *gorm.DB
}

func NewContractRepository(db *gorm.DB) (*ContractRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("create contract repository: db is nil")
	}

	return &ContractRepository{db: db}, nil
}

func (r *ContractRepository) GetOrCreate(ctx context.Context, address string, startBlock uint64) (*models.Contract, error) {
	address = strings.ToLower(address)

	var contract models.Contract
	err := r.db.WithContext(ctx).
		Where("contract_address = ?", address).
		First(&contract).Error

	if err == nil {
		return &contract, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("get contract by address %s: %w", address, err)
	}

	contract = models.Contract{
		ContractAddress: address,
		LastBlock:       startBlock,
	}
	if err := r.db.WithContext(ctx).Create(&contract).Error; err != nil {
		return nil, fmt.Errorf("create contract for address %s: %w", address, err)
	}

	return &contract, nil
}

func (r *ContractRepository) UpdateLastBlock(ctx context.Context, address string, blockNumber uint64) error {
	address = strings.ToLower(address)

	if err := r.db.WithContext(ctx).
		Model(&models.Contract{}).
		Where("contract_address = ?", address).
		Update("last_block", blockNumber).Error; err != nil {
		return fmt.Errorf("update last_block for %s to %d: %w", address, blockNumber, err)
	}

	return nil
}

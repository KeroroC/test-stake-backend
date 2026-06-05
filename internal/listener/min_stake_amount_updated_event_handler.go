package listener

import (
	"context"
	"fmt"
	"log"
	"math/big"

	pkgabi "test-stake-backend/internal/abi"
	"test-stake-backend/internal/cache"
	"test-stake-backend/internal/models"
	"test-stake-backend/internal/repository"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/redis/go-redis/v9"
)

const minStakeAmountUpdatedEventName = "MinStakeAmountUpdated"

type MinStakeAmountUpdatedEventLogHandler struct {
	repo                         *repository.MinStakeAmountUpdatedEventRepository
	rdb                          *redis.Client
	contractABI                  abi.ABI
	minStakeAmountUpdatedEventID common.Hash
}

func NewMinStakeAmountUpdatedEventLogHandler(repo *repository.MinStakeAmountUpdatedEventRepository, rdb *redis.Client) (*MinStakeAmountUpdatedEventLogHandler, error) {
	if repo == nil {
		return nil, fmt.Errorf("create min stake amount updated event handler: repository is nil")
	}

	contractABI, err := pkgabi.LoadStakeABI()
	if err != nil {
		return nil, err
	}
	minStakeAmountUpdatedEvent, ok := contractABI.Events[minStakeAmountUpdatedEventName]
	if !ok {
		return nil, fmt.Errorf("create min stake amount updated event handler: MinStakeAmountUpdated event not found in ABI")
	}

	return &MinStakeAmountUpdatedEventLogHandler{
		repo:                         repo,
		rdb:                          rdb,
		contractABI:                  contractABI,
		minStakeAmountUpdatedEventID: minStakeAmountUpdatedEvent.ID,
	}, nil
}

func (h *MinStakeAmountUpdatedEventLogHandler) EventName() string {
	return minStakeAmountUpdatedEventName
}

func (h *MinStakeAmountUpdatedEventLogHandler) EventID() common.Hash {
	return h.minStakeAmountUpdatedEventID
}

func (h *MinStakeAmountUpdatedEventLogHandler) Handle(ctx context.Context, eventLog types.Log) error {
	event, err := h.parseLog(eventLog)
	if err != nil {
		return err
	}
	if err := h.repo.Create(ctx, event); err != nil {
		return err
	}

	if err := cache.DeleteByPrefix(ctx, h.rdb, "min-stake-amount-updated:list:"); err != nil {
		log.Printf("cache delete min-stake-amount-updated list prefix: %v", err)
	}

	log.Printf("min stake amount updated event inserted: tx=%s index=%d old_amount=%s new_amount=%s", event.TxHash, event.LogIndex, event.OldAmount, event.NewAmount)
	return nil
}

func (h *MinStakeAmountUpdatedEventLogHandler) parseLog(eventLog types.Log) (*models.MinStakeAmountUpdatedEvent, error) {
	if len(eventLog.Topics) < 1 {
		return nil, fmt.Errorf("MinStakeAmountUpdated log topics length = %d, want at least 1", len(eventLog.Topics))
	}
	if eventLog.Topics[0] != h.minStakeAmountUpdatedEventID {
		return nil, fmt.Errorf("unexpected event topic: %s", eventLog.Topics[0].Hex())
	}

	var unpacked struct {
		OldAmount *big.Int
		NewAmount *big.Int
	}
	if err := h.contractABI.UnpackIntoInterface(&unpacked, minStakeAmountUpdatedEventName, eventLog.Data); err != nil {
		return nil, fmt.Errorf("unpack MinStakeAmountUpdated log data: %w", err)
	}
	if unpacked.OldAmount == nil || unpacked.NewAmount == nil {
		return nil, fmt.Errorf("unpack MinStakeAmountUpdated log data: amount is nil")
	}

	return &models.MinStakeAmountUpdatedEvent{
		ContractAddress: eventLog.Address.Hex(),
		OldAmount:       unpacked.OldAmount.String(),
		NewAmount:       unpacked.NewAmount.String(),
		TxHash:          eventLog.TxHash.Hex(),
		BlockNumber:     eventLog.BlockNumber,
		LogIndex:        eventLog.Index,
		BlockHash:       eventLog.BlockHash.Hex(),
	}, nil
}

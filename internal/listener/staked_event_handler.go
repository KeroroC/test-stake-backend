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

const stakedEventName = "Staked"

type StakedEventLogHandler struct {
	repo          *repository.StakedEventRepository
	rdb           *redis.Client
	contractABI   abi.ABI
	stakedEventID common.Hash
}

func NewStakedEventLogHandler(repo *repository.StakedEventRepository, rdb *redis.Client) (*StakedEventLogHandler, error) {
	if repo == nil {
		return nil, fmt.Errorf("create staked event handler: repository is nil")
	}

	contractABI, err := pkgabi.LoadStakeABI()
	if err != nil {
		return nil, err
	}
	stakedEvent, ok := contractABI.Events[stakedEventName]
	if !ok {
		return nil, fmt.Errorf("create staked event handler: Staked event not found in ABI")
	}

	return &StakedEventLogHandler{
		repo:          repo,
		rdb:           rdb,
		contractABI:   contractABI,
		stakedEventID: stakedEvent.ID,
	}, nil
}

func (h *StakedEventLogHandler) EventName() string {
	return stakedEventName
}

func (h *StakedEventLogHandler) EventID() common.Hash {
	return h.stakedEventID
}

func (h *StakedEventLogHandler) Handle(ctx context.Context, eventLog types.Log) error {
	event, err := h.parseLog(eventLog)
	if err != nil {
		return err
	}
	if err := h.repo.Create(ctx, event); err != nil {
		return err
	}

	if err := cache.DeleteByPrefix(ctx, h.rdb, "staked:list:"); err != nil {
		log.Printf("cache delete staked list prefix: %v", err)
	}

	log.Printf("staked event inserted: tx=%s index=%d user=%s amount=%s", event.TxHash, event.LogIndex, event.User, event.Amount)
	return nil
}

func (h *StakedEventLogHandler) parseLog(eventLog types.Log) (*models.StakedEvent, error) {
	if len(eventLog.Topics) < 2 {
		return nil, fmt.Errorf("Staked log topics length = %d, want at least 2", len(eventLog.Topics))
	}
	if eventLog.Topics[0] != h.stakedEventID {
		return nil, fmt.Errorf("unexpected event topic: %s", eventLog.Topics[0].Hex())
	}

	var unpacked struct {
		Amount *big.Int
	}
	if err := h.contractABI.UnpackIntoInterface(&unpacked, stakedEventName, eventLog.Data); err != nil {
		return nil, fmt.Errorf("unpack Staked log data: %w", err)
	}
	if unpacked.Amount == nil {
		return nil, fmt.Errorf("unpack Staked log data: amount is nil")
	}

	return &models.StakedEvent{
		ContractAddress: eventLog.Address.Hex(),
		User:            common.BytesToAddress(eventLog.Topics[1].Bytes()).Hex(),
		Amount:          unpacked.Amount.String(),
		TxHash:          eventLog.TxHash.Hex(),
		BlockNumber:     eventLog.BlockNumber,
		LogIndex:        eventLog.Index,
		BlockHash:       eventLog.BlockHash.Hex(),
	}, nil
}

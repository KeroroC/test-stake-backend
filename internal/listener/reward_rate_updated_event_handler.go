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

const rewardRateUpdatedEventName = "RewardRateUpdated"

type RewardRateUpdatedEventLogHandler struct {
	repo                     *repository.RewardRateUpdatedEventRepository
	rdb                      *redis.Client
	contractABI              abi.ABI
	rewardRateUpdatedEventID common.Hash
}

func NewRewardRateUpdatedEventLogHandler(repo *repository.RewardRateUpdatedEventRepository, rdb *redis.Client) (*RewardRateUpdatedEventLogHandler, error) {
	if repo == nil {
		return nil, fmt.Errorf("create reward rate updated event handler: repository is nil")
	}

	contractABI, err := pkgabi.LoadStakeABI()
	if err != nil {
		return nil, err
	}
	rewardRateUpdatedEvent, ok := contractABI.Events[rewardRateUpdatedEventName]
	if !ok {
		return nil, fmt.Errorf("create reward rate updated event handler: RewardRateUpdated event not found in ABI")
	}

	return &RewardRateUpdatedEventLogHandler{
		repo:                     repo,
		rdb:                      rdb,
		contractABI:              contractABI,
		rewardRateUpdatedEventID: rewardRateUpdatedEvent.ID,
	}, nil
}

func (h *RewardRateUpdatedEventLogHandler) EventName() string {
	return rewardRateUpdatedEventName
}

func (h *RewardRateUpdatedEventLogHandler) EventID() common.Hash {
	return h.rewardRateUpdatedEventID
}

func (h *RewardRateUpdatedEventLogHandler) Handle(ctx context.Context, eventLog types.Log) error {
	event, err := h.parseLog(eventLog)
	if err != nil {
		return err
	}
	if err := h.repo.Create(ctx, event); err != nil {
		return err
	}

	if err := cache.DeleteByPrefix(ctx, h.rdb, "reward-rate-updated:list:"); err != nil {
		log.Printf("cache delete reward-rate-updated list prefix: %v", err)
	}

	log.Printf("reward rate updated event inserted: tx=%s index=%d old_rate=%s new_rate=%s", event.TxHash, event.LogIndex, event.OldRate, event.NewRate)
	return nil
}

func (h *RewardRateUpdatedEventLogHandler) parseLog(eventLog types.Log) (*models.RewardRateUpdatedEvent, error) {
	if len(eventLog.Topics) < 1 {
		return nil, fmt.Errorf("RewardRateUpdated log topics length = %d, want at least 1", len(eventLog.Topics))
	}
	if eventLog.Topics[0] != h.rewardRateUpdatedEventID {
		return nil, fmt.Errorf("unexpected event topic: %s", eventLog.Topics[0].Hex())
	}

	var unpacked struct {
		OldRate *big.Int
		NewRate *big.Int
	}
	if err := h.contractABI.UnpackIntoInterface(&unpacked, rewardRateUpdatedEventName, eventLog.Data); err != nil {
		return nil, fmt.Errorf("unpack RewardRateUpdated log data: %w", err)
	}
	if unpacked.OldRate == nil || unpacked.NewRate == nil {
		return nil, fmt.Errorf("unpack RewardRateUpdated log data: rate is nil")
	}

	return &models.RewardRateUpdatedEvent{
		ContractAddress: eventLog.Address.Hex(),
		OldRate:         unpacked.OldRate.String(),
		NewRate:         unpacked.NewRate.String(),
		TxHash:          eventLog.TxHash.Hex(),
		BlockNumber:     eventLog.BlockNumber,
		LogIndex:        eventLog.Index,
		BlockHash:       eventLog.BlockHash.Hex(),
	}, nil
}

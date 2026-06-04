package listener

import (
	"context"
	"fmt"
	"log"
	"math/big"

	pkgabi "test-stake-backend/internal/abi"
	"test-stake-backend/internal/models"
	"test-stake-backend/internal/repository"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	rewardClaimedEventName = "RewardClaimed"
)

type RewardClaimedEventLogHandler struct {
	repo              *repository.RewardClaimedEventRepository
	contractABI       abi.ABI
	rewardClaimedEventID common.Hash
}

func NewRewardClaimedEventLogHandler(repo *repository.RewardClaimedEventRepository) (*RewardClaimedEventLogHandler, error) {
	if repo == nil {
		return nil, fmt.Errorf("create reward claimed event handler: repository is nil")
	}

	contractABI, err := pkgabi.LoadStakeABI()
	if err != nil {
		return nil, err
	}
	rewardClaimedEvent, ok := contractABI.Events[rewardClaimedEventName]
	if !ok {
		return nil, fmt.Errorf("create reward claimed event handler: RewardClaimed event not found in ABI")
	}

	return &RewardClaimedEventLogHandler{
		repo:                 repo,
		contractABI:          contractABI,
		rewardClaimedEventID: rewardClaimedEvent.ID,
	}, nil
}

func (h *RewardClaimedEventLogHandler) EventName() string {
	return rewardClaimedEventName
}

func (h *RewardClaimedEventLogHandler) EventID() common.Hash {
	return h.rewardClaimedEventID
}

func (h *RewardClaimedEventLogHandler) Handle(ctx context.Context, eventLog types.Log) error {
	event, err := h.parseLog(eventLog)
	if err != nil {
		return err
	}
	if err := h.repo.Create(ctx, event); err != nil {
		return err
	}

	log.Printf("reward claimed event inserted: tx=%s index=%d user=%s amount=%s", event.TxHash, event.LogIndex, event.User, event.Amount)
	return nil
}

func (h *RewardClaimedEventLogHandler) parseLog(eventLog types.Log) (*models.RewardClaimedEvent, error) {
	if len(eventLog.Topics) < 2 {
		return nil, fmt.Errorf("RewardClaimed log topics length = %d, want at least 2", len(eventLog.Topics))
	}
	if eventLog.Topics[0] != h.rewardClaimedEventID {
		return nil, fmt.Errorf("unexpected event topic: %s", eventLog.Topics[0].Hex())
	}

	var unpacked struct {
		Amount *big.Int
	}
	if err := h.contractABI.UnpackIntoInterface(&unpacked, rewardClaimedEventName, eventLog.Data); err != nil {
		return nil, fmt.Errorf("unpack RewardClaimed log data: %w", err)
	}
	if unpacked.Amount == nil {
		return nil, fmt.Errorf("unpack RewardClaimed log data: amount is nil")
	}

	return &models.RewardClaimedEvent{
		ContractAddress: eventLog.Address.Hex(),
		User:            common.BytesToAddress(eventLog.Topics[1].Bytes()).Hex(),
		Amount:          unpacked.Amount.String(),
		TxHash:          eventLog.TxHash.Hex(),
		BlockNumber:     eventLog.BlockNumber,
		LogIndex:        eventLog.Index,
		BlockHash:       eventLog.BlockHash.Hex(),
	}, nil
}


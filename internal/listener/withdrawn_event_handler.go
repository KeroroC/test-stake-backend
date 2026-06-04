package listener

import (
	"context"
	"fmt"
	"log"
	"math/big"

	"test-stake-backend/internal/models"
	"test-stake-backend/internal/repository"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	withdrawnEventName = "Withdrawn"
)

type WithdrawnEventLogHandler struct {
	repo            *repository.WithdrawnEventRepository
	contractABI     abi.ABI
	withdrawnEventID common.Hash
}

func NewWithdrawnEventLogHandler(repo *repository.WithdrawnEventRepository) (*WithdrawnEventLogHandler, error) {
	if repo == nil {
		return nil, fmt.Errorf("create withdrawn event handler: repository is nil")
	}

	contractABI, err := loadStakeABI()
	if err != nil {
		return nil, err
	}
	withdrawnEvent, ok := contractABI.Events[withdrawnEventName]
	if !ok {
		return nil, fmt.Errorf("create withdrawn event handler: Withdrawn event not found in ABI")
	}

	return &WithdrawnEventLogHandler{
		repo:             repo,
		contractABI:      contractABI,
		withdrawnEventID: withdrawnEvent.ID,
	}, nil
}

func (h *WithdrawnEventLogHandler) EventName() string {
	return withdrawnEventName
}

func (h *WithdrawnEventLogHandler) EventID() common.Hash {
	return h.withdrawnEventID
}

func (h *WithdrawnEventLogHandler) Handle(ctx context.Context, eventLog types.Log) error {
	event, err := h.parseLog(eventLog)
	if err != nil {
		return err
	}
	if err := h.repo.Create(ctx, event); err != nil {
		return err
	}

	log.Printf("withdrawn event inserted: tx=%s index=%d user=%s amount=%s", event.TxHash, event.LogIndex, event.User, event.Amount)
	return nil
}

func (h *WithdrawnEventLogHandler) parseLog(eventLog types.Log) (*models.WithdrawnEvent, error) {
	if len(eventLog.Topics) < 2 {
		return nil, fmt.Errorf("Withdrawn log topics length = %d, want at least 2", len(eventLog.Topics))
	}
	if eventLog.Topics[0] != h.withdrawnEventID {
		return nil, fmt.Errorf("unexpected event topic: %s", eventLog.Topics[0].Hex())
	}

	var unpacked struct {
		Amount *big.Int
	}
	if err := h.contractABI.UnpackIntoInterface(&unpacked, withdrawnEventName, eventLog.Data); err != nil {
		return nil, fmt.Errorf("unpack Withdrawn log data: %w", err)
	}
	if unpacked.Amount == nil {
		return nil, fmt.Errorf("unpack Withdrawn log data: amount is nil")
	}

	return &models.WithdrawnEvent{
		ContractAddress: eventLog.Address.Hex(),
		User:            common.BytesToAddress(eventLog.Topics[1].Bytes()).Hex(),
		Amount:          unpacked.Amount.String(),
		TxHash:          eventLog.TxHash.Hex(),
		BlockNumber:     eventLog.BlockNumber,
		LogIndex:        eventLog.Index,
		BlockHash:       eventLog.BlockHash.Hex(),
	}, nil
}

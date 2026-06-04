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

const stakedEventName = "Staked"

// StakedEventLogHandler 解析 Staked 事件并写入数据库。
type StakedEventLogHandler struct {
	repo          *repository.StakedEventRepository
	contractABI   abi.ABI
	stakedEventID common.Hash
}

// NewStakedEventLogHandler 创建 StakedEventLogHandler。
func NewStakedEventLogHandler(repo *repository.StakedEventRepository) (*StakedEventLogHandler, error) {
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
		contractABI:   contractABI,
		stakedEventID: stakedEvent.ID,
	}, nil
}

// EventName 返回事件名称。
func (h *StakedEventLogHandler) EventName() string {
	return stakedEventName
}

// EventID 返回事件 topic0。
func (h *StakedEventLogHandler) EventID() common.Hash {
	return h.stakedEventID
}

// Handle 解析 Staked 日志并插入数据库。
func (h *StakedEventLogHandler) Handle(ctx context.Context, eventLog types.Log) error {
	event, err := h.parseLog(eventLog)
	if err != nil {
		return err
	}
	if err := h.repo.Create(ctx, event); err != nil {
		return err
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


package listener

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"test-stake-backend/internal/repository"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	listenerRetryDelay  = 5 * time.Second
	listenerLogChanSize = 128
)

type ContractEventHandler interface {
	EventName() string
	EventID() common.Hash
	Handle(ctx context.Context, eventLog types.Log) error
}

type ContractEventListener struct {
	wsURL        string
	contractAddr common.Address
	handlers     map[common.Hash]ContractEventHandler
	contractRepo *repository.ContractRepository
	startBlock   uint64
}

func NewContractEventListener(wsURL string, contractAddress string, contractRepo *repository.ContractRepository, startBlock uint64) (*ContractEventListener, error) {
	if strings.TrimSpace(wsURL) == "" {
		return nil, fmt.Errorf("create contract event listener: eth ws_url is empty")
	}
	if !common.IsHexAddress(contractAddress) {
		return nil, fmt.Errorf("create contract event listener: invalid contract address")
	}
	if contractRepo == nil {
		return nil, fmt.Errorf("create contract event listener: contract repository is nil")
	}
	if startBlock == 0 {
		return nil, fmt.Errorf("create contract event listener: eth start_block must greater than 0 (set it to the contract deployment block)")
	}

	return &ContractEventListener{
		wsURL:        wsURL,
		contractAddr: common.HexToAddress(contractAddress),
		handlers:     make(map[common.Hash]ContractEventHandler),
		contractRepo: contractRepo,
		startBlock:   startBlock,
	}, nil
}

func (l *ContractEventListener) Register(handler ContractEventHandler) error {
	if handler == nil {
		return fmt.Errorf("register contract event handler: handler is nil")
	}
	eventID := handler.EventID()
	if eventID == (common.Hash{}) {
		return fmt.Errorf("register contract event handler: empty event id")
	}
	if _, exists := l.handlers[eventID]; exists {
		return fmt.Errorf("register contract event handler: duplicate event id %s", eventID.Hex())
	}

	l.handlers[eventID] = handler
	return nil
}

func (l *ContractEventListener) Start(ctx context.Context) {
	for {
		if err := l.listen(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("contract event listener stopped: %v; retrying in %s", err, listenerRetryDelay)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(listenerRetryDelay):
		}
	}
}

func (l *ContractEventListener) listen(ctx context.Context) error {
	if len(l.handlers) == 0 {
		return fmt.Errorf("listen contract events: no handlers registered")
	}

	client, err := ethclient.DialContext(ctx, l.wsURL)
	if err != nil {
		return fmt.Errorf("dial eth websocket: %w", err)
	}
	defer client.Close()

	// 读取或创建合约记录，获取 lastBlock
	contract, err := l.contractRepo.GetOrCreate(ctx, l.contractAddr.Hex(), l.startBlock)
	if err != nil {
		return fmt.Errorf("get contract record: %w", err)
	}

	// 获取链上最新区块
	currentHead, err := client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("get current block number: %w", err)
	}

	// 回放落后区块
	if contract.LastBlock < currentHead {
		if err := l.replay(ctx, client, contract.LastBlock+1, currentHead); err != nil {
			return fmt.Errorf("replay blocks %d-%d: %w", contract.LastBlock+1, currentHead, err)
		}
		log.Printf("replay completed: blocks %d -> %d", contract.LastBlock+1, currentHead)
	}

	// 切换到实时订阅
	query := ethereum.FilterQuery{
		Addresses: []common.Address{l.contractAddr},
		Topics:    [][]common.Hash{l.eventIDs()},
	}
	logs := make(chan types.Log, listenerLogChanSize)
	sub, err := client.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		return fmt.Errorf("subscribe contract logs: %w", err)
	}
	defer sub.Unsubscribe()

	log.Printf("contract event listener started: contract=%s handlers=%d", l.contractAddr.Hex(), len(l.handlers))
	for {
		select {
		case err := <-sub.Err():
			return fmt.Errorf("subscription error: %w", err)
		case eventLog := <-logs:
			if eventLog.Removed {
				continue
			}
			l.dispatch(ctx, eventLog)
			if err := l.contractRepo.UpdateLastBlock(ctx, l.contractAddr.Hex(), eventLog.BlockNumber); err != nil {
				log.Printf("update last block to %d failed: %v", eventLog.BlockNumber, err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (l *ContractEventListener) replay(ctx context.Context, client *ethclient.Client, fromBlock, toBlock uint64) error {
	const replayBatchSize = 1000

	for from := fromBlock; from <= toBlock; from += replayBatchSize {
		to := from + replayBatchSize - 1
		to = min(to, toBlock)

		query := ethereum.FilterQuery{
			FromBlock: new(big.Int).SetUint64(from),
			ToBlock:   new(big.Int).SetUint64(to),
			Addresses: []common.Address{l.contractAddr},
			Topics:    [][]common.Hash{l.eventIDs()},
		}

		logs, err := client.FilterLogs(ctx, query)
		if err != nil {
			return fmt.Errorf("filter logs block %d-%d: %w", from, to, err)
		}

		for _, eventLog := range logs {
			if eventLog.Removed {
				continue
			}
			l.dispatch(ctx, eventLog)
		}

		if err := l.contractRepo.UpdateLastBlock(ctx, l.contractAddr.Hex(), to); err != nil {
			return fmt.Errorf("update last block to %d: %w", to, err)
		}

		log.Printf("replayed blocks %d-%d (%d events)", from, to, len(logs))
	}

	return nil
}

func (l *ContractEventListener) dispatch(ctx context.Context, eventLog types.Log) {
	if len(eventLog.Topics) == 0 {
		log.Printf("ignore contract log without topics: tx=%s index=%d", eventLog.TxHash.Hex(), eventLog.Index)
		return
	}

	handler, ok := l.handlers[eventLog.Topics[0]]
	if !ok {
		log.Printf("ignore unregistered event: topic=%s tx=%s index=%d", eventLog.Topics[0].Hex(), eventLog.TxHash.Hex(), eventLog.Index)
		return
	}
	if err := handler.Handle(ctx, eventLog); err != nil {
		log.Printf("handle %s event failed: tx=%s index=%d err=%v", handler.EventName(), eventLog.TxHash.Hex(), eventLog.Index, err)
	}
}

func (l *ContractEventListener) eventIDs() []common.Hash {
	eventIDs := make([]common.Hash, 0, len(l.handlers))
	for eventID := range l.handlers {
		eventIDs = append(eventIDs, eventID)
	}

	return eventIDs
}

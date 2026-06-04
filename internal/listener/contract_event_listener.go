package listener

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	listenerRetryDelay  = 5 * time.Second
	listenerLogChanSize = 128
)

// ContractEventHandler 处理某一种合约事件日志。
type ContractEventHandler interface {
	EventName() string
	EventID() common.Hash
	Handle(ctx context.Context, eventLog types.Log) error
}

// ContractEventListener 负责订阅合约事件，并按 topic 分发给已注册的 handler。
type ContractEventListener struct {
	wsURL        string
	contractAddr common.Address
	handlers     map[common.Hash]ContractEventHandler
}

// NewContractEventListener 创建通用合约事件监听器。
func NewContractEventListener(wsURL string, contractAddress string) (*ContractEventListener, error) {
	if strings.TrimSpace(wsURL) == "" {
		return nil, fmt.Errorf("create contract event listener: eth ws_url is empty")
	}
	if !common.IsHexAddress(contractAddress) {
		return nil, fmt.Errorf("create contract event listener: invalid contract address")
	}

	return &ContractEventListener{
		wsURL:        wsURL,
		contractAddr: common.HexToAddress(contractAddress),
		handlers:     make(map[common.Hash]ContractEventHandler),
	}, nil
}

// Register 注册一种事件处理器。
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

// Start 启动监听循环，直到 ctx 取消。
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
		case <-ctx.Done():
			return ctx.Err()
		}
	}
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

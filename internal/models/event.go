package models

import (
	"time"
)

// ContractEvent 是 API 返回和数据库持久化使用的统一事件模型。
// 保留原始 topics/data，方便后续 ABI 参数解析。
type ContractEvent struct {
	ID              int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ContractAddress string    `gorm:"type:varchar(42);not null;index:idx_contract_events_contract_block,priority:1" json:"contract_address"`
	EventName       string    `gorm:"type:varchar(64);index:idx_contract_events_event_name" json:"event_name"`
	EventSignature  string    `gorm:"type:varchar(255);not null" json:"event_signature"`
	Topic0          string    `gorm:"type:varchar(66);not null" json:"topic0"`
	TxHash          string    `gorm:"type:varchar(66);not null;uniqueIndex:uniq_contract_events_tx_log,priority:1;index:idx_contract_events_tx_hash" json:"tx_hash"`
	BlockNumber     uint64    `gorm:"not null;index:idx_contract_events_contract_block,priority:2" json:"block_number"`
	LogIndex        uint      `gorm:"not null;uniqueIndex:uniq_contract_events_tx_log,priority:2" json:"log_index"`
	Topics          []string  `gorm:"type:json;serializer:json;not null" json:"topics"`
	Data            string    `gorm:"type:longtext;not null" json:"data"`
	BlockHash       string    `gorm:"type:varchar(66);not null" json:"block_hash"`
	ObservedAt      time.Time `gorm:"autoCreateTime" json:"observed_at"`
}

package models

import (
	"time"
)

// ContractEvent 是 API 返回和数据库持久化使用的统一事件模型。
// 保留原始 topics/data，方便后续 ABI 参数解析。
//type ContractEvent struct {
//	ID              int64     `gorm:"primaryKey;autoIncrement" json:"id"`
//	ContractAddress string    `gorm:"type:varchar(42);not null;index:idx_contract_events_contract_block,priority:1" json:"contract_address"`
//	EventName       string    `gorm:"type:varchar(64);index:idx_contract_events_event_name" json:"event_name"`
//	EventSignature  string    `gorm:"type:varchar(255);not null" json:"event_signature"`
//	Topic0          string    `gorm:"type:varchar(66);not null" json:"topic0"`
//	TxHash          string    `gorm:"type:varchar(66);not null;uniqueIndex:uniq_contract_events_tx_log,priority:1;index:idx_contract_events_tx_hash" json:"tx_hash"`
//	BlockNumber     uint64    `gorm:"not null;index:idx_contract_events_contract_block,priority:2" json:"block_number"`
//	LogIndex        uint      `gorm:"not null;uniqueIndex:uniq_contract_events_tx_log,priority:2" json:"log_index"`
//	Topics          []string  `gorm:"type:json;serializer:json;not null" json:"topics"`
//	Data            string    `gorm:"type:longtext;not null" json:"data"`
//	BlockHash       string    `gorm:"type:varchar(66);not null" json:"block_hash"`
//	ObservedAt      time.Time `gorm:"autoCreateTime" json:"observed_at"`
//}

// StakedEvent 对应 Stake ABI 中的 Staked 事件：
// Staked(address indexed user, uint256 amount)
type StakedEvent struct {
	ID              int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ContractAddress string    `gorm:"type:varchar(42);not null;index:idx_staked_events_contract_block,priority:1" json:"contract_address"`
	User            string    `gorm:"type:varchar(42);not null;index:idx_staked_events_user" json:"user"`
	Amount          string    `gorm:"type:varchar(78);not null;comment:uint256 decimal string" json:"amount"`
	TxHash          string    `gorm:"type:varchar(66);not null;uniqueIndex:uniq_staked_events_tx_log,priority:1" json:"tx_hash"`
	BlockNumber     uint64    `gorm:"not null;index:idx_staked_events_contract_block,priority:2" json:"block_number"`
	LogIndex        uint      `gorm:"not null;uniqueIndex:uniq_staked_events_tx_log,priority:2" json:"log_index"`
	BlockHash       string    `gorm:"type:varchar(66);not null" json:"block_hash"`
	InsertedAt      time.Time `gorm:"autoCreateTime" json:"inserted_at"`
}

// RewardClaimedEvent 对应 RewardClaimed 事件
// RewardClaimed(address indexed user, uint256 amount)
type RewardClaimedEvent struct {
	ID              int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ContractAddress string    `gorm:"size:42;not null;index:idx_claim_events_contract_block,priority:1" json:"contract_address"`
	User            string    `gorm:"size:42;not null;index:idx_claim_events_user" json:"user"`
	Amount          string    `gorm:"size:78;not null;comment:uint256 decimal string" json:"amount"`
	TxHash          string    `gorm:"size:66;not null;uniqueIndex:uniq_claim_events_tx_log,priority:1" json:"tx_hash"`
	BlockNumber     uint64    `gorm:"not null;index:idx_claim_events_contract_block,priority:2" json:"block_number"`
	LogIndex        uint      `gorm:"not null;uniqueIndex:uniq_claim_events_tx_log,priority:2" json:"log_index"`
	BlockHash       string    `gorm:"size:66;not null" json:"block_hash"`
	InsertedAt      time.Time `gorm:"autoCreateTime" json:"inserted_at"`
}

// WithdrawnEvent 对应 Withdrawn 事件
// Withdrawn(address indexed user, uint256 amount)
type WithdrawnEvent struct {
	ID              int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ContractAddress string    `gorm:"size:42;not null;index:idx_withdrawn_events_contract_block,priority:1" json:"contract_address"`
	User            string    `gorm:"size:42;not null;index:idx_withdrawn_events_user" json:"user"`
	Amount          string    `gorm:"size:78;not null;comment:uint256 decimal string" json:"amount"`
	TxHash          string    `gorm:"size:66;not null;uniqueIndex:uniq_withdrawn_events_tx_log,priority:1" json:"tx_hash"`
	BlockNumber     uint64    `gorm:"not null;index:idx_withdrawn_events_contract_block,priority:2" json:"block_number"`
	LogIndex        uint      `gorm:"not null;uniqueIndex:uniq_withdrawn_events_tx_log,priority:2" json:"log_index"`
	BlockHash       string    `gorm:"size:66;not null" json:"block_hash"`
	InsertedAt      time.Time `gorm:"autoCreateTime" json:"inserted_at"`
}

// MinStakeAmountUpdatedEvent 对应 MinStakeAmountUpdated 事件
// MinStakeAmountUpdated(uint256 oldAmount, uint256 newAmount)
type MinStakeAmountUpdatedEvent struct {
	ID              int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ContractAddress string    `gorm:"size:42;not null;index:idx_min_stake_updated_events_contract_block,priority:1" json:"contract_address"`
	OldAmount       string    `gorm:"size:78;not null;comment:uint256 decimal string" json:"old_amount"`
	NewAmount       string    `gorm:"size:78;not null;comment:uint256 decimal string" json:"new_amount"`
	TxHash          string    `gorm:"size:66;not null;uniqueIndex:uniq_min_stake_updated_events_tx_log,priority:1" json:"tx_hash"`
	BlockNumber     uint64    `gorm:"not null;index:idx_min_stake_updated_events_contract_block,priority:2" json:"block_number"`
	LogIndex        uint      `gorm:"not null;uniqueIndex:uniq_min_stake_updated_events_tx_log,priority:2" json:"log_index"`
	BlockHash       string    `gorm:"size:66;not null" json:"block_hash"`
	InsertedAt      time.Time `gorm:"autoCreateTime" json:"inserted_at"`
}

// RewardRateUpdatedEvent 对应 RewardRateUpdated 事件
// RewardRateUpdated(uint256 oldRate, uint256 newRate)
type RewardRateUpdatedEvent struct {
	ID              int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ContractAddress string    `gorm:"size:42;not null;index:idx_reward_rate_updated_events_contract_block,priority:1" json:"contract_address"`
	OldRate         string    `gorm:"size:78;not null;comment:uint256 decimal string" json:"old_rate"`
	NewRate         string    `gorm:"size:78;not null;comment:uint256 decimal string" json:"new_rate"`
	TxHash          string    `gorm:"size:66;not null;uniqueIndex:uniq_reward_rate_updated_events_tx_log,priority:1" json:"tx_hash"`
	BlockNumber     uint64    `gorm:"not null;index:idx_reward_rate_updated_events_contract_block,priority:2" json:"block_number"`
	LogIndex        uint      `gorm:"not null;uniqueIndex:uniq_reward_rate_updated_events_tx_log,priority:2" json:"log_index"`
	BlockHash       string    `gorm:"size:66;not null" json:"block_hash"`
	InsertedAt      time.Time `gorm:"autoCreateTime" json:"inserted_at"`
}

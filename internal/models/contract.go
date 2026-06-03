package models

type Contract struct {
	ID              int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	ChainId         int    `gorm:"index;default:1" json:"chain_id"`
	ContractAddress string `gorm:"index;size:42" json:"contract_address"`
	EnableSync      bool   `gorm:"default:true;comment:是否启用同步" json:"enable_sync"`
	LastBlock       uint64 `gorm:"default:0;comment:最后同步区块高度" json:"last_block"`
}

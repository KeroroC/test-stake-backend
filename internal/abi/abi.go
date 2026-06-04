package abi

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

//go:embed Stake.abi.json
var StakeABI []byte

// LoadStakeABI 解析嵌入的 Stake 合约 ABI。
func LoadStakeABI() (abi.ABI, error) {
	contractABI, err := abi.JSON(strings.NewReader(string(StakeABI)))
	if err != nil {
		return abi.ABI{}, fmt.Errorf("parse stake ABI: %w", err)
	}

	return contractABI, nil
}

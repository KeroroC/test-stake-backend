package listener

import (
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const stakeABIPath = "internal/abi/Stake.abi.json"

func loadStakeABI() (abi.ABI, error) {
	abiBytes, err := os.ReadFile(stakeABIPath)
	if err != nil {
		return abi.ABI{}, fmt.Errorf("read stake ABI file: %w", err)
	}

	contractABI, err := abi.JSON(strings.NewReader(string(abiBytes)))
	if err != nil {
		return abi.ABI{}, fmt.Errorf("parse stake ABI: %w", err)
	}

	return contractABI, nil
}

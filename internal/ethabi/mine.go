package ethabi

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var hashABI = mustABI()

func MineCalldata(nonce *big.Int) ([]byte, error) {
	return hashABI.Pack("mine", nonce)
}

func mustABI() abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(`[{"inputs":[{"internalType":"uint256","name":"nonce","type":"uint256"}],"name":"mine","outputs":[],"stateMutability":"nonpayable","type":"function"}]`))
	if err != nil {
		panic(err)
	}
	return parsed
}


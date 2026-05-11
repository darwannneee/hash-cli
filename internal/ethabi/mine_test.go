package ethabi

import (
	"encoding/hex"
	"math/big"
	"testing"
)

func TestMineCalldata(t *testing.T) {
	data, err := MineCalldata(big.NewInt(42))
	if err != nil {
		t.Fatalf("MineCalldata returned error: %v", err)
	}
	if len(data) != 36 {
		t.Fatalf("len(data) = %d", len(data))
	}
	if hex.EncodeToString(data[:4]) != "1d5a4eb6" {
		t.Fatalf("selector = %x", data[:4])
	}
	if data[35] != 42 {
		t.Fatalf("encoded nonce tail = %d", data[35])
	}
}


package miner

import (
	"math/big"
	"testing"
)

func TestMeetsDifficulty(t *testing.T) {
	result := new(big.Int).SetUint64(99)
	difficulty := new(big.Int).SetUint64(100)
	if !MeetsDifficulty(result, difficulty) {
		t.Fatal("expected result below difficulty")
	}
	if MeetsDifficulty(difficulty, difficulty) {
		t.Fatal("expected equal result to fail")
	}
}


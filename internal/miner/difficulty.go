package miner

import "math/big"

func MeetsDifficulty(result *big.Int, difficulty *big.Int) bool {
	return result.Cmp(difficulty) < 0
}


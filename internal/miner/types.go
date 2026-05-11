package miner

import (
	"context"
	"math/big"
	"time"
)

type Snapshot struct {
	Running           bool
	Hashrate          float64
	TotalHashes       uint64
	Accepted          uint64
	Rejected          uint64
	Epoch             string
	EpochBlocksLeft   string
	Reward            string
	Difficulty        string
	Minted            string
	Remaining         string
	BlockNumber       string
	MintsInBlock      string
	LastNonceStart    uint64
	LastNonceEnd      uint64
	LastCandidate     uint64
	LastTX            string
	Wallet            string
	BalanceWei        string
	GPU               string
	DryRun            bool
	StartedAt         time.Time
	LastStateRefresh  time.Time
	LastSubmitAttempt time.Time
}

type Event struct {
	At      time.Time
	Level   string
	Message string
}

type WorkState struct {
	Challenge        [32]byte
	Difficulty       *big.Int
	Epoch            *big.Int
	EpochBlocksLeft  *big.Int
	Reward           *big.Int
	Minted           *big.Int
	Remaining        *big.Int
	BlockNumber      uint64
	MintsInBlock     *big.Int
	WalletBalanceWei *big.Int
}

type SearchResult struct {
	Found      bool
	Nonce      uint64
	HashesDone uint64
}

type Searcher interface {
	Search(challenge [32]byte, difficulty *big.Int, startNonce uint64, count uint64) (SearchResult, error)
}

type RefreshFunc func(ctx context.Context) (WorkState, error)

type SubmitFunc func(ctx context.Context, nonce *big.Int) (string, error)


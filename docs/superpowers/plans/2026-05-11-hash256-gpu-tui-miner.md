# HASH256 GPU TUI Miner Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go terminal UI miner for HASH256 that uses an NVIDIA RTX 4090 via CUDA/cgo, reads wallet settings from `.env`, and submits valid `mine(uint256)` transactions.

**Architecture:** Go owns configuration, RPC, wallet signing, transaction submission, state coordination, and TUI rendering. CUDA owns only the hot loop: search nonce ranges for `keccak256(abi.encode(challenge, nonce)) < difficulty`. A dry-run mode allows validation without sending transactions.

**Tech Stack:** Go 1.22+, go-ethereum, Bubble Tea, Lip Gloss, godotenv, cgo, CUDA C, nvcc.

---

## Files

- Create: `.gitignore`
- Create: `.env.example`
- Create: `go.mod`
- Create: `cmd/hashminer/main.go`
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Create: `internal/ethabi/mine.go`
- Create: `internal/ethabi/mine_test.go`
- Create: `internal/chain/client.go`
- Create: `internal/wallet/wallet.go`
- Create: `internal/miner/types.go`
- Create: `internal/miner/engine.go`
- Create: `internal/miner/difficulty.go`
- Create: `internal/miner/difficulty_test.go`
- Create: `internal/cuda/cuda.go`
- Create: `cuda/hash_miner.h`
- Create: `cuda/hash_miner.cu`
- Create: `internal/tui/model.go`
- Create: `internal/tui/view.go`
- Create: `internal/tui/keys.go`
- Create: `README.md`

## Task 1: Project Skeleton And Config

**Files:**
- Create: `.gitignore`
- Create: `.env.example`
- Create: `go.mod`
- Create: `cmd/hashminer/main.go`
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Add repository ignores**

Create `.gitignore`:

```gitignore
.env
hashminer
*.o
*.a
*.so
*.dylib
*.test
.DS_Store
```

- [ ] **Step 2: Add example environment**

Create `.env.example`:

```env
RPC_URL=https://ethereum-rpc.publicnode.com
PRIVATE_KEY=0xreplace_with_private_key
CONTRACT_ADDRESS=0xAC7b5d06fa1e77D08aea40d46cB7C5923A87A0cc
CHAIN_ID=1
CUDA_BLOCKS=4096
CUDA_THREADS=256
NONCES_PER_BATCH=1048576
GAS_LIMIT=160000
REFRESH_INTERVAL_MS=2500
DRY_RUN=true
```

- [ ] **Step 3: Add Go module**

Create `go.mod`:

```go
module hashminer

go 1.22

require (
	github.com/charmbracelet/bubbles v0.20.0
	github.com/charmbracelet/bubbletea v1.2.4
	github.com/charmbracelet/lipgloss v1.0.0
	github.com/ethereum/go-ethereum v1.14.13
	github.com/joho/godotenv v1.5.1
)
```

- [ ] **Step 4: Add config tests first**

Create `internal/config/config_test.go`:

```go
package config

import "testing"

func TestLoadFromMap(t *testing.T) {
	env := map[string]string{
		"RPC_URL":             "https://example.invalid",
		"PRIVATE_KEY":         "0xabc",
		"CONTRACT_ADDRESS":    "0xAC7b5d06fa1e77D08aea40d46cB7C5923A87A0cc",
		"CHAIN_ID":            "1",
		"CUDA_BLOCKS":         "4096",
		"CUDA_THREADS":        "256",
		"NONCES_PER_BATCH":    "1048576",
		"GAS_LIMIT":           "160000",
		"REFRESH_INTERVAL_MS": "2500",
		"DRY_RUN":             "true",
	}

	cfg, err := LoadFromMap(env)
	if err != nil {
		t.Fatalf("LoadFromMap returned error: %v", err)
	}
	if cfg.RPCURL != "https://example.invalid" {
		t.Fatalf("RPCURL = %q", cfg.RPCURL)
	}
	if cfg.ChainID != 1 {
		t.Fatalf("ChainID = %d", cfg.ChainID)
	}
	if !cfg.DryRun {
		t.Fatal("DryRun = false")
	}
}

func TestRedactedMapHidesPrivateKey(t *testing.T) {
	cfg := Config{PrivateKey: "0xsupersecret", RPCURL: "https://rpc"}
	redacted := cfg.RedactedMap()
	if redacted["PRIVATE_KEY"] == cfg.PrivateKey {
		t.Fatal("private key was not redacted")
	}
	if redacted["PRIVATE_KEY"] != "<redacted>" {
		t.Fatalf("PRIVATE_KEY = %q", redacted["PRIVATE_KEY"])
	}
}
```

- [ ] **Step 5: Implement config**

Create `internal/config/config.go`:

```go
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	RPCURL              string
	PrivateKey          string
	ContractAddress     string
	ChainID             int64
	CUDABlocks          int
	CUDAThreads         int
	NoncesPerBatch      uint64
	GasLimit            uint64
	MaxFeePerGas         string
	MaxPriorityFeePerGas string
	RefreshIntervalMS   int
	DryRun              bool
}

func Load() (Config, error) {
	_ = godotenv.Load()
	env := map[string]string{}
	for _, key := range []string{
		"RPC_URL", "PRIVATE_KEY", "CONTRACT_ADDRESS", "CHAIN_ID",
		"CUDA_BLOCKS", "CUDA_THREADS", "NONCES_PER_BATCH", "GAS_LIMIT",
		"MAX_FEE_PER_GAS", "MAX_PRIORITY_FEE_PER_GAS", "REFRESH_INTERVAL_MS", "DRY_RUN",
	} {
		env[key] = os.Getenv(key)
	}
	return LoadFromMap(env)
}

func LoadFromMap(env map[string]string) (Config, error) {
	cfg := Config{
		RPCURL:              env["RPC_URL"],
		PrivateKey:          env["PRIVATE_KEY"],
		ContractAddress:     env["CONTRACT_ADDRESS"],
		ChainID:             int64Value(env["CHAIN_ID"], 1),
		CUDABlocks:          intValue(env["CUDA_BLOCKS"], 4096),
		CUDAThreads:         intValue(env["CUDA_THREADS"], 256),
		NoncesPerBatch:      uint64Value(env["NONCES_PER_BATCH"], 1048576),
		GasLimit:            uint64Value(env["GAS_LIMIT"], 160000),
		MaxFeePerGas:         env["MAX_FEE_PER_GAS"],
		MaxPriorityFeePerGas: env["MAX_PRIORITY_FEE_PER_GAS"],
		RefreshIntervalMS:   intValue(env["REFRESH_INTERVAL_MS"], 2500),
		DryRun:              boolValue(env["DRY_RUN"], true),
	}

	if cfg.RPCURL == "" {
		return Config{}, errors.New("RPC_URL is required")
	}
	if cfg.PrivateKey == "" {
		return Config{}, errors.New("PRIVATE_KEY is required")
	}
	if cfg.ContractAddress == "" {
		return Config{}, errors.New("CONTRACT_ADDRESS is required")
	}
	if cfg.CUDAThreads <= 0 || cfg.CUDABlocks <= 0 || cfg.NoncesPerBatch == 0 {
		return Config{}, fmt.Errorf("invalid CUDA settings")
	}
	return cfg, nil
}

func (c Config) RedactedMap() map[string]string {
	return map[string]string{
		"RPC_URL":                  c.RPCURL,
		"PRIVATE_KEY":              "<redacted>",
		"CONTRACT_ADDRESS":         c.ContractAddress,
		"CHAIN_ID":                 strconv.FormatInt(c.ChainID, 10),
		"CUDA_BLOCKS":              strconv.Itoa(c.CUDABlocks),
		"CUDA_THREADS":             strconv.Itoa(c.CUDAThreads),
		"NONCES_PER_BATCH":         strconv.FormatUint(c.NoncesPerBatch, 10),
		"GAS_LIMIT":                strconv.FormatUint(c.GasLimit, 10),
		"MAX_FEE_PER_GAS":          c.MaxFeePerGas,
		"MAX_PRIORITY_FEE_PER_GAS": c.MaxPriorityFeePerGas,
		"REFRESH_INTERVAL_MS":      strconv.Itoa(c.RefreshIntervalMS),
		"DRY_RUN":                  strconv.FormatBool(c.DryRun),
	}
}

func intValue(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return n
}

func int64Value(s string, fallback int64) int64 {
	if s == "" {
		return fallback
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}

func uint64Value(s string, fallback uint64) uint64 {
	if s == "" {
		return fallback
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}

func boolValue(s string, fallback bool) bool {
	if s == "" {
		return fallback
	}
	n, err := strconv.ParseBool(s)
	if err != nil {
		return fallback
	}
	return n
}
```

- [ ] **Step 6: Add temporary CLI entrypoint**

Create `cmd/hashminer/main.go`:

```go
package main

import (
	"fmt"
	"os"

	"hashminer/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("hashminer configured for %s\n", cfg.ContractAddress)
}
```

- [ ] **Step 7: Verify on Vast, not local**

On the Vast instance only, run:

```bash
go mod tidy
go test ./internal/config
git add .gitignore .env.example go.mod go.sum cmd/hashminer/main.go internal/config
git commit -m "feat: add config loader"
```

Expected: tests pass and commit succeeds.

## Task 2: ABI Encoding And Difficulty Helpers

**Files:**
- Create: `internal/ethabi/mine.go`
- Create: `internal/ethabi/mine_test.go`
- Create: `internal/miner/difficulty.go`
- Create: `internal/miner/difficulty_test.go`

- [ ] **Step 1: Add ABI tests**

Create `internal/ethabi/mine_test.go`:

```go
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
```

- [ ] **Step 2: Implement ABI helper**

Create `internal/ethabi/mine.go`:

```go
package ethabi

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var mineABI = abi.MustNewType("uint256", "", nil)

func MineCalldata(nonce *big.Int) ([]byte, error) {
	args := abi.Arguments{{Type: mineABI}}
	encoded, err := args.Pack(nonce)
	if err != nil {
		return nil, err
	}
	return append([]byte{0x1d, 0x5a, 0x4e, 0xb6}, encoded...), nil
}
```

- [ ] **Step 3: Add difficulty tests**

Create `internal/miner/difficulty_test.go`:

```go
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
```

- [ ] **Step 4: Implement difficulty helper**

Create `internal/miner/difficulty.go`:

```go
package miner

import "math/big"

func MeetsDifficulty(result *big.Int, difficulty *big.Int) bool {
	return result.Cmp(difficulty) < 0
}
```

- [ ] **Step 5: Verify on Vast, not local**

On the Vast instance only, run:

```bash
go test ./internal/ethabi ./internal/miner
git add internal/ethabi internal/miner/difficulty.go internal/miner/difficulty_test.go
git commit -m "feat: add mine calldata and difficulty helpers"
```

Expected: tests pass and commit succeeds.

## Task 3: Chain And Wallet Clients

**Files:**
- Create: `internal/chain/client.go`
- Create: `internal/wallet/wallet.go`

- [ ] **Step 1: Implement chain client**

Create `internal/chain/client.go`:

```go
package chain

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	selGetChallenge = common.FromHex("0x2a21bc8b")
	selMiningState  = common.FromHex("0x4e8f2b45")
)

type MiningState struct {
	Era             *big.Int
	Reward          *big.Int
	Difficulty      *big.Int
	Minted          *big.Int
	Remaining       *big.Int
	Epoch           *big.Int
	EpochBlocksLeft *big.Int
}

type Client struct {
	rpc      *ethclient.Client
	contract common.Address
}

func Dial(ctx context.Context, rpcURL string, contract common.Address) (*Client, error) {
	rpc, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, err
	}
	return &Client{rpc: rpc, contract: contract}, nil
}

func (c *Client) Close() {
	c.rpc.Close()
}

func (c *Client) BlockNumber(ctx context.Context) (uint64, error) {
	return c.rpc.BlockNumber(ctx)
}

func (c *Client) BalanceAt(ctx context.Context, addr common.Address) (*big.Int, error) {
	return c.rpc.BalanceAt(ctx, addr, nil)
}

func (c *Client) Challenge(ctx context.Context, miner common.Address) ([32]byte, error) {
	input := append([]byte{}, selGetChallenge...)
	input = append(input, common.LeftPadBytes(miner.Bytes(), 32)...)
	out, err := c.rpc.CallContract(ctx, ethereum.CallMsg{To: &c.contract, Data: input}, nil)
	if err != nil {
		return [32]byte{}, err
	}
	var challenge [32]byte
	copy(challenge[:], out[:32])
	return challenge, nil
}

func (c *Client) MiningState(ctx context.Context) (MiningState, error) {
	out, err := c.rpc.CallContract(ctx, ethereum.CallMsg{To: &c.contract, Data: selMiningState}, nil)
	if err != nil {
		return MiningState{}, err
	}
	return MiningState{
		Era:             word(out, 0),
		Reward:          word(out, 1),
		Difficulty:      word(out, 2),
		Minted:          word(out, 3),
		Remaining:       word(out, 4),
		Epoch:           word(out, 5),
		EpochBlocksLeft: word(out, 6),
	}, nil
}

func word(out []byte, index int) *big.Int {
	start := index * 32
	end := start + 32
	if len(out) < end {
		return new(big.Int)
	}
	return new(big.Int).SetBytes(out[start:end])
}
```

- [ ] **Step 2: Implement wallet**

Create `internal/wallet/wallet.go`:

```go
package wallet

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"strings"

	"hashminer/internal/ethabi"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Wallet struct {
	key     *ecdsa.PrivateKey
	address common.Address
	chainID *big.Int
}

func FromPrivateKey(hexKey string, chainID int64) (*Wallet, error) {
	trimmed := strings.TrimPrefix(hexKey, "0x")
	key, err := crypto.HexToECDSA(trimmed)
	if err != nil {
		return nil, err
	}
	return &Wallet{
		key:     key,
		address: crypto.PubkeyToAddress(key.PublicKey),
		chainID: big.NewInt(chainID),
	}, nil
}

func (w *Wallet) Address() common.Address {
	return w.address
}

func (w *Wallet) SignMineTx(ctx context.Context, rpc *ethclient.Client, contract common.Address, nonceValue *big.Int, gasLimit uint64) (*types.Transaction, error) {
	accountNonce, err := rpc.PendingNonceAt(ctx, w.address)
	if err != nil {
		return nil, err
	}
	gasTipCap, err := rpc.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, err
	}
	header, err := rpc.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}
	gasFeeCap := new(big.Int).Mul(header.BaseFee, big.NewInt(2))
	gasFeeCap.Add(gasFeeCap, gasTipCap)
	data, err := ethabi.MineCalldata(nonceValue)
	if err != nil {
		return nil, err
	}
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   w.chainID,
		Nonce:     accountNonce,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       gasLimit,
		To:        &contract,
		Value:     big.NewInt(0),
		Data:      data,
	})
	return types.SignTx(tx, types.LatestSignerForChainID(w.chainID), w.key)
}
```

- [ ] **Step 3: Verify on Vast, not local**

On the Vast instance only, run:

```bash
go test ./internal/chain ./internal/wallet
git add internal/chain internal/wallet
git commit -m "feat: add chain and wallet clients"
```

Expected: packages compile and commit succeeds.

## Task 4: CUDA Wrapper And Kernel Stub

**Files:**
- Create: `internal/cuda/cuda.go`
- Create: `cuda/hash_miner.h`
- Create: `cuda/hash_miner.cu`

- [ ] **Step 1: Add C header**

Create `cuda/hash_miner.h`:

```c
#pragma once

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct HashMinerResult {
  int found;
  uint64_t nonce;
  uint64_t hashes_done;
} HashMinerResult;

HashMinerResult hashminer_search(
  const uint8_t challenge[32],
  const uint8_t difficulty[32],
  uint64_t start_nonce,
  uint64_t count,
  int blocks,
  int threads
);

#ifdef __cplusplus
}
#endif
```

- [ ] **Step 2: Add CUDA implementation shell**

Create `cuda/hash_miner.cu`:

```cuda
#include "hash_miner.h"
#include <cuda_runtime.h>

__global__ void search_kernel(
  const uint8_t* challenge,
  const uint8_t* difficulty,
  uint64_t start_nonce,
  uint64_t count,
  uint64_t* found_nonce,
  int* found
) {
  uint64_t idx = blockIdx.x * blockDim.x + threadIdx.x;
  uint64_t stride = gridDim.x * blockDim.x;

  for (uint64_t i = idx; i < count && atomicAdd(found, 0) == 0; i += stride) {
    uint64_t nonce = start_nonce + i;

    // First implementation checkpoint: compile and exercise GPU dispatch.
    // The next task replaces this placeholder condition with Ethereum Keccak.
    if (nonce == 0xffffffffffffffffULL && challenge[0] == difficulty[0]) {
      *found_nonce = nonce;
      atomicExch(found, 1);
      return;
    }
  }
}

extern "C" HashMinerResult hashminer_search(
  const uint8_t challenge[32],
  const uint8_t difficulty[32],
  uint64_t start_nonce,
  uint64_t count,
  int blocks,
  int threads
) {
  uint8_t* d_challenge = nullptr;
  uint8_t* d_difficulty = nullptr;
  uint64_t* d_found_nonce = nullptr;
  int* d_found = nullptr;
  uint64_t h_found_nonce = 0;
  int h_found = 0;

  cudaMalloc(&d_challenge, 32);
  cudaMalloc(&d_difficulty, 32);
  cudaMalloc(&d_found_nonce, sizeof(uint64_t));
  cudaMalloc(&d_found, sizeof(int));
  cudaMemcpy(d_challenge, challenge, 32, cudaMemcpyHostToDevice);
  cudaMemcpy(d_difficulty, difficulty, 32, cudaMemcpyHostToDevice);
  cudaMemcpy(d_found_nonce, &h_found_nonce, sizeof(uint64_t), cudaMemcpyHostToDevice);
  cudaMemcpy(d_found, &h_found, sizeof(int), cudaMemcpyHostToDevice);

  search_kernel<<<blocks, threads>>>(d_challenge, d_difficulty, start_nonce, count, d_found_nonce, d_found);
  cudaDeviceSynchronize();

  cudaMemcpy(&h_found, d_found, sizeof(int), cudaMemcpyDeviceToHost);
  cudaMemcpy(&h_found_nonce, d_found_nonce, sizeof(uint64_t), cudaMemcpyDeviceToHost);

  cudaFree(d_challenge);
  cudaFree(d_difficulty);
  cudaFree(d_found_nonce);
  cudaFree(d_found);

  HashMinerResult result;
  result.found = h_found;
  result.nonce = h_found_nonce;
  result.hashes_done = count;
  return result;
}
```

- [ ] **Step 3: Add Go cgo wrapper**

Create `internal/cuda/cuda.go`:

```go
package cuda

/*
#cgo CFLAGS: -I${SRCDIR}/../../cuda
#cgo LDFLAGS: ${SRCDIR}/../../cuda/hash_miner.o -lcudart -L/usr/local/cuda/lib64
#include "hash_miner.h"
*/
import "C"
import (
	"encoding/binary"
	"math/big"
	"unsafe"
)

type Result struct {
	Found      bool
	Nonce      uint64
	HashesDone uint64
}

func Search(challenge [32]byte, difficulty *big.Int, startNonce uint64, count uint64, blocks int, threads int) Result {
	diff := difficulty.FillBytes(make([]byte, 32))
	res := C.hashminer_search(
		(*C.uint8_t)(unsafe.Pointer(&challenge[0])),
		(*C.uint8_t)(unsafe.Pointer(&diff[0])),
		C.uint64_t(startNonce),
		C.uint64_t(count),
		C.int(blocks),
		C.int(threads),
	)
	return Result{
		Found:      res.found != 0,
		Nonce:      uint64(res.nonce),
		HashesDone: uint64(res.hashes_done),
	}
}

func EncodeNonceABIWord(nonce uint64) [32]byte {
	var out [32]byte
	binary.BigEndian.PutUint64(out[24:], nonce)
	return out
}
```

- [ ] **Step 4: Verify on Vast, not local**

On the Vast instance only, run:

```bash
nvcc -O3 -c cuda/hash_miner.cu -o cuda/hash_miner.o
go test ./internal/cuda
git add internal/cuda cuda
git commit -m "feat: add cuda wrapper"
```

Expected: CUDA object builds, Go package compiles, commit succeeds.

## Task 5: Miner Engine And Events

**Files:**
- Create: `internal/miner/types.go`
- Create: `internal/miner/engine.go`

- [ ] **Step 1: Add miner types**

Create `internal/miner/types.go`:

```go
package miner

import (
	"math/big"
	"time"
)

type Snapshot struct {
	Running         bool
	Hashrate        float64
	TotalHashes     uint64
	Accepted        uint64
	Rejected        uint64
	Epoch           string
	EpochBlocksLeft string
	Reward          string
	Difficulty      string
	Minted          string
	Remaining       string
	LastNonceStart  uint64
	LastNonceEnd    uint64
	LastTX          string
	Wallet          string
	BalanceWei      string
	GPU             string
}

type Event struct {
	At      time.Time
	Level   string
	Message string
}

type ChainReader interface {
	MiningState(ctx context.Context) (ChainMiningState, error)
	Challenge(ctx context.Context, miner common.Address) ([32]byte, error)
	BlockNumber(ctx context.Context) (uint64, error)
}

type ChainMiningState struct {
	Era             *big.Int
	Reward          *big.Int
	Difficulty      *big.Int
	Minted          *big.Int
	Remaining       *big.Int
	Epoch           *big.Int
	EpochBlocksLeft *big.Int
}
```

Manual correction while implementing: add imports `context` and `github.com/ethereum/go-ethereum/common` to this file.

- [ ] **Step 2: Add engine**

Create `internal/miner/engine.go`:

```go
package miner

import (
	"context"
	"math/big"
	"sync"
	"time"

	"hashminer/internal/config"
	"hashminer/internal/cuda"

	"github.com/ethereum/go-ethereum/common"
)

type Engine struct {
	cfg      config.Config
	wallet   common.Address
	events   chan Event
	updates  chan Snapshot
	mu       sync.RWMutex
	running  bool
	snapshot Snapshot
}

func NewEngine(cfg config.Config, wallet common.Address) *Engine {
	return &Engine{
		cfg:     cfg,
		wallet:  wallet,
		events:  make(chan Event, 512),
		updates: make(chan Snapshot, 64),
		running: true,
		snapshot: Snapshot{
			Running: true,
			Wallet:  wallet.Hex(),
			GPU:     "NVIDIA RTX 4090",
		},
	}
}

func (e *Engine) Events() <-chan Event {
	return e.events
}

func (e *Engine) Updates() <-chan Snapshot {
	return e.updates
}

func (e *Engine) TogglePause() {
	e.mu.Lock()
	e.running = !e.running
	e.snapshot.Running = e.running
	snap := e.snapshot
	e.mu.Unlock()
	e.publish(snap)
}

func (e *Engine) Run(ctx context.Context, challenge [32]byte, difficulty *big.Int) {
	ticker := time.NewTicker(time.Millisecond * time.Duration(e.cfg.RefreshIntervalMS))
	defer ticker.Stop()

	var start uint64
	var total uint64
	last := time.Now()

	for {
		select {
		case <-ctx.Done():
			e.log("info", "miner stopped")
			return
		default:
		}

		e.mu.RLock()
		running := e.running
		e.mu.RUnlock()
		if !running {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				continue
			}
		}

		res := cuda.Search(challenge, difficulty, start, e.cfg.NoncesPerBatch, e.cfg.CUDABlocks, e.cfg.CUDAThreads)
		total += res.HashesDone
		elapsed := time.Since(last).Seconds()
		if elapsed <= 0 {
			elapsed = 1
		}
		hashrate := float64(total) / elapsed

		e.mu.Lock()
		e.snapshot.Hashrate = hashrate
		e.snapshot.TotalHashes = total
		e.snapshot.LastNonceStart = start
		e.snapshot.LastNonceEnd = start + e.cfg.NoncesPerBatch - 1
		if res.Found {
			e.snapshot.LastTX = "dry-run found nonce"
			e.snapshot.Accepted++
		}
		snap := e.snapshot
		e.mu.Unlock()
		e.publish(snap)

		if res.Found {
			e.log("info", "found candidate nonce")
		}
		start += e.cfg.NoncesPerBatch
	}
}

func (e *Engine) publish(snapshot Snapshot) {
	select {
	case e.updates <- snapshot:
	default:
	}
}

func (e *Engine) log(level string, message string) {
	select {
	case e.events <- Event{At: time.Now(), Level: level, Message: message}:
	default:
	}
}
```

- [ ] **Step 3: Verify on Vast, not local**

On the Vast instance only, run:

```bash
go test ./internal/miner
git add internal/miner
git commit -m "feat: add miner engine"
```

Expected: package compiles and commit succeeds.

## Task 6: Bubble Tea TUI

**Files:**
- Create: `internal/tui/model.go`
- Create: `internal/tui/view.go`
- Create: `internal/tui/keys.go`
- Modify: `cmd/hashminer/main.go`

- [ ] **Step 1: Add key handling**

Create `internal/tui/keys.go`:

```go
package tui

type Action int

const (
	ActionNone Action = iota
	ActionQuit
	ActionTogglePause
	ActionRefresh
	ActionHelp
)
```

- [ ] **Step 2: Add TUI model**

Create `internal/tui/model.go`:

```go
package tui

import (
	"fmt"

	"hashminer/internal/miner"

	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	tabs     []string
	active   int
	snapshot miner.Snapshot
	logs     []miner.Event
	actions  chan Action
	updates  <-chan miner.Snapshot
	events   <-chan miner.Event
	help     bool
}

func New(updates <-chan miner.Snapshot, events <-chan miner.Event, actions chan Action) Model {
	return Model{
		tabs:    []string{"Dashboard", "Mining", "Chain", "Transactions", "Logs", "Settings"},
		actions: actions,
		updates: updates,
		events:  events,
	}
}

type snapshotMsg miner.Snapshot
type eventMsg miner.Event

func (m Model) Init() tea.Cmd {
	return tea.Batch(waitSnapshot(m.updates), waitEvent(m.events))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.KeyMsg:
		switch v.String() {
		case "ctrl+c", "q":
			m.actions <- ActionQuit
			return m, tea.Quit
		case "right", "l":
			m.active = (m.active + 1) % len(m.tabs)
		case "left", "h":
			m.active--
			if m.active < 0 {
				m.active = len(m.tabs) - 1
			}
		case "p":
			m.actions <- ActionTogglePause
		case "r":
			m.actions <- ActionRefresh
		case "?":
			m.help = !m.help
		}
	case snapshotMsg:
		m.snapshot = miner.Snapshot(v)
		return m, waitSnapshot(m.updates)
	case eventMsg:
		m.logs = append(m.logs, miner.Event(v))
		if len(m.logs) > 200 {
			m.logs = m.logs[len(m.logs)-200:]
		}
		return m, waitEvent(m.events)
	}
	return m, nil
}

func waitSnapshot(ch <-chan miner.Snapshot) tea.Cmd {
	return func() tea.Msg {
		v := <-ch
		return snapshotMsg(v)
	}
}

func waitEvent(ch <-chan miner.Event) tea.Cmd {
	return func() tea.Msg {
		v := <-ch
		return eventMsg(v)
	}
}

func (m Model) title() string {
	return fmt.Sprintf("%s (%d/%d)", m.tabs[m.active], m.active+1, len(m.tabs))
}
```

- [ ] **Step 3: Add views**

Create `internal/tui/view.go`:

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	tabStyle    = lipgloss.NewStyle().Padding(0, 1)
	activeTab   = tabStyle.Copy().Bold(true).Foreground(lipgloss.Color("10"))
	panel       = lipgloss.NewStyle().Padding(1, 2).Border(lipgloss.RoundedBorder())
	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func (m Model) View() string {
	var tabs []string
	for i, t := range m.tabs {
		if i == m.active {
			tabs = append(tabs, activeTab.Render(t))
		} else {
			tabs = append(tabs, tabStyle.Render(t))
		}
	}
	body := m.page()
	help := "h/l or arrows: tabs | p: pause | r: refresh | ?: help | q: quit"
	if m.help {
		help = "HASH256 GPU miner TUI. Settings are read from .env and PRIVATE_KEY is always redacted."
	}
	return strings.Join(tabs, "") + "\n" + panel.Render(body) + "\n" + statusStyle.Render(help)
}

func (m Model) page() string {
	s := m.snapshot
	switch m.tabs[m.active] {
	case "Dashboard":
		return fmt.Sprintf("Running: %v\nHashrate: %.2f H/s\nAccepted: %d\nRejected: %d\nWallet: %s\nGPU: %s", s.Running, s.Hashrate, s.Accepted, s.Rejected, s.Wallet, s.GPU)
	case "Mining":
		return fmt.Sprintf("Nonce range: %d - %d\nTotal hashes: %d\nLast tx: %s", s.LastNonceStart, s.LastNonceEnd, s.TotalHashes, s.LastTX)
	case "Chain":
		return fmt.Sprintf("Epoch: %s\nEpoch blocks left: %s\nReward: %s\nDifficulty: %s\nMinted: %s\nRemaining: %s", s.Epoch, s.EpochBlocksLeft, s.Reward, s.Difficulty, s.Minted, s.Remaining)
	case "Transactions":
		return fmt.Sprintf("Last transaction: %s", s.LastTX)
	case "Logs":
		start := 0
		if len(m.logs) > 20 {
			start = len(m.logs) - 20
		}
		lines := []string{}
		for _, entry := range m.logs[start:] {
			lines = append(lines, fmt.Sprintf("%s [%s] %s", entry.At.Format("15:04:05"), entry.Level, entry.Message))
		}
		return strings.Join(lines, "\n")
	case "Settings":
		return "Config loaded from .env\nPRIVATE_KEY=<redacted>"
	default:
		return ""
	}
}
```

- [ ] **Step 4: Wire CLI to TUI**

Replace `cmd/hashminer/main.go` with:

```go
package main

import (
	"context"
	"fmt"
	"math/big"
	"os"

	"hashminer/internal/config"
	"hashminer/internal/miner"
	"hashminer/internal/tui"
	"hashminer/internal/wallet"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ethereum/go-ethereum/common"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	w, err := wallet.FromPrivateKey(cfg.PrivateKey, cfg.ChainID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine := miner.NewEngine(cfg, w.Address())
	actions := make(chan tui.Action, 16)

	go func() {
		for action := range actions {
			switch action {
			case tui.ActionQuit:
				cancel()
				return
			case tui.ActionTogglePause:
				engine.TogglePause()
			case tui.ActionRefresh:
			}
		}
	}()

	// Placeholder challenge/difficulty until chain polling is enabled in Task 7.
	var challenge [32]byte
	difficulty := new(big.Int).Lsh(big.NewInt(1), 255)
	_ = common.HexToAddress(cfg.ContractAddress)
	go engine.Run(ctx, challenge, difficulty)

	if _, err := tea.NewProgram(tui.New(engine.Updates(), engine.Events(), actions), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 5: Verify on Vast, not local**

On the Vast instance only, run:

```bash
go test ./...
go build -o hashminer ./cmd/hashminer
git add internal/tui cmd/hashminer/main.go
git commit -m "feat: add tui dashboard"
```

Expected: tests pass, binary builds, commit succeeds.

## Task 7: Real Chain Polling And Transaction Submission

**Files:**
- Modify: `internal/chain/client.go`
- Modify: `internal/wallet/wallet.go`
- Modify: `internal/miner/engine.go`
- Modify: `cmd/hashminer/main.go`

- [ ] **Step 1: Expose raw ethclient and send transaction**

Add to `internal/chain/client.go`:

```go
func (c *Client) RPC() *ethclient.Client {
	return c.rpc
}
```

Add to `internal/wallet/wallet.go`:

```go
func (w *Wallet) SendMineTx(ctx context.Context, rpc *ethclient.Client, contract common.Address, nonceValue *big.Int, gasLimit uint64) (*types.Transaction, error) {
	tx, err := w.SignMineTx(ctx, rpc, contract, nonceValue, gasLimit)
	if err != nil {
		return nil, err
	}
	if err := rpc.SendTransaction(ctx, tx); err != nil {
		return nil, err
	}
	return tx, nil
}
```

- [ ] **Step 2: Update engine to submit when dry-run is false**

Extend `Engine` with a submit callback:

```go
type SubmitFunc func(ctx context.Context, nonce *big.Int) (string, error)

type Engine struct {
	cfg      config.Config
	wallet   common.Address
	submit   SubmitFunc
	events   chan Event
	updates  chan Snapshot
	mu       sync.RWMutex
	running  bool
	snapshot Snapshot
}
```

Change constructor signature:

```go
func NewEngine(cfg config.Config, wallet common.Address, submit SubmitFunc) *Engine
```

Inside `Run`, replace found nonce handling with:

```go
if res.Found {
	nonce := new(big.Int).SetUint64(res.Nonce)
	if e.cfg.DryRun || e.submit == nil {
		e.log("info", "dry-run found candidate nonce")
		e.snapshot.Accepted++
	} else {
		txHash, err := e.submit(ctx, nonce)
		if err != nil {
			e.snapshot.Rejected++
			e.log("error", err.Error())
		} else {
			e.snapshot.Accepted++
			e.snapshot.LastTX = txHash
			e.log("info", "submitted mine transaction "+txHash)
		}
	}
}
```

- [ ] **Step 3: Wire chain client in main**

In `cmd/hashminer/main.go`, dial the chain client before creating engine:

```go
contract := common.HexToAddress(cfg.ContractAddress)
chainClient, err := chain.Dial(ctx, cfg.RPCURL, contract)
if err != nil {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
defer chainClient.Close()

state, err := chainClient.MiningState(ctx)
if err != nil {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
challenge, err := chainClient.Challenge(ctx, w.Address())
if err != nil {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

submit := func(ctx context.Context, nonce *big.Int) (string, error) {
	tx, err := w.SendMineTx(ctx, chainClient.RPC(), contract, nonce, cfg.GasLimit)
	if err != nil {
		return "", err
	}
	return tx.Hash().Hex(), nil
}

engine := miner.NewEngine(cfg, w.Address(), submit)
go engine.Run(ctx, challenge, state.Difficulty)
```

Add import:

```go
"hashminer/internal/chain"
```

- [ ] **Step 4: Verify on Vast, not local**

On the Vast instance only, run:

```bash
go test ./...
go build -o hashminer ./cmd/hashminer
git add internal/chain internal/wallet internal/miner cmd/hashminer/main.go
git commit -m "feat: wire chain polling and transaction submission"
```

Expected: tests pass, binary builds, commit succeeds.

## Task 8: Replace CUDA Placeholder With Keccak

**Files:**
- Modify: `cuda/hash_miner.cu`
- Modify: `internal/cuda/cuda.go`
- Create: `internal/cuda/encode_test.go`

- [ ] **Step 1: Add ABI nonce encoding test**

Create `internal/cuda/encode_test.go`:

```go
package cuda

import "testing"

func TestEncodeNonceABIWord(t *testing.T) {
	got := EncodeNonceABIWord(1)
	if got[31] != 1 {
		t.Fatalf("last byte = %d", got[31])
	}
	for i := 0; i < 31; i++ {
		if got[i] != 0 {
			t.Fatalf("byte %d = %d", i, got[i])
		}
	}
}
```

- [ ] **Step 2: Implement Ethereum Keccak in CUDA**

Replace the placeholder condition in `cuda/hash_miner.cu` with a Keccak-256 device implementation:

```cuda
// Implement keccakf1600 round constants, rotation offsets, theta/rho/pi/chi/iota.
// Build a 64-byte message: challenge[0..31] then nonce ABI word[0..31].
// Apply Keccak padding 0x01 ... 0x80 for Ethereum Keccak.
// Compare the 32-byte digest as big-endian uint256 against difficulty.
```

Concrete source to use during implementation: port a compact public-domain Keccak-f[1600] implementation into `__device__` functions and keep it self-contained in `cuda/hash_miner.cu`.

- [ ] **Step 3: Verify on Vast, not local**

On the Vast instance only, run:

```bash
nvcc -O3 -c cuda/hash_miner.cu -o cuda/hash_miner.o
go test ./internal/cuda
go build -o hashminer ./cmd/hashminer
git add cuda/hash_miner.cu internal/cuda
git commit -m "feat: add cuda keccak mining kernel"
```

Expected: CUDA builds, Go tests pass, binary builds, commit succeeds.

## Task 9: README And Vast Runbook

**Files:**
- Create: `README.md`

- [ ] **Step 1: Add runbook**

Create `README.md`:

```markdown
# HASH256 GPU TUI Miner

Go + CUDA TUI miner for HASH256.

## Vast AI Setup

Use an NVIDIA CUDA development image with Go installed, or install Go in a CUDA image.

Check GPU:

```bash
nvidia-smi
nvcc --version
go version
```

Copy config:

```bash
cp .env.example .env
nano .env
```

Set:

```env
RPC_URL=your_rpc_url
PRIVATE_KEY=0xyour_private_key
CONTRACT_ADDRESS=0xAC7b5d06fa1e77D08aea40d46cB7C5923A87A0cc
DRY_RUN=true
```

Build:

```bash
go mod tidy
nvcc -O3 -c cuda/hash_miner.cu -o cuda/hash_miner.o
go build -o hashminer ./cmd/hashminer
```

Run dry mode first:

```bash
./hashminer
```

When stable, set:

```env
DRY_RUN=false
```

Then run again:

```bash
./hashminer
```

## Controls

- `h/l` or arrows: switch pages
- `p`: pause/resume
- `r`: refresh
- `?`: help
- `q`: quit

## Security

Do not commit `.env`. The app redacts `PRIVATE_KEY` in TUI and logs.
```

- [ ] **Step 2: Verify on Vast, not local**

On the Vast instance only, run:

```bash
git add README.md
git commit -m "docs: add vast runbook"
```

Expected: commit succeeds.

## Self-Review

- Spec coverage: config, wallet env, CUDA/cgo, TUI pages, dry-run, chain polling, transaction submission, and Vast verification are covered.
- Local execution rule: commands are documented for Vast only; this plan should not be executed on the local PC.
- Known implementation risk: Task 8 requires a complete CUDA Keccak implementation. The placeholder kernel in Task 4 is intentionally only a build bridge and must not be used for real mining.

package miner

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"hashminer/internal/config"

	"github.com/ethereum/go-ethereum/common"
)

type Engine struct {
	cfg      config.Config
	wallet   common.Address
	searcher Searcher
	refresh  RefreshFunc
	submit   SubmitFunc

	events  chan Event
	updates chan Snapshot

	mu       sync.RWMutex
	running  bool
	snapshot Snapshot
	work     WorkState
}

func NewEngine(cfg config.Config, wallet common.Address, searcher Searcher, refresh RefreshFunc, submit SubmitFunc) *Engine {
	now := time.Now()
	return &Engine{
		cfg:      cfg,
		wallet:   wallet,
		searcher: searcher,
		refresh:  refresh,
		submit:   submit,
		events:   make(chan Event, 512),
		updates:  make(chan Snapshot, 64),
		running:  true,
		snapshot: Snapshot{
			Running:   true,
			Wallet:    wallet.Hex(),
			GPU:       "NVIDIA RTX 4090",
			DryRun:    cfg.DryRun,
			StartedAt: now,
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
	if snap.Running {
		e.log("info", "mining resumed")
	} else {
		e.log("info", "mining paused")
	}
}

func (e *Engine) ForceRefresh(ctx context.Context) {
	if err := e.refreshWork(ctx); err != nil {
		e.log("error", err.Error())
	}
}

func (e *Engine) Run(ctx context.Context) {
	e.log("info", "miner starting")
	if err := e.refreshWork(ctx); err != nil {
		e.log("error", err.Error())
		return
	}

	var start uint64
	var total uint64
	started := time.Now()
	refreshTicker := time.NewTicker(time.Millisecond * time.Duration(e.cfg.RefreshIntervalMS))
	defer refreshTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			e.log("info", "miner stopped")
			return
		case <-refreshTicker.C:
			if err := e.refreshWork(ctx); err != nil {
				e.log("error", err.Error())
			}
		default:
		}

		e.mu.RLock()
		running := e.running
		work := e.work
		e.mu.RUnlock()

		if !running {
			select {
			case <-ctx.Done():
				return
			case <-time.After(250 * time.Millisecond):
				continue
			}
		}
		if work.Difficulty == nil || work.Difficulty.Sign() == 0 {
			time.Sleep(250 * time.Millisecond)
			continue
		}

		res, err := e.searcher.Search(work.Challenge, work.Difficulty, start, e.cfg.NoncesPerBatch)
		if err != nil {
			e.log("error", err.Error())
			time.Sleep(time.Second)
			continue
		}

		total += res.HashesDone
		elapsed := time.Since(started).Seconds()
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
			e.snapshot.LastCandidate = res.Nonce
			e.snapshot.LastSubmitAttempt = time.Now()
		}
		snap := e.snapshot
		e.mu.Unlock()
		e.publish(snap)

		if res.Found {
			e.handleCandidate(ctx, res.Nonce)
			if err := e.refreshWork(ctx); err != nil {
				e.log("error", err.Error())
			}
		}
		start += e.cfg.NoncesPerBatch
	}
}

func (e *Engine) handleCandidate(ctx context.Context, nonce uint64) {
	if e.cfg.DryRun || e.submit == nil {
		e.mu.Lock()
		e.snapshot.Accepted++
		e.snapshot.LastTX = "dry-run"
		snap := e.snapshot
		e.mu.Unlock()
		e.publish(snap)
		e.log("info", fmt.Sprintf("dry-run found candidate nonce %d", nonce))
		return
	}

	txHash, err := e.submit(ctx, new(big.Int).SetUint64(nonce))
	e.mu.Lock()
	if err != nil {
		e.snapshot.Rejected++
	} else {
		e.snapshot.Accepted++
		e.snapshot.LastTX = txHash
	}
	snap := e.snapshot
	e.mu.Unlock()
	e.publish(snap)

	if err != nil {
		e.log("error", err.Error())
		return
	}
	e.log("info", "submitted mine transaction "+txHash)
}

func (e *Engine) refreshWork(ctx context.Context) error {
	if e.refresh == nil {
		return nil
	}
	work, err := e.refresh(ctx)
	if err != nil {
		return err
	}
	e.mu.Lock()
	e.work = work
	e.snapshot.Epoch = intString(work.Epoch)
	e.snapshot.EpochBlocksLeft = intString(work.EpochBlocksLeft)
	e.snapshot.Reward = intString(work.Reward)
	e.snapshot.Difficulty = intString(work.Difficulty)
	e.snapshot.Minted = intString(work.Minted)
	e.snapshot.Remaining = intString(work.Remaining)
	e.snapshot.BlockNumber = fmt.Sprintf("%d", work.BlockNumber)
	e.snapshot.MintsInBlock = intString(work.MintsInBlock)
	e.snapshot.BalanceWei = intString(work.WalletBalanceWei)
	e.snapshot.LastStateRefresh = time.Now()
	snap := e.snapshot
	e.mu.Unlock()
	e.publish(snap)
	return nil
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

func intString(v *big.Int) string {
	if v == nil {
		return "0"
	}
	return v.String()
}


package main

import (
	"context"
	"fmt"
	"math/big"
	"os"

	"hashminer/internal/chain"
	"hashminer/internal/config"
	"hashminer/internal/cuda"
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

	contract := common.HexToAddress(cfg.ContractAddress)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chainClient, err := chain.Dial(ctx, cfg.RPCURL, contract)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer chainClient.Close()

	refresh := func(ctx context.Context) (miner.WorkState, error) {
		state, err := chainClient.MiningState(ctx)
		if err != nil {
			return miner.WorkState{}, err
		}
		challenge, err := chainClient.Challenge(ctx, w.Address())
		if err != nil {
			return miner.WorkState{}, err
		}
		blockNumber, err := chainClient.BlockNumber(ctx)
		if err != nil {
			return miner.WorkState{}, err
		}
		mintsInBlock, err := chainClient.MintsInBlock(ctx, blockNumber)
		if err != nil {
			return miner.WorkState{}, err
		}
		balance, err := chainClient.BalanceAt(ctx, w.Address())
		if err != nil {
			balance = new(big.Int)
		}
		return miner.WorkState{
			Challenge:        challenge,
			Difficulty:       state.Difficulty,
			Epoch:            state.Epoch,
			EpochBlocksLeft:  state.EpochBlocksLeft,
			Reward:           state.Reward,
			Minted:           state.Minted,
			Remaining:        state.Remaining,
			BlockNumber:      blockNumber,
			MintsInBlock:     mintsInBlock,
			WalletBalanceWei: balance,
		}, nil
	}

	submit := func(ctx context.Context, nonce *big.Int) (string, error) {
		tx, err := w.SendMineTx(ctx, chainClient.RPC(), contract, nonce, cfg.GasLimit)
		if err != nil {
			return "", err
		}
		return tx.Hash().Hex(), nil
	}

	searcher := cuda.NewSearcher(cfg.CUDABlocks, cfg.CUDAThreads)
	engine := miner.NewEngine(cfg, w.Address(), searcher, refresh, submit)
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
				engine.ForceRefresh(ctx)
			}
		}
	}()

	go engine.Run(ctx)

	model := tui.New(engine.Updates(), engine.Events(), actions, cfg.RedactedMap())
	if _, err := tea.NewProgram(model, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

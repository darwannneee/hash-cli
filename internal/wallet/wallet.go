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


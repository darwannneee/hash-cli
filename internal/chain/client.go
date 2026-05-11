package chain

import (
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
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
	abi      abi.ABI
}

func Dial(ctx context.Context, rpcURL string, contract common.Address) (*Client, error) {
	rpc, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, err
	}
	return &Client{rpc: rpc, contract: contract, abi: hashABI()}, nil
}

func (c *Client) Close() {
	c.rpc.Close()
}

func (c *Client) RPC() *ethclient.Client {
	return c.rpc
}

func (c *Client) Contract() common.Address {
	return c.contract
}

func (c *Client) BlockNumber(ctx context.Context) (uint64, error) {
	return c.rpc.BlockNumber(ctx)
}

func (c *Client) BalanceAt(ctx context.Context, addr common.Address) (*big.Int, error) {
	return c.rpc.BalanceAt(ctx, addr, nil)
}

func (c *Client) Challenge(ctx context.Context, miner common.Address) ([32]byte, error) {
	input, err := c.abi.Pack("getChallenge", miner)
	if err != nil {
		return [32]byte{}, err
	}
	out, err := c.rpc.CallContract(ctx, ethereum.CallMsg{To: &c.contract, Data: input}, nil)
	if err != nil {
		return [32]byte{}, err
	}
	values, err := c.abi.Unpack("getChallenge", out)
	if err != nil {
		return [32]byte{}, err
	}
	return values[0].([32]byte), nil
}

func (c *Client) MiningState(ctx context.Context) (MiningState, error) {
	input, err := c.abi.Pack("miningState")
	if err != nil {
		return MiningState{}, err
	}
	out, err := c.rpc.CallContract(ctx, ethereum.CallMsg{To: &c.contract, Data: input}, nil)
	if err != nil {
		return MiningState{}, err
	}
	values, err := c.abi.Unpack("miningState", out)
	if err != nil {
		return MiningState{}, err
	}
	return MiningState{
		Era:             values[0].(*big.Int),
		Reward:          values[1].(*big.Int),
		Difficulty:      values[2].(*big.Int),
		Minted:          values[3].(*big.Int),
		Remaining:       values[4].(*big.Int),
		Epoch:           values[5].(*big.Int),
		EpochBlocksLeft: values[6].(*big.Int),
	}, nil
}

func (c *Client) MintsInBlock(ctx context.Context, blockNumber uint64) (*big.Int, error) {
	input, err := c.abi.Pack("mintsInBlock", new(big.Int).SetUint64(blockNumber))
	if err != nil {
		return nil, err
	}
	out, err := c.rpc.CallContract(ctx, ethereum.CallMsg{To: &c.contract, Data: input}, nil)
	if err != nil {
		return nil, err
	}
	values, err := c.abi.Unpack("mintsInBlock", out)
	if err != nil {
		return nil, err
	}
	return values[0].(*big.Int), nil
}

func hashABI() abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(`[
		{"inputs":[{"internalType":"address","name":"miner","type":"address"}],"name":"getChallenge","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"stateMutability":"view","type":"function"},
		{"inputs":[],"name":"miningState","outputs":[{"internalType":"uint256","name":"era","type":"uint256"},{"internalType":"uint256","name":"reward","type":"uint256"},{"internalType":"uint256","name":"difficulty","type":"uint256"},{"internalType":"uint256","name":"minted","type":"uint256"},{"internalType":"uint256","name":"remaining","type":"uint256"},{"internalType":"uint256","name":"epoch","type":"uint256"},{"internalType":"uint256","name":"epochBlocksLeft_","type":"uint256"}],"stateMutability":"view","type":"function"},
		{"inputs":[{"internalType":"uint256","name":"","type":"uint256"}],"name":"mintsInBlock","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}
	]`))
	if err != nil {
		panic(err)
	}
	return parsed
}


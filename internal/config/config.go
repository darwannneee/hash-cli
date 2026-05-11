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
	for _, key := range envKeys {
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
	if cfg.CUDABlocks <= 0 || cfg.CUDAThreads <= 0 || cfg.NoncesPerBatch == 0 {
		return Config{}, fmt.Errorf("invalid CUDA settings")
	}
	if cfg.GasLimit == 0 {
		return Config{}, fmt.Errorf("GAS_LIMIT must be greater than zero")
	}
	if cfg.RefreshIntervalMS <= 0 {
		return Config{}, fmt.Errorf("REFRESH_INTERVAL_MS must be greater than zero")
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

var envKeys = []string{
	"RPC_URL",
	"PRIVATE_KEY",
	"CONTRACT_ADDRESS",
	"CHAIN_ID",
	"CUDA_BLOCKS",
	"CUDA_THREADS",
	"NONCES_PER_BATCH",
	"GAS_LIMIT",
	"MAX_FEE_PER_GAS",
	"MAX_PRIORITY_FEE_PER_GAS",
	"REFRESH_INTERVAL_MS",
	"DRY_RUN",
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


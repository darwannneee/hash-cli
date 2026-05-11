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


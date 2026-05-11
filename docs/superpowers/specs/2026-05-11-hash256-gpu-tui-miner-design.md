# HASH256 GPU TUI Miner Design

## Goal

Build a Go CLI miner for HASH256 that uses an NVIDIA RTX 4090 through a CUDA kernel and presents a terminal UI by default. The miner reads wallet credentials and runtime settings from environment variables or a `.env` file, never from hardcoded source values.

## Scope

The first version will:

- Mine HASH256 by searching nonces on GPU.
- Track HASH256 mining state from Ethereum RPC.
- Sign and submit `mine(uint256)` transactions with the configured wallet.
- Show a keyboard-driven TUI for status, logs, mining progress, and transactions.
- Support pause, resume, refresh, and quit from the TUI.

The first version will not:

- Manage multiple wallets.
- Auto-tune every CUDA parameter.
- Provide a web UI.
- Store private keys in source code.

## Architecture

The project will be a Go module with a small CUDA component:

- `cmd/hashminer`: CLI entrypoint.
- `internal/config`: loads `.env` and environment variables.
- `internal/chain`: Ethereum RPC reads and HASH256 contract calls.
- `internal/wallet`: private key parsing, signing, gas settings, and transaction submission.
- `internal/miner`: mining loop, epoch refresh, pause/resume, and event fan-out.
- `internal/cuda`: cgo wrapper for GPU mining calls.
- `internal/tui`: Bubble Tea terminal UI.
- `cuda/hash_miner.cu`: CUDA kernel for nonce search.

The mining engine runs independently from the TUI. The engine publishes state updates and log events over Go channels. The TUI consumes those events and renders pages without blocking mining work.

## Mining Algorithm

The miner mirrors the verified HASH256 contract:

- Challenge is fetched from `getChallenge(minerAddress)`.
- The epoch is `block.number / 100`, so the miner refreshes challenge when epoch changes.
- Candidate proof is `keccak256(abi.encode(challenge, nonce))`.
- A nonce is valid when `uint256(result) < currentDifficulty`.
- Valid nonces are submitted through `mine(uint256)`.

The CUDA kernel must implement Ethereum-compatible Keccak, not NIST SHA3. It hashes the ABI encoding of `(bytes32,uint256)`, which is 64 bytes: 32 bytes challenge plus 32 bytes big-endian nonce.

## TUI Design

The default command opens a Bubble Tea TUI with these pages:

- `Dashboard`: hashrate, accepted/rejected counts, epoch, difficulty, reward, wallet address, balance, and GPU status.
- `Mining`: CUDA worker status, batch size, nonce ranges, found nonce queue, and submit status.
- `Chain`: block number, epoch blocks left, total mining minted, remaining mining supply, and current block mint count.
- `Transactions`: submitted tx hash, nonce, reward, gas settings, and pending/success/fail status.
- `Logs`: realtime miner, RPC, CUDA, and transaction logs.
- `Settings`: read-only config from `.env` with secrets redacted, plus runtime pause/resume status.

Keyboard controls:

- Left/right arrows or `h`/`l`: switch pages.
- `p`: pause or resume mining.
- `r`: force refresh chain state.
- `q` or `ctrl+c`: quit.
- `?`: show help overlay.

## Configuration

The miner reads config from `.env` and inherited environment variables. Required values:

- `RPC_URL`
- `PRIVATE_KEY`
- `CONTRACT_ADDRESS`

Optional values:

- `CHAIN_ID`, default `1`
- `CUDA_BLOCKS`
- `CUDA_THREADS`
- `NONCES_PER_BATCH`
- `GAS_LIMIT`
- `MAX_FEE_PER_GAS`
- `MAX_PRIORITY_FEE_PER_GAS`
- `REFRESH_INTERVAL_MS`

The TUI must redact `PRIVATE_KEY` in all views and logs.

## Error Handling

Startup fails fast if required config is missing, the private key is invalid, CUDA cannot initialize, or the contract cannot be queried.

Runtime errors are shown in logs and status panels. Recoverable errors include temporary RPC failure, transaction replacement, transaction revert, block cap reached, stale epoch, and insufficient work. The miner should refresh chain state after these errors and continue unless the user quits.

## Testing And Verification

The first implementation should include:

- Unit tests for config parsing and secret redaction.
- Unit tests for ABI input construction for `(bytes32,uint256)`.
- Unit tests for difficulty comparison.
- A CPU reference Keccak path to validate CUDA kernel output for known vectors.
- A dry-run mode or submit-disabled path for testing without sending transactions.

Manual verification:

- Build with `go test ./...`.
- Build CUDA wrapper with `nvcc`.
- Run TUI with a `.env` file.
- Confirm pages switch, pause/resume works, and logs update.
- Confirm found nonces are either submitted or logged in dry-run mode.

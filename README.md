# HASH256 GPU TUI Miner

Go + CUDA terminal UI miner for HASH256.

This repo is designed to be built on an NVIDIA CUDA machine such as a Vast AI RTX 4090 instance. Do not put a private key in source code; use `.env`.

## Vast AI Setup

Use a CUDA development image, or install Go inside a CUDA image.

Check the machine:

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

Set at minimum:

```env
RPC_URL=your_rpc_url
PRIVATE_KEY=0xyour_private_key
CONTRACT_ADDRESS=0xAC7b5d06fa1e77D08aea40d46cB7C5923A87A0cc
DRY_RUN=true
```

Build on the Vast instance:

```bash
go mod tidy
nvcc -O3 -c cuda/hash_miner.cu -o cuda/hash_miner.o
go build -o hashminer ./cmd/hashminer
```

Run dry mode first:

```bash
./hashminer
```

When the TUI is stable and dry-run behavior looks right, set:

```env
DRY_RUN=false
```

Then run:

```bash
./hashminer
```

## TUI Controls

- `h/l` or arrows: switch pages
- `p`: pause/resume
- `r`: refresh chain state
- `?`: help
- `q`: quit

## Security

- `.env` is gitignored.
- `PRIVATE_KEY` is redacted in the TUI settings page.
- Start with `DRY_RUN=true` before allowing real transactions.

## Notes

The CUDA kernel hashes the same preimage the HASH256 contract checks:

```text
keccak256(abi.encode(challenge, nonce))
```

The ABI encoding is 64 bytes: 32 bytes challenge plus 32 bytes big-endian uint256 nonce.


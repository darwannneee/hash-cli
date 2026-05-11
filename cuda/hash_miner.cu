#include "hash_miner.h"

#include <cuda_runtime.h>
#include <stdint.h>

__device__ __constant__ uint64_t k_round_constants[24] = {
  0x0000000000000001ULL, 0x0000000000008082ULL, 0x800000000000808aULL,
  0x8000000080008000ULL, 0x000000000000808bULL, 0x0000000080000001ULL,
  0x8000000080008081ULL, 0x8000000000008009ULL, 0x000000000000008aULL,
  0x0000000000000088ULL, 0x0000000080008009ULL, 0x000000008000000aULL,
  0x000000008000808bULL, 0x800000000000008bULL, 0x8000000000008089ULL,
  0x8000000000008003ULL, 0x8000000000008002ULL, 0x8000000000000080ULL,
  0x000000000000800aULL, 0x800000008000000aULL, 0x8000000080008081ULL,
  0x8000000000008080ULL, 0x0000000080000001ULL, 0x8000000080008008ULL
};

__device__ __constant__ int k_rotc[24] = {
  1, 3, 6, 10, 15, 21, 28, 36, 45, 55, 2, 14,
  27, 41, 56, 8, 25, 43, 62, 18, 39, 61, 20, 44
};

__device__ __constant__ int k_piln[24] = {
  10, 7, 11, 17, 18, 3, 5, 16, 8, 21, 24, 4,
  15, 23, 19, 13, 12, 2, 20, 14, 22, 9, 6, 1
};

__device__ inline uint64_t rotl64(uint64_t x, int y) {
  return (x << y) | (x >> (64 - y));
}

__device__ void keccakf(uint64_t st[25]) {
  uint64_t bc[5];

  for (int round = 0; round < 24; round++) {
    for (int i = 0; i < 5; i++) {
      bc[i] = st[i] ^ st[i + 5] ^ st[i + 10] ^ st[i + 15] ^ st[i + 20];
    }

    for (int i = 0; i < 5; i++) {
      uint64_t t = bc[(i + 4) % 5] ^ rotl64(bc[(i + 1) % 5], 1);
      for (int j = 0; j < 25; j += 5) {
        st[j + i] ^= t;
      }
    }

    uint64_t t = st[1];
    for (int i = 0; i < 24; i++) {
      int j = k_piln[i];
      uint64_t tmp = st[j];
      st[j] = rotl64(t, k_rotc[i]);
      t = tmp;
    }

    for (int j = 0; j < 25; j += 5) {
      for (int i = 0; i < 5; i++) {
        bc[i] = st[j + i];
      }
      for (int i = 0; i < 5; i++) {
        st[j + i] ^= (~bc[(i + 1) % 5]) & bc[(i + 2) % 5];
      }
    }

    st[0] ^= k_round_constants[round];
  }
}

__device__ uint64_t load64_le(const uint8_t* p) {
  uint64_t v = 0;
  for (int i = 0; i < 8; i++) {
    v |= ((uint64_t)p[i]) << (8 * i);
  }
  return v;
}

__device__ void store64_le(uint8_t* p, uint64_t v) {
  for (int i = 0; i < 8; i++) {
    p[i] = (uint8_t)(v >> (8 * i));
  }
}

__device__ void build_message(uint8_t msg[136], const uint8_t challenge[32], uint64_t nonce) {
  for (int i = 0; i < 136; i++) {
    msg[i] = 0;
  }
  for (int i = 0; i < 32; i++) {
    msg[i] = challenge[i];
  }
  for (int i = 0; i < 8; i++) {
    msg[63 - i] = (uint8_t)(nonce >> (8 * i));
  }
  msg[64] = 0x01;
  msg[135] = 0x80;
}

__device__ void keccak256_abi_challenge_nonce(
  const uint8_t challenge[32],
  uint64_t nonce,
  uint8_t digest[32]
) {
  uint8_t msg[136];
  uint64_t st[25];

  for (int i = 0; i < 25; i++) {
    st[i] = 0;
  }
  build_message(msg, challenge, nonce);

  for (int i = 0; i < 17; i++) {
    st[i] ^= load64_le(msg + (i * 8));
  }
  keccakf(st);
  for (int i = 0; i < 4; i++) {
    store64_le(digest + (i * 8), st[i]);
  }
}

__device__ bool digest_below_difficulty(const uint8_t digest[32], const uint8_t difficulty[32]) {
  for (int i = 0; i < 32; i++) {
    if (digest[i] < difficulty[i]) return true;
    if (digest[i] > difficulty[i]) return false;
  }
  return false;
}

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
    uint8_t digest[32];
    keccak256_abi_challenge_nonce(challenge, nonce, digest);
    if (digest_below_difficulty(digest, difficulty)) {
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


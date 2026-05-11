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


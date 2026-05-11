package cuda

/*
#cgo CFLAGS: -I${SRCDIR}/../../cuda
#cgo LDFLAGS: ${SRCDIR}/../../cuda/hash_miner.o -lcudart -L/usr/local/cuda/lib64
#include "hash_miner.h"
*/
import "C"
import (
	"encoding/binary"
	"math/big"
	"unsafe"

	"hashminer/internal/miner"
)

type Searcher struct {
	Blocks  int
	Threads int
}

func NewSearcher(blocks int, threads int) Searcher {
	return Searcher{Blocks: blocks, Threads: threads}
}

func (s Searcher) Search(challenge [32]byte, difficulty *big.Int, startNonce uint64, count uint64) (miner.SearchResult, error) {
	diff := difficulty.FillBytes(make([]byte, 32))
	res := C.hashminer_search(
		(*C.uint8_t)(unsafe.Pointer(&challenge[0])),
		(*C.uint8_t)(unsafe.Pointer(&diff[0])),
		C.uint64_t(startNonce),
		C.uint64_t(count),
		C.int(s.Blocks),
		C.int(s.Threads),
	)
	return miner.SearchResult{
		Found:      res.found != 0,
		Nonce:      uint64(res.nonce),
		HashesDone: uint64(res.hashes_done),
	}, nil
}

func EncodeNonceABIWord(nonce uint64) [32]byte {
	var out [32]byte
	binary.BigEndian.PutUint64(out[24:], nonce)
	return out
}


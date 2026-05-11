package cuda

import "testing"

func TestEncodeNonceABIWord(t *testing.T) {
	got := EncodeNonceABIWord(1)
	if got[31] != 1 {
		t.Fatalf("last byte = %d", got[31])
	}
	for i := 0; i < 31; i++ {
		if got[i] != 0 {
			t.Fatalf("byte %d = %d", i, got[i])
		}
	}
}


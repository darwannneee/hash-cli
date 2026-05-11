package tui

import "testing"

func TestFormatHashrateUsesCompactUnits(t *testing.T) {
	tests := map[float64]string{
		56823099.26:   "56.8 MH/s",
		100000:        "100 KH/s",
		1250000:       "1.25 MH/s",
		999:           "999 H/s",
		1000000000000: "1.00 TH/s",
	}

	for input, expected := range tests {
		if got := formatHashrate(input); got != expected {
			t.Fatalf("formatHashrate(%f) = %q, want %q", input, got, expected)
		}
	}
}

func TestFormatWeiAsETH(t *testing.T) {
	got := formatWeiAsETH("2274130000000000")
	if got != "0.002274 ETH" {
		t.Fatalf("formatWeiAsETH returned %q", got)
	}
}

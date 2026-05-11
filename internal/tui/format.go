package tui

import (
	"math/big"
	"strconv"
)

func formatHashrate(rate float64) string {
	units := []string{"H/s", "KH/s", "MH/s", "GH/s", "TH/s"}
	value := rate
	unit := units[0]
	for i := 1; i < len(units) && value >= 1000; i++ {
		value /= 1000
		unit = units[i]
	}
	if value >= 100 {
		return strconv.FormatFloat(value, 'f', 0, 64) + " " + unit
	}
	if value >= 10 {
		return strconv.FormatFloat(value, 'f', 1, 64) + " " + unit
	}
	return strconv.FormatFloat(value, 'f', 2, 64) + " " + unit
}

func formatWeiAsETH(wei string) string {
	if wei == "" || wei == "-" {
		return "-"
	}
	value, ok := new(big.Int).SetString(wei, 10)
	if !ok {
		return wei
	}
	eth := new(big.Rat).SetFrac(value, big.NewInt(1_000_000_000_000_000_000))
	out := eth.FloatString(6)
	for len(out) > 1 && out[len(out)-1] == '0' {
		out = out[:len(out)-1]
	}
	if out[len(out)-1] == '.' {
		out += "0"
	}
	return out + " ETH"
}

package balance

import (
	"fmt"
	"math/big"
	"strings"
)

// parseUnits converts a human-readable decimal string ("0.5", "50", ".25") into
// base units for a token with `decimals` decimals (0.5 with 18 decimals -> 5e17).
// It is exact (string-based, no floats) and rejects more fractional digits than
// the token supports instead of silently rounding.
func parseUnits(s string, decimals uint8) (*big.Int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty amount")
	}
	intPart, fracPart, _ := strings.Cut(s, ".")
	if intPart == "" && fracPart == "" {
		return nil, fmt.Errorf("invalid amount %q", s)
	}
	if !allDigits(intPart) || !allDigits(fracPart) { // also rejects "-1", "1e3", "1.2.3"
		return nil, fmt.Errorf("invalid amount %q: expected a non-negative decimal number", s)
	}
	if len(fracPart) > int(decimals) {
		return nil, fmt.Errorf("amount %q has more than %d decimal places", s, decimals)
	}
	digits := intPart + fracPart + strings.Repeat("0", int(decimals)-len(fracPart))
	digits = strings.TrimLeft(digits, "0")
	if digits == "" {
		return new(big.Int), nil
	}
	v, ok := new(big.Int).SetString(digits, 10)
	if !ok {
		return nil, fmt.Errorf("invalid amount %q", s)
	}
	return v, nil
}

// formatUnits is the inverse: base units -> human-readable decimal string with
// trailing zeros trimmed (5e17, 18 -> "0.5"; 2e18, 18 -> "2").
func formatUnits(v *big.Int, decimals uint8) string {
	if v == nil {
		return ""
	}
	neg := v.Sign() < 0
	s := new(big.Int).Abs(v).String()
	if decimals > 0 {
		if len(s) <= int(decimals) {
			s = strings.Repeat("0", int(decimals)-len(s)+1) + s // left-pad so there is an integer part
		}
		cut := len(s) - int(decimals)
		s = s[:cut] + "." + strings.TrimRight(s[cut:], "0")
		s = strings.TrimSuffix(s, ".")
	}
	if neg {
		s = "-" + s
	}
	return s
}

func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

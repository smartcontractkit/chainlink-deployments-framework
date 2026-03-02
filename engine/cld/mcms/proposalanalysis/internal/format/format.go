package format

import (
	"math/big"
	"strings"
)

// CommaGroupBigInt adds comma separators to a big.Int for readability.
// E.g: 1000000 -> "1,000,000".
func CommaGroupBigInt(n *big.Int) string {
	if n == nil {
		return "0"
	}

	s := n.String()
	sign := ""
	if strings.HasPrefix(s, "-") {
		sign = "-"
		s = s[1:]
	}

	if len(s) <= 3 {
		return sign + s
	}

	var b strings.Builder
	b.WriteString(sign)
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteRune(',')
		}
		b.WriteRune(ch)
	}

	return b.String()
}

// FormatTokenAmount converts a raw token amount to a
// human-readable decimal string using the token's decimals.
func FormatTokenAmount(amount *big.Int, decimals uint8) string {
	if amount == nil || amount.Sign() == 0 {
		return "0"
	}

	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	whole := new(big.Int).Div(amount, divisor)
	remainder := new(big.Int).Mod(amount, divisor)

	if remainder.Sign() == 0 {
		return whole.String()
	}

	fracStr := remainder.String()
	if len(fracStr) < int(decimals) {
		fracStr = strings.Repeat("0", int(decimals)-len(fracStr)) + fracStr
	}
	fracStr = strings.TrimRight(fracStr, "0")

	return whole.String() + "." + fracStr
}

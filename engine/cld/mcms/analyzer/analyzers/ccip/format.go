package ccip

import (
	"fmt"
	"math/big"
	"strings"
)

// Ported from rddtool-ccip/mcmanalyzer/rendering/formatter/numbers.go:FormatAmountBigInt.
func FormatTokenAmount(amount *big.Int, decimals uint8) string {
	if amount == nil || amount.Sign() == 0 {
		return "0"
	}

	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	floatAmount := new(big.Float).Quo(
		new(big.Float).SetInt(amount),
		new(big.Float).SetInt(divisor),
	)

	floatVal, _ := floatAmount.Float64()

	formatted := strings.TrimRight(fmt.Sprintf("%.8f", floatVal), "0")
	formatted = strings.TrimRight(formatted, ".")

	return formatted
}

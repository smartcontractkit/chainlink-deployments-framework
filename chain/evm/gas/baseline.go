package gas

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

// BaselineFromTransactOpts returns the gas limit and price (legacy GasPrice or EIP-1559 GasFeeCap)
// currently set on the deployer transactor. Used as the starting point for boost retries after
// an attempt with zero overrides.
func BaselineFromTransactOpts(opts *bind.TransactOpts) (limit, priceWei uint64) {
	if opts == nil {
		return 0, 0
	}

	limit = opts.GasLimit
	priceWei = weiFromBigInt(opts.GasFeeCap)
	if priceWei == 0 {
		priceWei = weiFromBigInt(opts.GasPrice)
	}

	return limit, priceWei
}

// ResolveBoostPreviousGas returns the gas values to bump from, preferring explicit overrides
// from the last attempt and falling back to deployer baseline when overrides were zero.
func ResolveBoostPreviousGas(previousLimit, previousPrice, baselineLimit, baselinePrice uint64) (uint64, uint64) {
	prevLimit := previousLimit
	if prevLimit == 0 {
		prevLimit = baselineLimit
	}

	prevPrice := previousPrice
	if prevPrice == 0 {
		prevPrice = baselinePrice
	}

	return prevLimit, prevPrice
}

func weiFromBigInt(v *big.Int) uint64 {
	if v == nil || v.Sign() <= 0 || !v.IsUint64() {
		return 0
	}

	return v.Uint64()
}

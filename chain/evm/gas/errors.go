package gas

import (
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/vm"
)

// IsRetryable reports whether err indicates a gas-related failure that may succeed with higher gas.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, vm.ErrOutOfGas) ||
		errors.Is(err, vm.ErrCodeStoreOutOfGas) ||
		errors.Is(err, core.ErrIntrinsicGas) ||
		errors.Is(err, core.ErrFeeCapTooLow) ||
		errors.Is(err, core.ErrGasLimitReached) ||
		errors.Is(err, txpool.ErrUnderpriced) ||
		errors.Is(err, txpool.ErrReplaceUnderpriced) ||
		errors.Is(err, txpool.ErrTxGasPriceTooLow) ||
		errors.Is(err, txpool.ErrGasLimit) {
		return true
	}

	msg := strings.ToLower(err.Error())
	for _, pattern := range retryableErrorPatterns {
		if strings.Contains(msg, pattern) {
			return true
		}
	}

	return false
}

var retryableErrorPatterns = []string{
	"out of gas",
	"gas required exceeds allowance",
	"intrinsic gas too low",
	"underpriced",
	"replacement transaction underpriced",
	"max fee per gas less than block base fee",
	"exceeds block gas limit",
}

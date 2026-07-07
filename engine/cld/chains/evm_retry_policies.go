package chains

import (
	"strings"
	"time"

	evmclient "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider/rpcclient"
)

// isNonceTooLowError reports whether err indicates a nonce-too-low condition.
func isNonceTooLowError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "nonce too low")
}

// isNoContractCodeError reports whether err indicates contract code is not yet available.
func isNoContractCodeError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "no contract code")
}

func nonceTooLowRetryPolicy(delay time.Duration) evmclient.ErrorRetryPolicy {
	return evmclient.ErrorRetryPolicy{
		Match: isNonceTooLowError,
		Delay: delay,
	}
}

func noContractCodeRetryPolicy(delay time.Duration) evmclient.ErrorRetryPolicy {
	return evmclient.ErrorRetryPolicy{
		Match: isNoContractCodeError,
		Delay: delay,
	}
}

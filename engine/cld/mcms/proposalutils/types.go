package proposalutils

import (
	"math/big"

	mcmstypes "github.com/smartcontractkit/mcms/types"
)

// MCMSWithTimelockConfig holds the configuration for an MCMS with timelock.
type MCMSWithTimelockConfig struct {
	Canceller        mcmstypes.Config `json:"canceller"`
	Bypasser         mcmstypes.Config `json:"bypasser"`
	Proposer         mcmstypes.Config `json:"proposer"`
	TimelockMinDelay *big.Int         `json:"timelockMinDelay"`
	Label            *string          `json:"label"`
	GasBoostConfig   *GasBoostConfig  `json:"gasBoostConfig"`
	Qualifier        *string          `json:"qualifier"`
}

// GasBoostConfig defines the configuration for EVM gas boosting during retries.
// It allows customization of the initial gas limit, gas limit increment, initial gas price, and gas price increment.
type GasBoostConfig struct {
	InitialGasLimit   uint64 `json:"initialGasLimit"`
	GasLimitIncrement uint64 `json:"gasLimitIncrement"`
	InitialGasPrice   uint64 `json:"initialGasPrice"`
	GasPriceIncrement uint64 `json:"gasPriceIncrement"`
}

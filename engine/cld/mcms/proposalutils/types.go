package proposalutils

import (
	"math/big"

	"github.com/smartcontractkit/ccip-owner-contracts/pkg/config"
	mcmstypes "github.com/smartcontractkit/mcms/types"
)

// DEPRECATED: use MCMSWithTimelockConfig
// this will removed once all ccip code is migrated to use the new MCMSWithTimelockConfig type.
type MCMSWithTimelockConfigLegacy struct {
	Canceller        config.Config `json:"canceller"`
	Bypasser         config.Config `json:"bypasser"`
	Proposer         config.Config `json:"proposer"`
	TimelockMinDelay *big.Int      `json:"timelockMinDelay"`
	Label            *string       `json:"label"`
}

// MCMSWithTimelockConfig holds the configuration for an MCMS with timelock.
// Unlike the legacy MCMSWithTimelockConfigLegacy type above, this variant uses the
// newer mcmstypes.Config definitions.
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

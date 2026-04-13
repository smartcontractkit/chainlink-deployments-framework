package proposalutils

import (
	"math/big"

	"github.com/smartcontractkit/ccip-owner-contracts/pkg/config"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

const (
	BypasserManyChainMultisig  cldf.ContractType = "BypasserManyChainMultiSig"
	CancellerManyChainMultisig cldf.ContractType = "CancellerManyChainMultiSig"
	ProposerManyChainMultisig  cldf.ContractType = "ProposerManyChainMultiSig"
	ManyChainMultisig          cldf.ContractType = "ManyChainMultiSig"
	RBACTimelock               cldf.ContractType = "RBACTimelock"
	CallProxy                  cldf.ContractType = "CallProxy"

	// LinkToken is the burn/mint link token. It should be used everywhere for
	// new deployments. Corresponds to
	// https://github.com/smartcontractkit/chainlink/blob/develop/core/gethwrappers/shared/generated/link_token/link_token.go#L34
	LinkToken cldf.ContractType = "LinkToken"
	// StaticLinkToken represents the (very old) non-burn/mint link token.
	// It is not used in new deployments, but still exists on some chains
	// and has a distinct ABI from the new LinkToken.
	// Corresponds to the ABI
	// https://github.com/smartcontractkit/chainlink/blob/develop/core/gethwrappers/generated/link_token_interface/link_token_interface.go#L34
	StaticLinkToken cldf.ContractType = "StaticLinkToken"
	// mcms Solana specific
	ManyChainMultisigProgram         cldf.ContractType = "ManyChainMultiSigProgram"
	RBACTimelockProgram              cldf.ContractType = "RBACTimelockProgram"
	AccessControllerProgram          cldf.ContractType = "AccessControllerProgram"
	ProposerAccessControllerAccount  cldf.ContractType = "ProposerAccessControllerAccount"
	ExecutorAccessControllerAccount  cldf.ContractType = "ExecutorAccessControllerAccount"
	CancellerAccessControllerAccount cldf.ContractType = "CancellerAccessControllerAccount"
	BypasserAccessControllerAccount  cldf.ContractType = "BypasserAccessControllerAccount"
)

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

package proposalutils

import (
	"errors"

	ownerhelpers "github.com/smartcontractkit/ccip-owner-contracts/pkg/gethwrappers"
)

var (
	ErrMissingTimelockBinding = errors.New("missing Timelock RBACTimelock binding")
	ErrMissingCancellerMCM    = errors.New("missing CancellerMcm ManyChainMultiSig binding")
	ErrMissingProposerMCM     = errors.New("missing ProposerMcm ManyChainMultiSig binding")
	ErrMissingBypasserMCM     = errors.New("missing BypasserMcm ManyChainMultiSig binding")
	ErrMissingCallProxy       = errors.New("missing CallProxy binding")
)

// MCMSWithTimelockContracts holds the Go bindings
// for a MCMSWithTimelock standard contract deployment.
type MCMSWithTimelockContracts struct {
	CancellerMcm *ownerhelpers.ManyChainMultiSig
	BypasserMcm  *ownerhelpers.ManyChainMultiSig
	ProposerMcm  *ownerhelpers.ManyChainMultiSig
	Timelock     *ownerhelpers.RBACTimelock
	CallProxy    *ownerhelpers.CallProxy
}

// Validate checks all contract bindings are non-nil, ensuring the struct is ready
// for use generating views or interactions.
func (state MCMSWithTimelockContracts) Validate() error {
	if state.Timelock == nil {
		return ErrMissingTimelockBinding
	}
	if state.CancellerMcm == nil {
		return ErrMissingCancellerMCM
	}
	if state.ProposerMcm == nil {
		return ErrMissingProposerMCM
	}
	if state.BypasserMcm == nil {
		return ErrMissingBypasserMCM
	}
	if state.CallProxy == nil {
		return ErrMissingCallProxy
	}

	return nil
}

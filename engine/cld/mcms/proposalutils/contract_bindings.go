package proposalutils

import (
	"errors"

	ownerhelpers "github.com/smartcontractkit/ccip-owner-contracts/pkg/gethwrappers"
)

type MCMSWithTimelockContracts struct {
	CancellerMcm *ownerhelpers.ManyChainMultiSig
	BypasserMcm  *ownerhelpers.ManyChainMultiSig
	ProposerMcm  *ownerhelpers.ManyChainMultiSig
	Timelock     *ownerhelpers.RBACTimelock
	CallProxy    *ownerhelpers.CallProxy
}

// Validate checks that all fields are non-nil, ensuring it's ready
// for use generating views or interactions.
func (state MCMSWithTimelockContracts) Validate() error {
	if state.Timelock == nil {
		return errors.New("timelock not found")
	}
	if state.CancellerMcm == nil {
		return errors.New("canceller not found")
	}
	if state.ProposerMcm == nil {
		return errors.New("proposer not found")
	}
	if state.BypasserMcm == nil {
		return errors.New("bypasser not found")
	}
	if state.CallProxy == nil {
		return errors.New("call proxy not found")
	}
	return nil
}

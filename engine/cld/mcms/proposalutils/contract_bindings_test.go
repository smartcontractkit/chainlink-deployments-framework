package proposalutils

import (
	"testing"

	ownerhelpers "github.com/smartcontractkit/ccip-owner-contracts/pkg/gethwrappers"
	"github.com/stretchr/testify/require"
)

func TestMCMSWithTimelockContractsValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(MCMSWithTimelockContracts) MCMSWithTimelockContracts
		wantErr error
	}{
		{
			name: "success",
			mutate: func(state MCMSWithTimelockContracts) MCMSWithTimelockContracts {
				return state
			},
		},
		{
			name: "missing timelock binding",
			mutate: func(state MCMSWithTimelockContracts) MCMSWithTimelockContracts {
				state.Timelock = nil
				return state
			},
			wantErr: ErrMissingTimelockBinding,
		},
		{
			name: "missing canceller binding",
			mutate: func(state MCMSWithTimelockContracts) MCMSWithTimelockContracts {
				state.CancellerMcm = nil
				return state
			},
			wantErr: ErrMissingCancellerMCM,
		},
		{
			name: "missing proposer binding",
			mutate: func(state MCMSWithTimelockContracts) MCMSWithTimelockContracts {
				state.ProposerMcm = nil
				return state
			},
			wantErr: ErrMissingProposerMCM,
		},
		{
			name: "missing bypasser binding",
			mutate: func(state MCMSWithTimelockContracts) MCMSWithTimelockContracts {
				state.BypasserMcm = nil
				return state
			},
			wantErr: ErrMissingBypasserMCM,
		},
		{
			name: "missing call proxy binding",
			mutate: func(state MCMSWithTimelockContracts) MCMSWithTimelockContracts {
				state.CallProxy = nil
				return state
			},
			wantErr: ErrMissingCallProxy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := tt.mutate(validContractsState())
			err := state.Validate()
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}

			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func validContractsState() MCMSWithTimelockContracts {
	return MCMSWithTimelockContracts{
		CancellerMcm: new(ownerhelpers.ManyChainMultiSig),
		BypasserMcm:  new(ownerhelpers.ManyChainMultiSig),
		ProposerMcm:  new(ownerhelpers.ManyChainMultiSig),
		Timelock:     new(ownerhelpers.RBACTimelock),
		CallProxy:    new(ownerhelpers.CallProxy),
	}
}

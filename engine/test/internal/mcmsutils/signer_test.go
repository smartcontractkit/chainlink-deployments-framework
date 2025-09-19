package mcmsutils

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	mcmslib "github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSigner_SignMCMS(t *testing.T) {
	t.Parallel()

	// Generate a test private key
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	tests := []struct {
		name     string
		proposal func() *mcmslib.Proposal
		wantErr  string
	}{
		{
			name: "successfully signs MCMS proposal",
			proposal: func() *mcmslib.Proposal {
				return stubMCMSProposal()
			},
		},
		{
			name: "fails to signs proposal with empty operations",
			proposal: func() *mcmslib.Proposal {
				proposal := stubMCMSProposal()
				proposal.Operations = []mcmstypes.Operation{} // Empty operations

				return proposal
			},
			wantErr: "failed to validate MCMS proposal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup environment and signer
			signer, err := NewSigner()
			require.NoError(t, err)
			require.NotNil(t, signer)

			// Verify signer properties
			assert.True(t, signer.isEVMSim)

			// Create prop
			prop := tt.proposal()

			// Sign the proposal
			err = signer.SignMCMS(t.Context(), prop, privateKey)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, prop)
				require.Len(t, prop.Signatures, 1)

				// Verify signature properties
				sig := prop.Signatures[0]
				assert.NotEmpty(t, sig.R, "Signature R component should not be empty")
				assert.NotEmpty(t, sig.S, "Signature S component should not be empty")
				// V can be 0 or 1 for ECDSA signatures
				assert.LessOrEqual(t, sig.V, uint8(1), "Signature V component should be 0 or 1")
			}
		})
	}
}

func TestSigner_SignTimelock(t *testing.T) {
	t.Parallel()

	// Generate a test private key
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	tests := []struct {
		name             string
		timelockProposal func() *mcmslib.TimelockProposal
		wantErr          string
	}{
		{
			name: "signer creation succeeds",
			timelockProposal: func() *mcmslib.TimelockProposal {
				return stubTimelockProposal(mcmstypes.TimelockActionSchedule)
			},
		},
		{
			name: "invalid proposal",
			timelockProposal: func() *mcmslib.TimelockProposal {
				prop := stubTimelockProposal(mcmstypes.TimelockActionSchedule)
				prop.ChainMetadata = map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata{} // No chain metadata will fail validation

				return prop
			},
			wantErr: "failed to validate MCMS proposal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup environment and signer
			signer, err := NewSigner()
			require.NoError(t, err)
			require.NotNil(t, signer)

			// Verify signer properties
			assert.True(t, signer.isEVMSim)

			// Create proposal
			proposal := tt.timelockProposal()
			require.NotNil(t, proposal)

			// Sign the proposal
			err = signer.SignTimelock(t.Context(), proposal, privateKey)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Len(t, proposal.Signatures, 1)
				assert.NotEmpty(t, proposal.Signatures[0].R)
				assert.NotEmpty(t, proposal.Signatures[0].S)
				// V can be 0 or 1. Since V is a uint8 it can never be lower than 0.
				assert.LessOrEqual(t, proposal.Signatures[0].V, uint8(1))
			}
		})
	}
}

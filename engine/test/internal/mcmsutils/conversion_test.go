package mcmsutils

import (
	"encoding/json"
	"testing"

	chainselectors "github.com/smartcontractkit/chain-selectors"
	mcmslib "github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertTimelock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupProposal func(t *testing.T) mcmslib.TimelockProposal
		wantErr       string
		wantOpLen     int
	}{
		{
			name: "successfully converts EVM timelock proposal",
			setupProposal: func(t *testing.T) mcmslib.TimelockProposal {
				t.Helper()

				return *stubTimelockProposal(mcmstypes.TimelockActionSchedule)
			},
			wantOpLen: 1,
		},
		{
			name: "handles proposal with multiple chains",
			setupProposal: func(t *testing.T) mcmslib.TimelockProposal {
				t.Helper()

				additionalSelector := chainselectors.APTOS_LOCALNET.Selector
				prop := stubTimelockProposal(mcmstypes.TimelockActionSchedule)

				prop.ChainMetadata[mcmstypes.ChainSelector(additionalSelector)] = mcmstypes.ChainMetadata{
					MCMAddress: "0x1",
				}

				prop.Operations = append(prop.Operations, mcmstypes.BatchOperation{
					ChainSelector: mcmstypes.ChainSelector(additionalSelector),
					Transactions: []mcmstypes.Transaction{
						{
							To:               "0x123",
							AdditionalFields: json.RawMessage(`{"value": 0}`),
							Data:             []byte{1, 2, 3},
							OperationMetadata: mcmstypes.OperationMetadata{
								ContractType: "test",
								Tags:         []string{"test"},
							},
						},
					},
				})

				return *prop
			},
			wantOpLen: 2,
		},
		{
			name: "proposal with no operations",
			setupProposal: func(t *testing.T) mcmslib.TimelockProposal {
				t.Helper()

				prop := stubTimelockProposal(mcmstypes.TimelockActionSchedule)

				prop.Operations = []mcmstypes.BatchOperation{}

				return *prop
			},
			wantOpLen: 0,
		},
		{
			name: "fails with invalid chain selector",
			setupProposal: func(t *testing.T) mcmslib.TimelockProposal {
				t.Helper()

				return mcmslib.TimelockProposal{
					BaseProposal: mcmslib.BaseProposal{
						ChainMetadata: map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata{
							999999: { // Invalid selector
								StartingOpCount: 0,
								MCMAddress:      "0x0000000000000000000000000000000000000000",
							},
						},
					},
				}
			},
			wantErr: "failed to get selector family for chain",
		},
		{
			name: "fails to get converter factory",
			setupProposal: func(t *testing.T) mcmslib.TimelockProposal {
				t.Helper()

				return mcmslib.TimelockProposal{
					BaseProposal: mcmslib.BaseProposal{
						ChainMetadata: map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata{
							// We are choosing Tron here because it is not supported by the converter factory. This may change in the future.
							mcmstypes.ChainSelector(chainselectors.TRON_TESTNET_NILE.Selector): {
								MCMAddress: "0x0000000000000000000000000000000000000000",
							},
						},
					},
				}
			},
			wantErr: "failed to get converter factory for chain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			proposal := tt.setupProposal(t)

			ctx := t.Context()
			got, err := convertTimelock(ctx, proposal)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)

				// Verify the converted proposal is valid
				assert.Equal(t, mcmstypes.KindProposal, got.Kind)
				assert.Equal(t, proposal.Version, got.Version)
				assert.Len(t, got.Operations, tt.wantOpLen)
			}
		})
	}
}

package provider

import (
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/canton"
	chaincommon "github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
)

func Test_RPCChainProviderConfig_validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  RPCChainProviderConfig
		wantErr string
	}{
		{
			name: "valid config",
			config: RPCChainProviderConfig{
				Endpoints: []canton.ParticipantEndpoints{
					{
						JSONLedgerAPIURL: "",
						GRPCLedgerAPIURL: "",
						AdminAPIURL:      "",
						ValidatorAPIURL:  "",
					},
				},
				JWTProviders: []canton.JWTProvider{
					canton.NewStaticJWTProvider("token"),
				},
			},
		},
		{
			name:    "empty participants",
			config:  RPCChainProviderConfig{},
			wantErr: "no participants specified",
		},
		{
			name: "mismatched participants and JWT providers",
			config: RPCChainProviderConfig{
				Endpoints: []canton.ParticipantEndpoints{
					{
						JSONLedgerAPIURL: "",
						GRPCLedgerAPIURL: "",
						AdminAPIURL:      "",
						ValidatorAPIURL:  "",
					},
					{
						JSONLedgerAPIURL: "",
						GRPCLedgerAPIURL: "",
						AdminAPIURL:      "",
						ValidatorAPIURL:  "",
					},
				},
				JWTProviders: []canton.JWTProvider{
					canton.NewStaticJWTProvider("token"),
				},
			},
			wantErr: "number of participants must match number of JWT providers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.config.validate()
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_RPCChainProvider_Initialize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		giveSelector uint64
		giveConfig   RPCChainProviderConfig
		wantErr      string
	}{
		{
			name:         "valid initialization",
			giveSelector: chainsel.CANTON_LOCALNET.Selector,
			giveConfig: RPCChainProviderConfig{
				Endpoints: []canton.ParticipantEndpoints{
					{
						JSONLedgerAPIURL: "",
						GRPCLedgerAPIURL: "",
						AdminAPIURL:      "",
						ValidatorAPIURL:  "",
					},
				},
				JWTProviders: []canton.JWTProvider{
					canton.NewStaticJWTProvider("testToken"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := NewRPCChainProvider(tt.giveSelector, tt.giveConfig)
			chain, err := provider.Initialize(t.Context())
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, chain)

				gotChain, ok := chain.(*canton.Chain)
				require.True(t, ok, "expected chain to be of type *canton.Chain")
				require.Equal(t, tt.giveSelector, gotChain.Selector)
				require.Len(t, gotChain.Participants, len(tt.giveConfig.Endpoints))

				for i, participant := range gotChain.Participants {
					jwt, err := participant.JWTProvider.Token(t.Context())
					require.NoError(t, err)
					require.NotEmpty(t, jwt)
					require.Equal(t, "testToken", jwt)
					require.Equal(t, tt.giveConfig.Endpoints[i], participant.Endpoints)
				}

				// Check that subsequent calls to Initialize don't re-initialize the chain
				chainBefore := provider.chain
				chain2, err := provider.Initialize(t.Context())
				require.NoError(t, err)
				require.Equal(t, chain, chain2)
				require.Same(t, chainBefore, provider.chain)
			}
		})
	}
}

func Test_RPCChainProvider_Name(t *testing.T) {
	t.Parallel()

	p := &RPCChainProvider{}
	require.Equal(t, "Canton RPC Chain Provider", p.Name())
}

func Test_RPCChainProvider_ChainSelector(t *testing.T) {
	t.Parallel()

	p := &RPCChainProvider{selector: chainsel.CANTON_LOCALNET.Selector}
	require.Equal(t, chainsel.CANTON_LOCALNET.Selector, p.ChainSelector())
}

func Test_RPCChainProvider_BlockChain(t *testing.T) {
	t.Parallel()

	chain := &canton.Chain{
		ChainMetadata: chaincommon.ChainMetadata{Selector: chainsel.CANTON_LOCALNET.Selector},
		Participants: []canton.Participant{
			{Name: "Participant 1"},
			{Name: "Participant 2"},
		},
	}

	provider := &RPCChainProvider{
		chain: chain,
	}

	require.Equal(t, *chain, provider.BlockChain())
}

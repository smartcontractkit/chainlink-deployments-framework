package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/canton"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/canton/provider/authentication"
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
				Participants: []ParticipantConfig{
					{
						JSONLedgerAPIURL: "json-ledger-api",
						GRPCLedgerAPIURL: "grpc-ledger-api",
						AdminAPIURL:      "",
						ValidatorAPIURL:  "validator-api",
						UserID:           "user-id",
						PartyID:          "party-id",
						AuthProvider:     authentication.InsecureStaticProvider{AccessToken: ""},
					},
				},
			},
		},
		{
			name:    "empty participants",
			config:  RPCChainProviderConfig{},
			wantErr: "no participants specified",
		},
		{
			name: "invalid config - no JSONLedgerAPIURL",
			config: RPCChainProviderConfig{
				Participants: []ParticipantConfig{
					{
						JSONLedgerAPIURL: "",
					},
				},
			},
			wantErr: "no JSON Ledger API URL set",
		},
		{
			name: "invalid config - GRPCLedgerAPIURL",
			config: RPCChainProviderConfig{
				Participants: []ParticipantConfig{
					{
						JSONLedgerAPIURL: "json-ledger-api",
						GRPCLedgerAPIURL: "",
					},
				},
			},
			wantErr: "no gRPC Ledger API URL set",
		},
		{
			name: "invalid config - ValidatorAPIURL",
			config: RPCChainProviderConfig{
				Participants: []ParticipantConfig{
					{
						JSONLedgerAPIURL: "json-ledger-api",
						GRPCLedgerAPIURL: "grpc-ledger-api",
						AdminAPIURL:      "admin-api",
						ValidatorAPIURL:  "",
					},
				},
			},
			wantErr: "no Validator API URL set",
		},
		{
			name: "invalid config - no UserID",
			config: RPCChainProviderConfig{
				Participants: []ParticipantConfig{
					{
						JSONLedgerAPIURL: "json-ledger-api",
						GRPCLedgerAPIURL: "grpc-ledger-api",
						AdminAPIURL:      "admin-api",
						ValidatorAPIURL:  "validator-api",
						UserID:           "",
					},
				},
			},
			wantErr: "no User ID set",
		},
		{
			name: "invalid config - no PartyID",
			config: RPCChainProviderConfig{
				Participants: []ParticipantConfig{
					{
						JSONLedgerAPIURL: "json-ledger-api",
						GRPCLedgerAPIURL: "grpc-ledger-api",
						AdminAPIURL:      "admin-api",
						ValidatorAPIURL:  "validator-api",
						UserID:           "user-id",
						PartyID:          "",
					},
				},
			},
			wantErr: "no Party ID set",
		},
		{
			name: "invalid config - no AuthProvider",
			config: RPCChainProviderConfig{
				Participants: []ParticipantConfig{
					{
						JSONLedgerAPIURL: "json-ledger-api",
						GRPCLedgerAPIURL: "grpc-ledger-api",
						AdminAPIURL:      "admin-api",
						ValidatorAPIURL:  "validator-api",
						UserID:           "user-id",
						PartyID:          "party-id",
						AuthProvider:     nil,
					},
				},
			},
			wantErr: "no authentication provider set",
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
				Participants: []ParticipantConfig{
					{
						JSONLedgerAPIURL: "participant1-json-ledger-api.localhost:8080",
						GRPCLedgerAPIURL: "participant1-grpc-ledger-api-url.localhost:8080",
						AdminAPIURL:      "participant1-admin-api-url.localhost:8080",
						ValidatorAPIURL:  "participant1-validator-api-url.localhost:8080",
						UserID:           "participant1",
						PartyID:          "local-party-1",
						AuthProvider:     authentication.InsecureStaticProvider{AccessToken: "testToken"},
					},
				},
			},
		}, {
			name:         "valid initialization without Admin API",
			giveSelector: chainsel.CANTON_LOCALNET.Selector,
			giveConfig: RPCChainProviderConfig{
				Participants: []ParticipantConfig{
					{
						JSONLedgerAPIURL: "participant1-json-ledger-api.localhost:8080",
						GRPCLedgerAPIURL: "participant1-grpc-ledger-api-url.localhost:8080",
						AdminAPIURL:      "", // Not set
						ValidatorAPIURL:  "participant1-validator-api-url.localhost:8080",
						UserID:           "participant1",
						PartyID:          "local-party-1",
						AuthProvider:     authentication.InsecureStaticProvider{AccessToken: "testToken"},
					},
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
				assert.Equal(t, tt.giveSelector, gotChain.Selector)
				assert.Len(t, gotChain.Participants, len(tt.giveConfig.Participants))

				for i, participant := range gotChain.Participants {
					// Validate TokenSource is set
					token, err := participant.TokenSource.Token()
					require.NoError(t, err)
					assert.NotEmpty(t, token)
					assert.Equal(t, "testToken", token.AccessToken)
					// Validate endpoints are populated
					assert.Equal(t, tt.giveConfig.Participants[i].JSONLedgerAPIURL, participant.Endpoints.JSONLedgerAPIURL)
					assert.Equal(t, tt.giveConfig.Participants[i].GRPCLedgerAPIURL, participant.Endpoints.GRPCLedgerAPIURL)
					assert.Equal(t, tt.giveConfig.Participants[i].AdminAPIURL, participant.Endpoints.AdminAPIURL)
					assert.Equal(t, tt.giveConfig.Participants[i].ValidatorAPIURL, participant.Endpoints.ValidatorAPIURL)
					assert.Equal(t, tt.giveConfig.Participants[i].UserID, participant.UserID)
					assert.Equal(t, tt.giveConfig.Participants[i].PartyID, participant.PartyID)
					// Validate service clients have been created
					assert.NotNil(t, participant.LedgerServices.CommandCompletion)
					assert.NotNil(t, participant.LedgerServices.Command)
					assert.NotNil(t, participant.LedgerServices.CommandSubmission)
					assert.NotNil(t, participant.LedgerServices.EventQuery)
					assert.NotNil(t, participant.LedgerServices.PackageService)
					assert.NotNil(t, participant.LedgerServices.State)
					assert.NotNil(t, participant.LedgerServices.Update)
					assert.NotNil(t, participant.LedgerServices.Version)
					assert.NotNil(t, participant.LedgerServices.Admin.CommandInspection)
					assert.NotNil(t, participant.LedgerServices.Admin.IdentityProviderConfig)
					assert.NotNil(t, participant.LedgerServices.Admin.PackageManagement)
					assert.NotNil(t, participant.LedgerServices.Admin.ParticipantPruning)
					assert.NotNil(t, participant.LedgerServices.Admin.PartyManagement)
					assert.NotNil(t, participant.LedgerServices.Admin.UserManagement)
					// Validate admin service clients have been created
					if tt.giveConfig.Participants[i].AdminAPIURL != "" {
						require.NotNil(t, participant.AdminServices)
						assert.NotNil(t, participant.AdminServices.Package)
						assert.NotNil(t, participant.AdminServices.ParticipantInspection)
						assert.NotNil(t, participant.AdminServices.ParticipantRepair)
						assert.NotNil(t, participant.AdminServices.ParticipantStatus)
						assert.NotNil(t, participant.AdminServices.PartyManagement)
						assert.NotNil(t, participant.AdminServices.Ping)
						assert.NotNil(t, participant.AdminServices.Pruning)
						assert.NotNil(t, participant.AdminServices.ResourceManagement)
						assert.NotNil(t, participant.AdminServices.SynchronizerConnectivity)
						assert.NotNil(t, participant.AdminServices.TrafficControl)
					}
				}

				// Check that subsequent calls to Initialize don't re-initialize the chain
				chainBefore := provider.chain
				chain2, err := provider.Initialize(t.Context())
				require.NoError(t, err)
				assert.Equal(t, chain, chain2)
				assert.Same(t, chainBefore, provider.chain)
			}
		})
	}
}

func Test_RPCChainProvider_Name(t *testing.T) {
	t.Parallel()

	p := &RPCChainProvider{}
	assert.Equal(t, "Canton RPC Chain Provider", p.Name())
}

func Test_RPCChainProvider_ChainSelector(t *testing.T) {
	t.Parallel()

	p := &RPCChainProvider{selector: chainsel.CANTON_LOCALNET.Selector}
	assert.Equal(t, chainsel.CANTON_LOCALNET.Selector, p.ChainSelector())
}

func Test_RPCChainProvider_BlockChain(t *testing.T) {
	t.Parallel()

	chain := &canton.Chain{
		ChainMetadata: canton.ChainMetadata{Selector: chainsel.CANTON_LOCALNET.Selector},
		Participants: []canton.Participant{
			{Name: "Participant 1"},
			{Name: "Participant 2"},
		},
	}

	provider := &RPCChainProvider{
		chain: chain,
	}

	assert.Equal(t, *chain, provider.BlockChain())
}

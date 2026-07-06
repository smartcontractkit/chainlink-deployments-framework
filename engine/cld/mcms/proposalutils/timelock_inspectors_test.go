package proposalutils

import (
	"encoding/json"
	"testing"

	solrpc "github.com/gagliardetto/solana-go/rpc"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms/sdk/aptos"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	aptoschain "github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos"
	aptosmocks "github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos/mocks"
	chaincanton "github.com/smartcontractkit/chainlink-deployments-framework/chain/canton"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
	chainsui "github.com/smartcontractkit/chainlink-deployments-framework/chain/sui"
	suimocks "github.com/smartcontractkit/chainlink-deployments-framework/chain/sui/mocks"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/ton"
	tonmocks "github.com/smartcontractkit/chainlink-deployments-framework/chain/ton/mocks"
)

func TestMcmsTimelockInspectorForChain(t *testing.T) {
	t.Parallel()

	evmSelector := chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector
	solSelector := chainsel.SOLANA_DEVNET.Selector
	aptosSelector := chainsel.APTOS_TESTNET.Selector
	suiSelector := chainsel.SUI_TESTNET.Selector
	tonSelector := chainsel.TON_TESTNET.Selector
	cantonSelector := chainsel.CANTON_TESTNET.Selector

	suiMetadata := mcmstypes.ChainMetadata{
		AdditionalFields: json.RawMessage(`{
			"mcms_package_id":"0x1",
			"role":1,
			"account_obj":"0x2",
			"registry_obj":"0x3",
			"timelock_obj":"0x4",
			"deployer_state_obj":"0x5"
		}`),
	}

	chains := chain.NewBlockChains(map[uint64]chain.BlockChain{
		evmSelector: evm.Chain{
			Selector: evmSelector,
			Client:   evm.NewMockOnchainClient(t),
		},
		solSelector: solana.Chain{
			Selector: solSelector,
			Client:   solrpc.New("http://example.invalid"),
		},
		aptosSelector: aptoschain.Chain{
			Selector: aptosSelector,
			Client:   aptosmocks.NewMockAptosRpcClient(t),
		},
		suiSelector: chainsui.Chain{
			ChainMetadata: chainsui.ChainMetadata{Selector: suiSelector},
			Client:        suimocks.NewMockSuiPTBClient(t),
			Signer:        suimocks.NewMockSuiSigner(t),
		},
		tonSelector: ton.Chain{
			ChainMetadata: ton.ChainMetadata{Selector: tonSelector},
			Client:        tonmocks.NewMockAPIClientWrapped(t),
		},
		cantonSelector: chaincanton.Chain{
			ChainMetadata: chaincanton.ChainMetadata{Selector: cantonSelector},
			Participants: []chaincanton.Participant{
				{PartyID: "party::123"},
			},
		},
	})

	tests := []struct {
		name     string
		selector uint64
		metadata mcmstypes.ChainMetadata
		wantErr  string
	}{
		{name: "evm", selector: evmSelector},
		{name: "solana", selector: solSelector},
		{name: "aptos", selector: aptosSelector},
		{name: "aptos with metadata", selector: aptosSelector, metadata: mcmstypes.ChainMetadata{
			AdditionalFields: mustJSON(t, aptos.AdditionalFieldsMetadata{MCMSType: aptos.MCMSTypeRegular}),
		}},
		{name: "sui", selector: suiSelector, metadata: suiMetadata},
		{name: "ton", selector: tonSelector},
		{name: "canton", selector: cantonSelector},
		{
			name:     "missing evm chain",
			selector: evmSelector,
			wantErr:  "missing EVM chain client",
		},
		{
			name:     "aptos invalid metadata",
			selector: aptosSelector,
			metadata: mcmstypes.ChainMetadata{AdditionalFields: json.RawMessage("{")},
			wantErr:  "parse aptos metadata",
		},
		{
			name:     "sui invalid metadata",
			selector: suiSelector,
			metadata: mcmstypes.ChainMetadata{AdditionalFields: json.RawMessage("{")},
			wantErr:  "parse sui metadata",
		},
		{
			name:     "sui missing client",
			selector: suiSelector,
			metadata: suiMetadata,
			wantErr:  "missing Sui chain client",
		},
		{
			name:     "canton missing participant",
			selector: cantonSelector,
			wantErr:  "missing Canton chain participant",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			blockChains := chains
			if tt.wantErr == "missing EVM chain client" {
				blockChains = chain.NewBlockChains(nil)
			}
			if tt.wantErr == "missing Sui chain client" {
				blockChains = chain.NewBlockChains(nil)
			}
			if tt.wantErr == "missing Canton chain participant" {
				blockChains = chain.NewBlockChains(map[uint64]chain.BlockChain{
					cantonSelector: chaincanton.Chain{
						ChainMetadata: chaincanton.ChainMetadata{Selector: cantonSelector},
					},
				})
			}

			inspector, err := McmsTimelockInspectorForChain(blockChains, tt.selector, tt.metadata)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, inspector)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, inspector)
		})
	}
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()

	raw, err := json.Marshal(v)
	require.NoError(t, err)

	return raw
}

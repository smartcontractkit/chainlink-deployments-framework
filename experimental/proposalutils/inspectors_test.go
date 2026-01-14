package proposalutils

import (
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	cldfevm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	cldfsol "github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"

	mcmsTypes "github.com/smartcontractkit/mcms/types"
)

func TestMCMInspectorBuilder_BuildInspectors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                    string
		chainMetadata           map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata
		chainClientsEVM         map[uint64]cldfevm.Chain
		chainClientsSolana      map[uint64]cldfsol.Chain
		expectErr               bool
		errContains             string
		expectedInspectorsCount int
	}{
		{
			name:          "empty input",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{},
			chainClientsEVM: map[uint64]cldfevm.Chain{
				chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector: {Client: nil, Selector: chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector},
			},
			expectErr: false,
		},
		{
			name: "missing chain client",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
				1: {MCMAddress: "0xabc", StartingOpCount: 0},
			},
			chainClientsEVM: map[uint64]cldfevm.Chain{},
			expectErr:       true,
			errContains:     "error getting inspector for chain selector 1: error getting chainClient family: chain family not found for selector 1",
		},
		{
			name: "valid input",
			chainMetadata: map[mcmsTypes.ChainSelector]mcmsTypes.ChainMetadata{
				mcmsTypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector): {MCMAddress: "0xabc", StartingOpCount: 0},
				mcmsTypes.ChainSelector(chainsel.SOLANA_DEVNET.Selector):            {MCMAddress: "0xabc", StartingOpCount: 0},
				mcmsTypes.ChainSelector(chainsel.SOLANA_DEVNET.Selector):            {MCMAddress: "0xabc", StartingOpCount: 0},
			},
			chainClientsEVM: map[uint64]cldfevm.Chain{
				chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector: {Selector: chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector},
			},
			chainClientsSolana: map[uint64]cldfsol.Chain{
				chainsel.SOLANA_DEVNET.Selector: {Selector: chainsel.SOLANA_DEVNET.Selector},
			},
			expectErr:               false,
			expectedInspectorsCount: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			allChains := map[uint64]chain.BlockChain{}
			// Populate EVM chains
			for _, evmChain := range tc.chainClientsEVM {
				allChains[evmChain.ChainSelector()] = evmChain
			}
			// Populate SOL chains
			for _, solChain := range tc.chainClientsSolana {
				allChains[solChain.ChainSelector()] = solChain
			}
			builder := NewMCMInspectorFetcher(chain.NewBlockChains(allChains))
			inspectors, err := builder.FetchInspectors(tc.chainMetadata, mcmsTypes.TimelockActionSchedule)
			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
				require.Len(t, inspectors, tc.expectedInspectorsCount)
			}
		})
	}
}

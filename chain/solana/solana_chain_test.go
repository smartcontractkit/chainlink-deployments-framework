package solana_test

import (
	"sync"
	"testing"

	sollib "github.com/gagliardetto/solana-go"
	solsystem "github.com/gagliardetto/solana-go/programs/system"
	solrpc "github.com/gagliardetto/solana-go/rpc"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana/provider"
)

func TestChain_ChainInfot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		selector   uint64
		wantName   string
		wantString string
		wantFamily string
	}{
		{
			name:       "returns correct info",
			selector:   chainsel.SOLANA_MAINNET.Selector,
			wantString: "solana-mainnet (124615329519749607)",
			wantName:   chainsel.SOLANA_MAINNET.Name,
			wantFamily: chainsel.FamilySolana,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := solana.Chain{
				Selector: tt.selector,
			}
			assert.Equal(t, tt.selector, c.ChainSelector())
			assert.Equal(t, tt.wantString, c.String())
			assert.Equal(t, tt.wantName, c.Name())
			assert.Equal(t, tt.wantFamily, c.Family())
		})
	}
}

func TestChainMetadata_NetworkType(t *testing.T) {
	t.Parallel()

	c := solana.Chain{Selector: chainsel.SOLANA_MAINNET.Selector}
	got, err := c.NetworkType()
	require.NoError(t, err)
	assert.Equal(t, chainsel.NetworkTypeMainnet, got)

	c = solana.Chain{Selector: 0}
	_, err = c.NetworkType()
	require.Error(t, err)
}

func TestChainMetadata_IsNetworkType(t *testing.T) {
	t.Parallel()

	c := solana.Chain{Selector: chainsel.SOLANA_MAINNET.Selector}

	assert.True(t, c.IsNetworkType(chainsel.NetworkTypeMainnet))
	assert.False(t, c.IsNetworkType(chainsel.NetworkTypeTestnet))
}

func TestChain_ReadOnly(t *testing.T) {
	t.Parallel()

	receiverKey, err := sollib.NewRandomPrivateKey()
	require.NoError(t, err)
	receiverPubKey := receiverKey.PublicKey()

	solanaKey := blockchain.DefaultSolanaPrivateKey
	solanaConfig := provider.CTFChainProviderConfig{
		DeployerKeyGen: provider.PrivateKeyFromRaw(solanaKey),
		ProgramsPath:   t.TempDir(),
		ProgramIDs:     map[string]string{},
		Once:           &sync.Once{},
	}
	chainSelector := chainsel.SOLANA_DEVNET.Selector
	solanaProvider := provider.NewCTFChainProvider(t, chainSelector, solanaConfig)
	chain, err := solanaProvider.Initialize(t.Context())
	require.NoError(t, err)

	roChain, err := chain.ReadOnly()
	require.NoError(t, err)
	roSolChain, ok := roChain.(solana.Chain)
	require.True(t, ok)
	solChain, ok := chain.(solana.Chain)
	require.True(t, ok)

	// read with read-only client should work
	deployerPubKey, err := sollib.PublicKeyFromBase58(blockchain.DefaultSolanaPublicKey)
	require.NoError(t, err)
	balanceRes, err := roSolChain.Client.GetBalance(t.Context(), deployerPubKey, solrpc.CommitmentConfirmed)
	require.NoError(t, err)
	require.Equal(t, uint64(500000000000000000), balanceRes.Value)

	// write with read-write chain should work
	ix := solsystem.NewTransferInstruction(sollib.LAMPORTS_PER_SOL, solChain.DeployerKey.PublicKey(), receiverPubKey).Build()
	err = solChain.SendAndConfirm(t.Context(), []sollib.Instruction{ix})
	require.NoError(t, err)

	// write with read-only chain should fail
	ix = solsystem.NewTransferInstruction(sollib.LAMPORTS_PER_SOL, roSolChain.DeployerKey.PublicKey(), receiverPubKey).Build()
	err = roSolChain.SendAndConfirm(t.Context(), []sollib.Instruction{ix})
	require.ErrorContains(t, err, "Attempt to debit an account but found no record of a prior credit.")
}

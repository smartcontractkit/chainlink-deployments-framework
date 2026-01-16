package adapters

import (
	"testing"

	solrpc "github.com/gagliardetto/solana-go/rpc"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
	chainsui "github.com/smartcontractkit/chainlink-deployments-framework/chain/sui"
)

func TestChainAccess_UnknownSelector(t *testing.T) {
	t.Parallel()

	a := Wrap(chain.NewBlockChains(nil))

	evmClient, ok := a.EVMClient(999)
	require.False(t, ok)
	require.Nil(t, evmClient)

	solClient, ok := a.SolanaClient(999)
	require.False(t, ok)
	require.Nil(t, solClient)

	aptosClient, ok := a.AptosClient(999)
	require.False(t, ok)
	require.Nil(t, aptosClient)

	suiClient, suiSigner, ok := a.Sui(999)
	require.False(t, ok)
	require.Nil(t, suiClient)
	require.Nil(t, suiSigner)
}

func TestChainAccess_SelectorsAndLookups(t *testing.T) {
	t.Parallel()

	const (
		evmSel   = uint64(111)
		solSel   = uint64(222)
		aptosSel = uint64(333)
		suiSel   = uint64(444)
	)

	evmOnchain := evm.NewMockOnchainClient(t)
	solClient := solrpc.New("http://example.invalid")
	suiSigner, err := chainsui.NewSignerFromSeed(make([]byte, 32))
	require.NoError(t, err)

	chains := chain.NewBlockChains(map[uint64]chain.BlockChain{
		evmSel:   evm.Chain{Selector: evmSel, Client: evmOnchain},
		solSel:   solana.Chain{Selector: solSel, Client: solClient},
		aptosSel: aptos.Chain{Selector: aptosSel, Client: nil},
		suiSel: chainsui.Chain{
			ChainMetadata: chainsui.ChainMetadata{Selector: suiSel},
			Client:        nil,
			Signer:        suiSigner,
		},
	})

	a := Wrap(chains)
	require.Equal(t, chains.ListChainSelectors(), a.Selectors())

	gotEVM, ok := a.EVMClient(evmSel)
	require.True(t, ok)
	require.Equal(t, evmOnchain, gotEVM)

	gotSol, ok := a.SolanaClient(solSel)
	require.True(t, ok)
	require.Equal(t, solClient, gotSol)

	gotAptos, ok := a.AptosClient(aptosSel)
	require.True(t, ok)
	require.Nil(t, gotAptos)

	gotSuiClient, gotSuiSigner, ok := a.Sui(suiSel)
	require.True(t, ok)
	require.Nil(t, gotSuiClient)
	require.Equal(t, suiSigner, gotSuiSigner)
}

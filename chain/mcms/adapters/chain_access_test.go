package adapters

import (
	"testing"

	gethbind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	gethcommon "github.com/ethereum/go-ethereum/common"
	sol "github.com/gagliardetto/solana-go"
	solrpc "github.com/gagliardetto/solana-go/rpc"
	"github.com/stretchr/testify/require"
	tonwallet "github.com/xssnick/tonutils-go/ton/wallet"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos"
	aptosmocks "github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos/mocks"
	chaincanton "github.com/smartcontractkit/chainlink-deployments-framework/chain/canton"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/stellar"
	chainsui "github.com/smartcontractkit/chainlink-deployments-framework/chain/sui"
	suimocks "github.com/smartcontractkit/chainlink-deployments-framework/chain/sui/mocks"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/ton"
	tonmocks "github.com/smartcontractkit/chainlink-deployments-framework/chain/ton/mocks"
)

func TestChainAccess_UnknownSelector(t *testing.T) {
	t.Parallel()

	a := Wrap(chain.NewBlockChains(nil))

	evmClient, ok := a.EVMClient(999)
	require.False(t, ok)
	require.Nil(t, evmClient)

	evmSigner, ok := a.EVMSigner(999)
	require.False(t, ok)
	require.Nil(t, evmSigner)

	solClient, ok := a.SolanaClient(999)
	require.False(t, ok)
	require.Nil(t, solClient)

	solSigner, ok := a.SolanaSigner(999)
	require.False(t, ok)
	require.Nil(t, solSigner)

	aptosClient, ok := a.AptosClient(999)
	require.False(t, ok)
	require.Nil(t, aptosClient)

	aptosSigner, ok := a.AptosSigner(999)
	require.False(t, ok)
	require.Nil(t, aptosSigner)

	suiClient, ok := a.SuiClient(999)
	require.False(t, ok)
	require.Nil(t, suiClient)

	suiSigner, ok := a.SuiSigner(999)
	require.False(t, ok)
	require.Nil(t, suiSigner)

	tonClient, ok := a.TonClient(999)
	require.False(t, ok)
	require.Nil(t, tonClient)

	tonSigner, ok := a.TonSigner(999)
	require.False(t, ok)
	require.Nil(t, tonSigner)

	cantonChain, ok := a.CantonChain(999)
	require.False(t, ok)
	require.Equal(t, chaincanton.Chain{}, cantonChain)
}

func TestChainAccess_SelectorsAndLookups(t *testing.T) {
	t.Parallel()

	const (
		evmSel     = uint64(111)
		solSel     = uint64(222)
		aptosSel   = uint64(333)
		suiSel     = uint64(444)
		tonSel     = uint64(555)
		stellarSel = uint64(666)
		cantonSel  = uint64(777)
	)

	evmClient := evm.NewMockOnchainClient(t)
	evmSigner := &gethbind.TransactOpts{From: gethcommon.HexToAddress("0x123")}
	aptosClient := aptosmocks.NewMockAptosRpcClient(t)
	aptosSigner := aptosmocks.NewMockTransactionSigner(t)
	solClient := solrpc.New("http://example.invalid")
	solSigner := &sol.PrivateKey{1, 2, 3}
	suiClient := suimocks.NewMockISuiAPI(t)
	suiSigner, _ := chainsui.NewSignerFromSeed(make([]byte, 32))
	tonClient := tonmocks.NewMockAPIClientWrapped(t)
	tonSigner := &tonwallet.Wallet{}

	chains := chain.NewBlockChains(map[uint64]chain.BlockChain{
		evmSel:     evm.Chain{Selector: evmSel, Client: evmClient, DeployerKey: evmSigner},
		solSel:     solana.Chain{Selector: solSel, Client: solClient, DeployerKey: solSigner},
		aptosSel:   aptos.Chain{Selector: aptosSel, Client: aptosClient, DeployerSigner: aptosSigner},
		suiSel:     chainsui.Chain{ChainMetadata: chainsui.ChainMetadata{Selector: suiSel}, Client: suiClient, Signer: suiSigner},
		tonSel:     ton.Chain{ChainMetadata: ton.ChainMetadata{Selector: tonSel}, Client: tonClient, Wallet: tonSigner},
		stellarSel: stellar.Chain{ChainMetadata: stellar.ChainMetadata{Selector: stellarSel}, Client: nil},
		cantonSel:  chaincanton.Chain{ChainMetadata: chaincanton.ChainMetadata{Selector: cantonSel}, Participants: nil},
	})

	a := Wrap(chains)
	require.Equal(t, chains.ListChainSelectors(), a.Selectors())

	gotEVMClient, ok := a.EVMClient(evmSel)
	require.True(t, ok)
	require.Equal(t, evmClient, gotEVMClient)

	gotEVMSigner, ok := a.EVMSigner(evmSel)
	require.True(t, ok)
	require.Equal(t, evmSigner, gotEVMSigner)

	gotSolClient, ok := a.SolanaClient(solSel)
	require.True(t, ok)
	require.Equal(t, solClient, gotSolClient)

	gotSolSigner, ok := a.SolanaSigner(solSel)
	require.True(t, ok)
	require.Equal(t, solSigner, gotSolSigner)

	gotAptosClient, ok := a.AptosClient(aptosSel)
	require.True(t, ok)
	require.Equal(t, aptosClient, gotAptosClient)

	gotAptosSigner, ok := a.AptosSigner(aptosSel)
	require.True(t, ok)
	require.Equal(t, aptosSigner, gotAptosSigner)

	gotSuiClient, ok := a.SuiClient(suiSel)
	require.True(t, ok)
	require.Equal(t, suiClient, gotSuiClient)

	gotSuiSigner, ok := a.SuiSigner(suiSel)
	require.True(t, ok)
	require.Equal(t, suiSigner, gotSuiSigner)

	gotTonClient, ok := a.TonClient(tonSel)
	require.True(t, ok)
	require.Equal(t, tonClient, gotTonClient)

	gotTonSigner, ok := a.TonSigner(tonSel)
	require.True(t, ok)
	require.Equal(t, tonSigner, gotTonSigner)

	gotStellar, ok := a.StellarClient(stellarSel)
	require.True(t, ok)
	require.Nil(t, gotStellar)

	gotCanton, ok := a.CantonChain(cantonSel)
	require.True(t, ok)
	require.Equal(t, cantonSel, gotCanton.Selector)
	require.Nil(t, gotCanton.Participants)
}

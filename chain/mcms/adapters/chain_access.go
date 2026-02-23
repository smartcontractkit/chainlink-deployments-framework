package adapters

import (
	aptoslib "github.com/aptos-labs/aptos-go-sdk"
	"github.com/block-vision/sui-go-sdk/sui"
	solrpc "github.com/gagliardetto/solana-go/rpc"
	"github.com/smartcontractkit/mcms/sdk/evm"
	mcmssui "github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/stellar/go-stellar-sdk/clients/rpcclient"
	"github.com/xssnick/tonutils-go/ton"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	cldfaptos "github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos"
	cldfevm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	cldfsol "github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
	cldfstellar "github.com/smartcontractkit/chainlink-deployments-framework/chain/stellar"
	cldfsui "github.com/smartcontractkit/chainlink-deployments-framework/chain/sui"
	cldfton "github.com/smartcontractkit/chainlink-deployments-framework/chain/ton"
)

type ChainsFetcher interface {
	ListChainSelectors(options ...chain.ChainSelectorsOption) []uint64
	EVMChains() map[uint64]cldfevm.Chain
	SolanaChains() map[uint64]cldfsol.Chain
	AptosChains() map[uint64]cldfaptos.Chain
	SuiChains() map[uint64]cldfsui.Chain
	TonChains() map[uint64]cldfton.Chain
	StellarChains() map[uint64]cldfstellar.Chain
}

// ChainAccessAdapter adapts CLDF's chain.BlockChains into a selector + lookup style API.
// It is used to make it compatible with the mcms lib chain access interface.
type ChainAccessAdapter struct {
	inner ChainsFetcher
}

// Wrap returns a ChainAccessAdapter adapter around the given CLDF BlockChains.
func Wrap(inner ChainsFetcher) ChainAccessAdapter {
	return ChainAccessAdapter{inner: inner}
}

// Selectors returns all known chain selectors (sorted by CLDF).
func (a *ChainAccessAdapter) Selectors() []uint64 {
	return a.inner.ListChainSelectors()
}

// EVMClient returns an EVM client for the given selector.
func (a *ChainAccessAdapter) EVMClient(selector uint64) (evm.ContractDeployBackend, bool) {
	ch, ok := a.inner.EVMChains()[selector]
	if !ok {
		return nil, false
	}

	return ch.Client, true
}

// SolanaClient returns the Solana RPC client for the given selector.
func (a *ChainAccessAdapter) SolanaClient(selector uint64) (*solrpc.Client, bool) {
	ch, ok := a.inner.SolanaChains()[selector]
	if !ok {
		return nil, false
	}

	return ch.Client, true
}

// AptosClient returns the Aptos RPC client for the given selector.
func (a *ChainAccessAdapter) AptosClient(selector uint64) (aptoslib.AptosRpcClient, bool) {
	ch, ok := a.inner.AptosChains()[selector]
	if !ok {
		return nil, false
	}

	return ch.Client, true
}

// SuiClient returns the Sui API client and signer for the given selector.
func (a *ChainAccessAdapter) SuiClient(selector uint64) (sui.ISuiAPI, mcmssui.SuiSigner, bool) {
	ch, ok := a.inner.SuiChains()[selector]
	if !ok {
		return nil, nil, false
	}

	return ch.Client, ch.Signer, true
}

// TonClient returns the Ton API client for the given selector.
func (a *ChainAccessAdapter) TonClient(selector uint64) (ton.APIClientWrapped, bool) {
	ch, ok := a.inner.TonChains()[selector]
	if !ok {
		return nil, false
	}

	return ch.Client, true
}

// StellarClient returns the Stellar RPC client for the given selector.
func (a *ChainAccessAdapter) StellarClient(selector uint64) (*rpcclient.Client, bool) {
	ch, ok := a.inner.StellarChains()[selector]
	if !ok {
		return nil, false
	}

	return ch.Client, true
}

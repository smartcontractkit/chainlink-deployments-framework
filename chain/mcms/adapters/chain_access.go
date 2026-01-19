package adapters

import (
	aptoslib "github.com/aptos-labs/aptos-go-sdk"
	"github.com/block-vision/sui-go-sdk/sui"
	solrpc "github.com/gagliardetto/solana-go/rpc"
	"github.com/smartcontractkit/mcms/sdk/evm"
	mcmssui "github.com/smartcontractkit/mcms/sdk/sui"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
)

// ChainAccessAdapter adapts CLDF's chain.BlockChains into a selector + lookup style API.
// It is used to make it compatible with the mcms lib chain access interface.
type ChainAccessAdapter struct {
	inner chain.BlockChains
}

// Wrap returns a ChainAccessAdapter adapter around the given CLDF BlockChains.
func Wrap(inner chain.BlockChains) ChainAccessAdapter {
	return ChainAccessAdapter{inner: inner}
}

// Selectors returns all known chain selectors (sorted by CLDF).
func (a ChainAccessAdapter) Selectors() []uint64 {
	return a.inner.ListChainSelectors()
}

// EVMClient returns a EVM client for the given selector.
func (a ChainAccessAdapter) EVMClient(selector uint64) (evm.ContractDeployBackend, bool) {
	ch, ok := a.inner.EVMChains()[selector]
	if !ok {
		return nil, false
	}

	return ch.Client, true
}

// SolanaClient returns the Solana RPC client for the given selector.
func (a ChainAccessAdapter) SolanaClient(selector uint64) (*solrpc.Client, bool) {
	ch, ok := a.inner.SolanaChains()[selector]
	if !ok {
		return nil, false
	}

	return ch.Client, true
}

// AptosClient returns the Aptos RPC client for the given selector.
func (a ChainAccessAdapter) AptosClient(selector uint64) (aptoslib.AptosRpcClient, bool) {
	ch, ok := a.inner.AptosChains()[selector]
	if !ok {
		return nil, false
	}

	return ch.Client, true
}

// SuiClient returns the Sui API client and signer for the given selector.
func (a ChainAccessAdapter) SuiClient(selector uint64) (sui.ISuiAPI, mcmssui.SuiSigner, bool) {
	ch, ok := a.inner.SuiChains()[selector]
	if !ok {
		return nil, nil, false
	}

	return ch.Client, ch.Signer, true
}

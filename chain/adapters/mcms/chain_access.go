package mcms

import (
	aptoslib "github.com/aptos-labs/aptos-go-sdk"
	"github.com/block-vision/sui-go-sdk/sui"
	solrpc "github.com/gagliardetto/solana-go/rpc"

	mcmssdk "github.com/smartcontractkit/mcms/sdk"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
)

// ChainAccess adapts CLDF's chain.BlockChains into a selector + lookup style API.
// It used to make compatible with mcms lib chain access interface.
type ChainAccess struct {
	inner chain.BlockChains
}

// Wrap returns a ChainAccess adapter around the given CLDF BlockChains.
func Wrap(inner chain.BlockChains) ChainAccess {
	return ChainAccess{inner: inner}
}

// Selectors returns all known chain selectors (sorted by CLDF).
func (a ChainAccess) Selectors() []uint64 {
	return a.inner.ListChainSelectors()
}

// EVMClient returns a EVM client for the given selector.
func (a ChainAccess) EVMClient(selector uint64) (mcmssdk.ContractDeployBackend, bool) {
	ch, ok := a.inner.EVMChains()[selector]
	if !ok {
		return nil, false
	}

	return ch.Client, true
}

// SolanaClient returns the Solana RPC client for the given selector.
func (a ChainAccess) SolanaClient(selector uint64) (*solrpc.Client, bool) {
	ch, ok := a.inner.SolanaChains()[selector]
	if !ok {
		return nil, false
	}

	return ch.Client, true
}

// AptosClient returns the Aptos RPC client for the given selector.
func (a ChainAccess) AptosClient(selector uint64) (aptoslib.AptosRpcClient, bool) {
	ch, ok := a.inner.AptosChains()[selector]
	if !ok {
		return nil, false
	}

	return ch.Client, true
}

// Sui returns the Sui API client and signer for the given selector.
func (a ChainAccess) Sui(selector uint64) (sui.ISuiAPI, mcmssdk.SuiSigner, bool) {
	ch, ok := a.inner.SuiChains()[selector]
	if !ok {
		return nil, nil, false
	}

	return ch.Client, ch.Signer, true
}

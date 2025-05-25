package chain

import "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"

type Provider interface {
	Initialize() error
	Name() string
	ChainSelector() uint64
	BlockChain() BlockChain
}

type EVMProvider interface {
	Provider
	Chain() *evm.Chain
}

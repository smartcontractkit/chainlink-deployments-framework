package chain

import "context"

// Provider is an interface for blockchain providers that can initialize a blockchain instance.
type Provider interface {
	Initialize(ctx context.Context) (BlockChain, error)
	Name() string
	ChainSelector() uint64
	BlockChain() BlockChain
}

// ChainLoader is an interface for loading a blockchain instance lazily.
// It's used by the lazy loading mechanism in BlockChains to load chains on-demand.
type ChainLoader interface {
	Load(ctx context.Context, selector uint64) (BlockChain, error)
}

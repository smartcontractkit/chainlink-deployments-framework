package chain

import "context"

type Provider interface {
	Initialize(ctx context.Context) (BlockChain, error)
	Name() string
	ChainSelector() uint64
	BlockChain() BlockChain
}

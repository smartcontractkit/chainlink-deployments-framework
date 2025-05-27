package chain

type Provider interface {
	Initialize() (BlockChain, error)
	Name() string
	ChainSelector() uint64
	BlockChain() BlockChain
}

package chain

type Provider interface {
	Initialize() error
	Name() string
	ChainSelector() uint64
	BlockChain() BlockChain
}

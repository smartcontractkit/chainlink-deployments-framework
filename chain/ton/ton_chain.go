package ton

import (
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
)

// Chain represents a TON chain.
type Chain struct {
	Selector      uint64           // Canonical chain identifier
	Client        *ton.APIClient   // RPC client via Lite Server
	Wallet        *wallet.Wallet   // Wallet abstraction (signing, sending)
	WalletAddress *address.Address // Address of deployer wallet
	URL           string           // Liteserver URL
	DeployerSeed  string           // Optional: mnemonic or raw seed
}

// ChainSelector returns the chain selector of the chain
func (c Chain) ChainSelector() uint64 {
	return c.Selector
}

// String returns chain name and selector "<name> (<selector>)"
func (c Chain) String() string {
	return common.ChainMetadata{Selector: c.Selector}.String()
}

// Name returns the name of the chain
func (c Chain) Name() string {
	return common.ChainMetadata{Selector: c.Selector}.Name()
}

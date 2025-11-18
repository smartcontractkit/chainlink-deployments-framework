package ton

import (
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
)

type ChainMetadata = common.ChainMetadata

// Chain represents a TON chain.
type Chain struct {
	ChainMetadata                      // Contains canonical chain identifier
	Client        ton.APIClientWrapped // APIClient for Lite Server connection
	Wallet        *wallet.Wallet       // Wallet abstraction (signing, sending)
	WalletAddress *address.Address     // Address of deployer wallet
	URL           string               // Liteserver URL
}

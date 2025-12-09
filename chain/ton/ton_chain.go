package ton

import (
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
	"github.com/xssnick/tonutils-go/tlb"
)

type ChainMetadata = common.ChainMetadata

// TxOps holds configuration for transaction operations.
type TxOps struct {
	Wallet *wallet.Wallet // Wallet abstraction (signing, sending)
	Amount tlb.Coins      // Default amount for msg transfers
}

// Chain represents a TON chain.
type Chain struct {
	ChainMetadata                  // Contains canonical chain identifier
	Client        *ton.APIClient   // APIClient for Lite Server connection
	Wallet        *wallet.Wallet   // Wallet abstraction (signing, sending)
	WalletAddress *address.Address // Address of deployer wallet
	URL           string           // Liteserver URL
	TxOps         TxOps            // Transaction operations configuration
}

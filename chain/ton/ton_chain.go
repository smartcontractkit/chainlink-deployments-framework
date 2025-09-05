package ton

import (
	"fmt"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
)

type ChainMetadata = common.ChainMetadata

// Chain represents a TON chain.
type Chain struct {
	ChainMetadata                  // Contains canonical chain identifier
	Client        *ton.APIClient   // RPC client via Lite Server
	Wallet        *wallet.Wallet   // Wallet abstraction (signing, sending)
	WalletAddress *address.Address // Address of deployer wallet
	URL           string           // Liteserver URL
	DeployerSeed  string           // Optional: mnemonic or raw seed
}

// AddressToBytes converts a TON address string to bytes.
func (c Chain) AddressToBytes(addressStr string) ([]byte, error) {
	addr, err := address.ParseAddr(addressStr)
	if err != nil {
		return nil, fmt.Errorf("invalid TON address format: %s, error: %w", addressStr, err)
	}

	// Return the raw 32-byte address data
	return addr.Data(), nil
}

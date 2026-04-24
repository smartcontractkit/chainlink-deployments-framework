package stellar

import (
	"log"

	"github.com/stellar/go-stellar-sdk/clients/rpcclient"
	"github.com/stellar/go-stellar-sdk/keypair"

	chaincommon "github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
)

type ChainMetadata = chaincommon.ChainMetadata

// Chain represents a Stellar network instance used by the Chainlink Deployments Framework (CLDF).
type Chain struct {
	ChainMetadata

	// Client is the Soroban RPC client for interacting with the Stellar network
	Client *rpcclient.Client

	// Signer is the keypair used for signing transactions
	Signer StellarSigner

	// URL is the Soroban RPC endpoint URL
	URL string

	// FriendbotURL is the Friendbot endpoint URL for funding test accounts (optional, only required for testing)
	FriendbotURL string

	// NetworkPassphrase identifies the Stellar network
	NetworkPassphrase string
}

func (c Chain) ReadOnly() any {
	keyPair, err := keypair.Random()
	if err != nil {
		log.Fatalf("failed to create keypair for chain %v: %v", c, err.Error())
	}
	c.Signer = NewStellarKeypairSigner(keyPair)

	return c
}

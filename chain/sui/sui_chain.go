package sui

import (
<<<<<<< HEAD
	"crypto/ed25519"
	"crypto/rand"
	"fmt"

	"github.com/block-vision/sui-go-sdk/sui"
||||||| parent of 8e78f6e (update sui client)
	"github.com/block-vision/sui-go-sdk/sui"
=======
	cslclient "github.com/smartcontractkit/chainlink-sui/relayer/client"
>>>>>>> 8e78f6e (update sui client)

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
)

type ChainMetadata = common.ChainMetadata

// Chain represents an Sui chain.
type Chain struct {
	ChainMetadata
	Client    cslclient.SuiPTBClient
	Signer    SuiSigner
	URL       string
	FaucetURL string

	// TODO: Implement ConfirmTransaction. Current tooling relies on node local execution
}

func (c Chain) ReadOnly() (common.BlockChain, error) {
	privateKey := make([]byte, ed25519.SeedSize)
	_, err := rand.Read(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key for read-only chain %v: %w", c, err)
	}
	c.Signer, _ = NewSignerFromSeed(privateKey)

	return c, nil
}

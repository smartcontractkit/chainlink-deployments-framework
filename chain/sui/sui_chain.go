package sui

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"

	"github.com/block-vision/sui-go-sdk/sui"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
)

type ChainMetadata = common.ChainMetadata

// Chain represents an Sui chain.
type Chain struct {
	ChainMetadata
	Client    sui.ISuiAPI
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

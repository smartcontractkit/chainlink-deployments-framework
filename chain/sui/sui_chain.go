package sui

import (
	"crypto/rand"
	"log"

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

func (c Chain) ReadOnly() any {
	privateKey := make([]byte, 64)
	_, err := rand.Read(privateKey)
	if err != nil {
		log.Fatalf("unable to generate private key for read-only chain %v: %s", c, err.Error())
	}
	c.Signer, _ = NewSignerFromSeed(privateKey)

	return c
}

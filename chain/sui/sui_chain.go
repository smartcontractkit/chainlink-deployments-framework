package sui

import (
	"crypto/ed25519"

	"github.com/block-vision/sui-go-sdk/sui"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
)

type ChainMetadata = common.ChainMetadata

// Chain represents an Sui chain.
type Chain struct {
	ChainMetadata
	Client sui.ISuiAPI
	// TODO: sui-go currently does not have a working Signer interface, so we
	// have the raw private key for now.
	DeployerKey ed25519.PrivateKey
	URL         string

	Confirm func(txHash string, opts ...any) error
}

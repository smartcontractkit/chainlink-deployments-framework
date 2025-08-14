package sui

import (
	"github.com/block-vision/sui-go-sdk/sui"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
)

type ChainMetadata = common.ChainMetadata

// Chain represents an Sui chain.
type Chain struct {
	ChainMetadata
	Client sui.ISuiAPI
	Signer SuiSigner
	URL    string
	// TODO: Implement ConfirmTransaction. Current tooling relies on node local execution
}

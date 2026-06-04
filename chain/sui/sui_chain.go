package sui

import (
	cslclient "github.com/smartcontractkit/chainlink-sui/relayer/client"

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

package stellar

import (
	chaincommon "github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
)

type ChainMetadata = chaincommon.ChainMetadata

// Chain represents a Stellar network instance used by the Chainlink Deployments Framework (CLDF).
type Chain struct {
	ChainMetadata
}

package stellar

import (
	chaincommon "github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
)

type ChainMetadata = chaincommon.ChainMetadata

// Chain represents a Stellar network instance used CLDF.
type Chain struct {
	ChainMetadata
}

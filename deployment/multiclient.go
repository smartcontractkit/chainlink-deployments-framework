package deployment

import (
	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	cldf_evm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
)

type RetryConfig = cldf_evm.RetryConfig

// MultiClient provides failover functionality for Ethereum RPC clients.
type MultiClient = cldf_evm.MultiClient

// NewMultiClient creates a new MultiClient with failover capabilities.
func NewMultiClient(lggr logger.Logger, rpcsCfg RPCConfig, opts ...func(client *MultiClient)) (*MultiClient, error) {
	return cldf_evm.NewMultiClient(lggr, rpcsCfg, opts...)
}

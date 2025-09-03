package deployment

import (
	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider/rpcclient"
)

type RetryConfig = rpcclient.RetryConfig

// MultiClient provides failover functionality for Ethereum RPC clients.
type MultiClient = rpcclient.MultiClient

// NewMultiClient creates a new MultiClient with failover capabilities.
func NewMultiClient(lggr logger.Logger, rpcsCfg RPCConfig, opts ...func(client *MultiClient)) (*MultiClient, error) {
	return rpcclient.NewMultiClient(lggr, rpcsCfg, opts...)
}

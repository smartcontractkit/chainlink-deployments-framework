package deployment

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider/rpcclient"
)

type RetryConfig = rpcclient.RetryConfig

// MultiClient provides failover functionality for Ethereum RPC clients.
type MultiClient = rpcclient.MultiClient

// NewMultiClient creates a new MultiClient with failover capabilities.
var NewMultiClient = rpcclient.NewMultiClient

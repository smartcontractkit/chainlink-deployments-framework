package deployment

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider/rpcclient"
)

// URLSchemePreference defines URL scheme preferences for RPC connections.
type URLSchemePreference = rpcclient.URLSchemePreference

const (
	URLSchemePreferenceNone = rpcclient.URLSchemePreferenceNone
	URLSchemePreferenceWS   = rpcclient.URLSchemePreferenceWS
	URLSchemePreferenceHTTP = rpcclient.URLSchemePreferenceHTTP
)

// URLSchemePreferenceFromString converts a string to URLSchemePreference.
var URLSchemePreferenceFromString = rpcclient.URLSchemePreferenceFromString

// RPC represents a single RPC endpoint configuration.
type RPC = rpcclient.RPC

// RPCConfig is a configuration for a chain.
// It contains a chain selector and a list of RPCs
type RPCConfig = rpcclient.RPCConfig

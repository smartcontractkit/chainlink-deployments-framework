package deployment

import (
	cldf_evm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
)

// URLSchemePreference defines URL scheme preferences for RPC connections.
type URLSchemePreference = cldf_evm.URLSchemePreference

const (
	URLSchemePreferenceNone = cldf_evm.URLSchemePreferenceNone
	URLSchemePreferenceWS   = cldf_evm.URLSchemePreferenceWS
	URLSchemePreferenceHTTP = cldf_evm.URLSchemePreferenceHTTP
)

// URLSchemePreferenceFromString converts a string to URLSchemePreference.
var URLSchemePreferenceFromString = cldf_evm.URLSchemePreferenceFromString

// RPC represents a single RPC endpoint configuration.
type RPC = cldf_evm.RPC

// RPCConfig is a configuration for a chain.
// It contains a chain selector and a list of RPCs
type RPCConfig = cldf_evm.RPCConfig

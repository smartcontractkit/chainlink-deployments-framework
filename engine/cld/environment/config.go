package environment

import (
	cldf_config_env "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
	cldf_config_network "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
)

// Config consolidates all the config that is required to be loaded for a domain environment.
//
// Specifically it contains the network config and secrets which is loaded from files or env vars.
type Config struct {
	Networks *cldf_config_network.Config // The network config loaded from the network manifest file
	Env      *cldf_config_env.Config     // The cld engine's environment config
}

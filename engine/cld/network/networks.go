package network

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	cldf_environment "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"

	cldf_config_network "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
)

const (
	// RpcsCldevSh is the host for the CL RPC Proxies.
	RpcsCldevSh = "rpcs.cldev.sh"
	// GapRPCProxy is the host for the GAP V2 proxy which is required to be used when accessing
	// CL RPC Proxies from Github.
	GapRPCProxy = "gap-rpc-proxy.public.main.prod.cldev.sh"
	// GapRPCProxyWSPort is the port for the GAP V2 proxy WebSocket endpoint.
	GapRPCProxyWSPort = "9443"
	// GapRPCProxyHTTPPort is the port for the GAP V2 proxy HTTP endpoint.
	GapRPCProxyHTTPPort = "4443"
)

// LoadNetworks retrieves the network configuration for the given domain and filters the networks
// according to the specified environment. This ensures that only networks relevant to the selected
// environment are accessible, minimizing the risk of accidental operations on unintended networks.
func LoadNetworks(
	env string, domain domain.Domain, lggr logger.Logger,
) (*cldf_config_network.Config, error) {
	cfg, err := loadNetworkConfig(domain)
	if err != nil {
		return nil, fmt.Errorf("failed to load network config: %w", err)
	}

	var (
		typesAll     = []cldf_config_network.NetworkType{cldf_config_network.NetworkTypeTestnet, cldf_config_network.NetworkTypeMainnet}
		typesTestnet = []cldf_config_network.NetworkType{cldf_config_network.NetworkTypeTestnet}
		typesMainnet = []cldf_config_network.NetworkType{cldf_config_network.NetworkTypeMainnet}
	)

	// Filter networks based on the environment
	var networkTypes []cldf_config_network.NetworkType
	switch env {
	case cldf_environment.Local, cldf_environment.StagingTestnet, cldf_environment.ProdTestnet:
		networkTypes = typesTestnet
	case cldf_environment.StagingMainnet, cldf_environment.ProdMainnet:
		networkTypes = typesMainnet
	case cldf_environment.Prod:
		networkTypes = typesAll
	// The following environments are legacy environments that are used to support domains which
	// have not transitioned to the new environment structure.
	case cldf_environment.Testnet, cldf_environment.SolStaging:
		networkTypes = typesTestnet
	case cldf_environment.Staging:
		if domain.Key() == "data-streams" {
			networkTypes = typesAll
		} else {
			networkTypes = typesTestnet
		}
	case cldf_environment.Mainnet:
		networkTypes = typesMainnet
	default:
		lggr.Errorf("Unknown environment: %s", env)

		return nil, fmt.Errorf("unknown env: %s", env)
	}

	lggr.Infof("Loaded %s Networks for %s/%s", networkTypes, domain.Key(), env)

	return cfg.FilterWith(cldf_config_network.TypesFilter(networkTypes...)), nil
}

// loadNetworkConfig loads the network config from the .config directory in the given domain.
func loadNetworkConfig(domain domain.Domain) (*cldf_config_network.Config, error) {
	// Check if the .config directory exists in the domain
	configDir := filepath.Join(domain.DirPath(), ".config")
	if _, err := os.Stat(configDir); err != nil {
		return nil, fmt.Errorf("cannot find config directory: %w", err)
	}

	// Find all yaml config files in the .config directory and any subdirectories
	var configFiles []string

	yamlFiles, err := filepath.Glob(filepath.Join(configDir, "**", "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to find config files: %w", err)
	}
	configFiles = append(configFiles, yamlFiles...)

	ymlFiles, err := filepath.Glob(filepath.Join(configDir, "**", "*.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to find config files: %w", err)
	}
	configFiles = append(configFiles, ymlFiles...)

	if len(configFiles) == 0 {
		return nil, fmt.Errorf("no config files found in %s", configDir)
	}

	// If we are in CI, we want to use the GAP DNSs for the RPC URLs.
	// TODO: This is a temporary solution as this code will soon be moved to the framework. We
	// should take into account that reading the env var is not a great solution, as it can lead
	// to unexpected behavior based on an declared env var buried in code.
	//
	// Instead, we should consider moving the fetching of the env var into the commands that need
	// it, and determine the correct behavior based on the command.
	var loadOpts []cldf_config_network.LoadOption
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		loadOpts = append(loadOpts,
			cldf_config_network.WithHTTPURLTransformer(gapURLTransformer(GapRPCProxyHTTPPort)),
			cldf_config_network.WithWSURLTransformer(gapURLTransformer(GapRPCProxyWSPort)),
		)
	}

	// Load the config
	cfg, err := cldf_config_network.Load(configFiles, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load config files: %w", err)
	}

	return cfg, nil
}

// gapURLTransformer providers a URLTransformer which replaces the RPC host in a URI string with the GAP V2 proxy address.
func gapURLTransformer(port string) func(string) string {
	return func(uri string) string {
		return strings.Replace(uri, RpcsCldevSh, fmt.Sprintf("%s:%s", GapRPCProxy, port), 1)
	}
}

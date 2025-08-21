package network

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"

	cldf_config_network "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
)

// Config holds knobs that must stay private (hosts/ports, and whether to enable GAP transforms).
// In the public repo, you can accept a *Config from the private caller. If nil, no GAP transform.
type Config struct {
	// RPCsHost is the original host to be replaced when routing via GAP
	RPCsHost string
	// GapProxyHost is the GAP V2 proxy host
	GapProxyHost string
	// GapWSPort is the GAP V2 proxy WebSocket port
	GapWSPort string
	// GapHTTPPort is the GAP V2 proxy HTTP port
	GapHTTPPort string
	// UseGAP toggles URL transformation. Set this in the private repo
	UseGAP bool
}

// LoadNetworks retrieves the network configuration for the given domain and filters the networks
// according to the specified environment. The GAP/host settings are injected via cfg (may be nil).
func LoadNetworks(
	env string, d domain.Domain, lggr logger.Logger, cfg *Config,
) (*cldf_config_network.Config, error) {
	netCfg, err := loadNetworkConfig(d, cfg)
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
	case environment.Local, environment.StagingTestnet, environment.ProdTestnet:
		networkTypes = typesTestnet
	case environment.StagingMainnet, environment.ProdMainnet:
		networkTypes = typesMainnet
	case environment.Prod:
		networkTypes = typesAll
	// Legacy environments supporting domains not yet migrated to the new structure.
	case environment.Testnet, environment.SolStaging:
		networkTypes = typesTestnet
	case environment.Staging:
		if d.Key() == "data-streams" {
			networkTypes = typesAll
		} else {
			networkTypes = typesTestnet
		}
	case environment.Mainnet:
		networkTypes = typesMainnet
	default:
		lggr.Errorf("Unknown environment: %s", env)
		return nil, fmt.Errorf("unknown env: %s", env)
	}

	lggr.Infof("Loaded %s Networks for %s/%s", networkTypes, d.Key(), env)
	return netCfg.FilterWith(cldf_config_network.TypesFilter(networkTypes...)), nil
}

// loadNetworkConfig loads the network config from the .config directory in the given domain.
// GAP/host behavior is fully controlled by the injected cfg (nil => no GAP transforms).
func loadNetworkConfig(d domain.Domain, cfg *Config) (*cldf_config_network.Config, error) {
	// Check if the .config directory exists in the domain
	configDir := filepath.Join(d.DirPath(), ".config")
	if _, err := os.Stat(configDir); err != nil {
		return nil, fmt.Errorf("cannot find config directory: %w", err)
	}

	// Find all yaml/yml config files in the .config directory and any subdirectories
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

	// Decide on URL transformers based on injected cfg.
	var loadOpts []cldf_config_network.LoadOption
	if cfg != nil && cfg.UseGAP {
		// We transform both HTTP and WS URLs by replacing the original RPC host with GAP host:port.
		loadOpts = append(loadOpts,
			cldf_config_network.WithHTTPURLTransformer(gapURLTransformer(cfg.RPCsHost, cfg.GapProxyHost, cfg.GapHTTPPort)),
			cldf_config_network.WithWSURLTransformer(gapURLTransformer(cfg.RPCsHost, cfg.GapProxyHost, cfg.GapWSPort)),
		)
	}

	// Load the config
	netCfg, err := cldf_config_network.Load(configFiles, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load config files: %w", err)
	}

	return netCfg, nil
}

// gapURLTransformer returns a URLTransformer which replaces the RPC host in a URI string
// with the GAP V2 proxy address (host:port).
func gapURLTransformer(originalHost, proxyHost, port string) func(string) string {
	return func(uri string) string {
		return strings.Replace(uri, originalHost, fmt.Sprintf("%s:%s", proxyHost, port), 1)
	}
}

/*
USAGE:

const (
	rpcsHost      = "host.sh"
	gapProxyHost  = "gap.sh"
	gapWSPort     = "9333"
	gapHTTPPort   = "4444"
)

cfg := &environment.Config{
	RPCsHost:     rpcsHost,
	GapProxyHost: gapProxyHost,
	GapWSPort:    gapWSPort,
	GapHTTPPort:  gapHTTPPort,
	UseGAP:       true,
}

netCfg, err := environment.LoadNetworks(env, dom, lggr, cfg)

In local dev, pass nil or a Config with UseGAP=false to avoid any transformation.
*/

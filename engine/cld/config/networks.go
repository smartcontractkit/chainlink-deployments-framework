package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	cfgdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/domain"
	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network/rpcmanifest"
	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

// LoadNetworks retrieves the network configuration for the given domain and filters the networks
// according to the specified environment. This ensures that only networks relevant to the selected
// environment are accessible, minimizing the risk of accidental operations on unintended networks.
//
// When the domain config sets rpc_manifest_url, networks with no rpcs (or with rpcs_mode: append)
// get their RPCs from that manifest.
func LoadNetworks(
	env string, dom fdomain.Domain, lggr logger.Logger,
) (*cfgnet.Config, error) {
	domainConfigPath := filepath.Join(dom.ConfigDomainFilePath())
	if _, statErr := os.Stat(domainConfigPath); statErr != nil {
		return nil, fmt.Errorf("domain config not found at %s: %w", domainConfigPath, statErr)
	}

	domainConfig, err := cfgdomain.Load(domainConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load domain config: %w", err)
	}

	networkTypes, err := networkTypesForEnv(domainConfig, env)
	if err != nil {
		return nil, fmt.Errorf("failed to load domain config network types: %w", err)
	}

	cfg, err := loadNetworkConfig(dom)
	if err != nil {
		return nil, fmt.Errorf("failed to load network config: %w", err)
	}

	// Validate before filtering: a missing or mistyped type matches no filter and would otherwise
	// be dropped silently.
	if err = cfg.ValidateStructure(); err != nil {
		return nil, fmt.Errorf("failed to validate networks configuration: %w", err)
	}

	// Filter before resolving RPCs so networks outside this environment never need the manifest.
	cfg = cfg.FilterWith(cfgnet.TypesFilter(networkTypes...))

	cfg, err = resolveRPCs(context.Background(), cfg, domainConfig.RPCManifestURL, lggr)
	if err != nil {
		return nil, err
	}

	lggr.Infof("Loaded %s Networks for %s/%s", networkTypes, dom.Key(), env)

	return cfg, nil
}

// loadNetworkConfig loads the network config from the .config/networks directory in the given fdomain.
func loadNetworkConfig(domain fdomain.Domain) (*cfgnet.Config, error) {
	// Check if the .config/networks directory exists in the domain
	configNetworkDir := domain.ConfigNetworksDirPath()
	if _, err := os.Stat(configNetworkDir); err != nil {
		return nil, fmt.Errorf("cannot find config directory: %w", err)
	}

	// Find all yaml config files in the .config/networks directory
	var configFiles []string

	yamlFiles, err := filepath.Glob(filepath.Join(configNetworkDir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to find config files: %w", err)
	}
	configFiles = append(configFiles, yamlFiles...)

	ymlFiles, err := filepath.Glob(filepath.Join(configNetworkDir, "*.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to find config files: %w", err)
	}
	configFiles = append(configFiles, ymlFiles...)

	if len(configFiles) == 0 {
		return nil, fmt.Errorf("no config files found in %s", configNetworkDir)
	}

	// Validation is deferred to LoadNetworks: empty rpcs may still be filled from the RPC manifest.
	cfg, err := cfgnet.Load(configFiles, cfgnet.WithSkipValidation())
	if err != nil {
		return nil, fmt.Errorf("failed to load config files: %w", err)
	}

	return cfg, nil
}

// networkTypesForEnv returns the network types configured for the given environment.
func networkTypesForEnv(domainConfig *cfgdomain.DomainConfig, env string) ([]cfgnet.NetworkType, error) {
	envConfig, ok := domainConfig.Environments[env]
	if !ok {
		return nil, fmt.Errorf("environment %s not found in domain config", env)
	}

	networkTypes := make([]cfgnet.NetworkType, 0, len(envConfig.NetworkTypes))
	for _, networkType := range envConfig.NetworkTypes {
		networkTypes = append(networkTypes, cfgnet.NetworkType(networkType))
	}

	return networkTypes, nil
}

// resolveRPCs validates cfg and returns it with RPCs from the domain's RPC manifest merged in. The
// manifest is fetched only when it is configured and a network has no rpcs or appends to them.
// The manifest URL may carry credentials, so it is kept out of logs and errors.
func resolveRPCs(
	ctx context.Context, cfg *cfgnet.Config, manifestURL string, lggr logger.Logger,
) (*cfgnet.Config, error) {
	manifestURL = strings.TrimSpace(manifestURL)
	networks := cfg.Networks()

	appends := func(n cfgnet.Network) bool { return n.RPCsMode == cfgnet.RPCsModeAppend }
	usesManifest := func(n cfgnet.Network) bool { return len(n.RPCs) == 0 || appends(n) }

	if manifestURL == "" && slices.ContainsFunc(networks, appends) {
		lggr.Warn("networks set rpcs_mode: append but no rpc_manifest_url is configured; using their yaml rpcs only")
	}

	if manifestURL == "" || !slices.ContainsFunc(networks, usesManifest) {
		if err := cfg.Validate(); err != nil {
			return nil, fmt.Errorf("failed to validate networks configuration: %w", err)
		}

		return cfg, nil
	}

	manifest, err := rpcmanifest.Load(ctx, manifestURL, rpcmanifest.WithLogger(lggr))
	if err != nil {
		return nil, fmt.Errorf("failed to load rpc manifest: %w", err)
	}

	merged, err := rpcmanifest.MergeInto(cfg, manifest, lggr)
	if err != nil {
		return nil, fmt.Errorf("failed to merge rpc manifest: %w", err)
	}

	return merged, nil
}

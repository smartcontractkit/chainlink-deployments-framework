package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
func LoadNetworks(
	env string, dom fdomain.Domain, lggr logger.Logger,
) (*cfgnet.Config, error) {
	// Load network types from domain config first so a missing/invalid domain config fails fast,
	// before we potentially make a network call to fetch the RPC manifest below.
	domainConfigPath := filepath.Join(dom.ConfigDomainFilePath())
	if _, statErr := os.Stat(domainConfigPath); statErr != nil {
		return nil, fmt.Errorf("domain config not found at %s: %w", domainConfigPath, statErr)
	}

	networkTypes, err := loadDomainConfigNetworkTypes(env, dom)
	if err != nil {
		return nil, fmt.Errorf("failed to load domain config network types: %w", err)
	}

	cfg, err := loadNetworkConfig(dom)
	if err != nil {
		return nil, fmt.Errorf("failed to load network config: %w", err)
	}

	// Validate structural fields (type, chain_selector) on every loaded network before filtering
	// by environment type below. Otherwise a network with a missing/mistyped `type` would never
	// match any environment's TypesFilter and would be silently dropped instead of failing
	// validation — hiding a yaml misconfiguration rather than surfacing it.
	if err = cfg.ValidateStructure(); err != nil {
		return nil, fmt.Errorf("failed to validate networks configuration: %w", err)
	}

	// Filter to the environment's network types before merging/validating against the RPC
	// manifest, so a network of a type this environment doesn't use (e.g. a mainnet chain with
	// no rpcs in a testnet-only environment) can't trigger an unnecessary manifest fetch or fail
	// validation for a chain this environment will never touch.
	cfg = cfg.FilterWith(cfgnet.TypesFilter(networkTypes...))

	cfg, err = mergeNetworksFromManifest(context.Background(), cfg, lggr)
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
	for _, fp := range yamlFiles {
		if isGeneratedNetworkConfig(fp) {
			continue
		}
		configFiles = append(configFiles, fp)
	}

	ymlFiles, err := filepath.Glob(filepath.Join(configNetworkDir, "*.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to find config files: %w", err)
	}
	for _, fp := range ymlFiles {
		if isGeneratedNetworkConfig(fp) {
			continue
		}
		configFiles = append(configFiles, fp)
	}

	if len(configFiles) == 0 {
		return nil, fmt.Errorf("no config files found in %s", configNetworkDir)
	}

	// Validation is deferred until after mergeNetworksFromManifest runs, since a network with an
	// empty rpcs list is only invalid if the RPC manifest doesn't end up filling it in.
	cfg, err := cfgnet.Load(configFiles, cfgnet.WithSkipValidation())
	if err != nil {
		return nil, fmt.Errorf("failed to load config files: %w", err)
	}

	return cfg, nil
}

// loadDomainConfigNetworkTypes loads network types from domain config for the given environment.
func loadDomainConfigNetworkTypes(env string, dom fdomain.Domain) ([]cfgnet.NetworkType, error) {
	domainConfigPath := filepath.Join(dom.ConfigDomainFilePath())
	domainConfig, err := cfgdomain.Load(domainConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load domain config: %w", err)
	}

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

// loadRPCManifest loads the RPC proxy manifest from source.
// source may be an HTTP(S) URL, file:// URI, or local filesystem path.
func loadRPCManifest(ctx context.Context, source string, lggr logger.Logger) (*rpcmanifest.Manifest, error) {
	return rpcmanifest.Load(ctx, source, rpcmanifest.WithLogger(lggr))
}

// mergeNetworksFromManifest merges RPC data from the RPC manifest into any network whose yaml
// config has an empty rpcs list; every other field is left untouched. The manifest is only
// fetched if at least one loaded network needs it, so domains where every network fully defines
// its own RPCs never make a network call. The manifest source defaults to rpcmanifest.ManifestURL
// and can be overridden with CLD_RPC_MANIFEST_URL (e.g. to point at a local file for testing).
func mergeNetworksFromManifest(ctx context.Context, cfg *cfgnet.Config, lggr logger.Logger) (*cfgnet.Config, error) {
	if !anyNetworkMissingRPCs(cfg) {
		if err := cfg.Validate(); err != nil {
			return nil, fmt.Errorf("validate network config: %w", err)
		}

		return cfg, nil
	}

	manifestURL := os.Getenv(rpcmanifest.EnvRPCManifestURL)
	if manifestURL == "" {
		manifestURL = rpcmanifest.ManifestURL
	}

	manifest, err := loadRPCManifest(ctx, manifestURL, lggr)
	if err != nil {
		return nil, fmt.Errorf("failed to load rpc manifest from %s: %w", manifestURL, err)
	}

	merged, err := rpcmanifest.MergeInto(cfg, manifest, lggr)
	if err != nil {
		return nil, fmt.Errorf("failed to merge rpc manifest: %w", err)
	}

	lggr.Infof("Merged RPC manifest from %s into network config", manifestURL)

	return merged, nil
}

// anyNetworkMissingRPCs reports whether any network in cfg has no RPCs defined in yaml.
func anyNetworkMissingRPCs(cfg *cfgnet.Config) bool {
	for _, network := range cfg.Networks() {
		if len(network.RPCs) == 0 {
			return true
		}
	}

	return false
}

// isGeneratedNetworkConfig reports whether path is a `.merged.yaml`/`.merged.yml` file —
// generated output written by tooling (see rpcmanifest.WriteConfigYAML) that reflects networks
// yaml already merged with RPC manifest data. These are excluded from loadNetworkConfig's glob so
// a previously generated snapshot never gets re-loaded as if it were hand-authored source yaml.
func isGeneratedNetworkConfig(path string) bool {
	name := filepath.Base(path)

	return strings.HasSuffix(name, ".merged.yaml") || strings.HasSuffix(name, ".merged.yml")
}

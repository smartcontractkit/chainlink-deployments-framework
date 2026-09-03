package environment

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	chainselremote "github.com/smartcontractkit/chain-selectors/remote"
	"gopkg.in/yaml.v3"

	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/catalog"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/chains"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	cfgenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// BuildParams provides all the necessary parameters to build an environment.
//
// The onchain, catalog and network fields reuse the same types as the file
// based configuration, so their schemas match a domain's .config/local.yaml and
// .config/networks/*.yaml respectively.
type BuildParams struct {
	// Domain and Environment identify which slice of the catalog to load, and
	// Environment becomes the resulting Environment.Name.
	Domain      string `yaml:"domain"`
	Environment string `yaml:"environment"`

	// Onchain supplies the deployer keys and KMS settings used to load chains.
	Onchain cfgenv.OnchainConfig `yaml:"onchain"`
	// Catalog locates the catalog service that backs the datastore.
	Catalog cfgenv.CatalogConfig `yaml:"catalog"`

	// Networks lists the networks to load. Type is derived from the chain
	// selector when unset; when set it must agree with the selector.
	Networks []cfgnet.Network `yaml:"networks"`

	// DefaultRPCBaseURL is an optional RPC proxy base. When set, each network
	// gets a "<base>/<chain selector>" RPC prepended ahead of its own. Any
	// trailing slashes on the base are trimmed before the selector is appended.
	DefaultRPCBaseURL string `yaml:"default_rpc_base_url"`
}

// BuildFromYAML builds an environment from a YAML encoded BuildParams document.
//
// Decoding is strict: an unrecognised key is an error rather than a silently
// ignored field, so that a mistyped key cannot leave a zero value in its place.
//
// WARNING: BuildParams carries secrets (deployer keys, KMS identifiers, catalog
// credentials). Prefer sourcing those from the environment and supplying them on
// the BuildParams directly over committing them to a file. If you must use them
// as a file, ensure it is stored securely and access is restricted.
func BuildFromYAML(ctx context.Context, data []byte, opts ...BuildOption) (fdeployment.Environment, error) {
	params, err := parseBuildParams(data)
	if err != nil {
		return fdeployment.Environment{}, err
	}

	return Build(ctx, params, opts...)
}

// Build constructs an environment from supplied parameters, rather than from a
// domain directory on disk as Load does. no domain directory/config is read
// from disk: chains come from params.Networks and the datastore is loaded from
// the catalog service named in params.Catalog (chain loaders may still read
// files referenced by params.Onchain).
//
// The returned environment omits the offchain client, OCR secrets and CRE
// runner, and its address book is empty. This is suitable for MCMS Execution,
// but not for Changeset execution.
func Build(ctx context.Context, params BuildParams, opts ...BuildOption) (fdeployment.Environment, error) {
	buildcfg, err := newBuildConfig()
	if err != nil {
		return fdeployment.Environment{}, err
	}
	buildcfg.Configure(opts)

	var (
		lggr   = buildcfg.lggr
		getCtx = func() context.Context { return ctx }
	)

	netConfig, err := buildNetworksConfig(ctx, params.Networks, params.DefaultRPCBaseURL)
	if err != nil {
		return fdeployment.Environment{}, err
	}

	cfg := &config.Config{
		Networks: netConfig,
		Env: &cfgenv.Config{
			Onchain: params.Onchain,
			Catalog: params.Catalog,
		},
	}

	blockChains, err := chains.LoadChains(ctx, lggr, cfg, cfg.Networks.ChainSelectors())
	if err != nil {
		return fdeployment.Environment{}, err
	}

	catalogStore, err := catalog.LoadCatalog(ctx, params.Domain, params.Environment, params.Catalog)
	if err != nil {
		return fdeployment.Environment{}, err
	}
	ds, err := fdatastore.LoadDataStoreFromCatalog(ctx, catalogStore)
	if err != nil {
		return fdeployment.Environment{}, fmt.Errorf("failed to load data from catalog: %w", err)
	}

	return fdeployment.Environment{
		// Name matches what `Load` sets for the same environment.
		Name:        params.Environment,
		Logger:      lggr,
		GetContext:  getCtx,
		BlockChains: blockChains,
		DataStore:   ds,
		OperationsBundle: operations.NewBundle(
			getCtx,
			buildcfg.lggr,
			buildcfg.reporter,
			operations.WithOperationRegistry(
				buildcfg.operationRegistry,
			),
		),

		// WARNING: the address book is deprecated and is never populated here. It
		// exists only so that changesets which still reach for it do not panic.
		ExistingAddresses: fdeployment.NewMemoryAddressBook(),

		// WARNING: node IDs live only in flat files, so they cannot be populated
		// from parameters and must not be relied upon here.
		NodeIDs: []string{},

		// TODO: not needed at this stage. Required once this is used for
		// changeset execution.
		// 	Offchain:          oc,
		//  OCRSecrets:  nil,
		//  CRERunner:         loadcfg.creRunner,
	}, nil
}

// resolveNetworkType derives the network type from the chain selector. A type
// supplied by the caller is honoured, but must agree with the selector so that a
// mismatch is reported rather than silently corrected.
func resolveNetworkType(ctx context.Context, network cfgnet.Network) (cfgnet.NetworkType, error) {
	cd, err := chainselremote.GetChainDetailsBySelector(ctx, network.ChainSelector)
	if err != nil {
		return "", err
	}

	derived := cfgnet.NetworkType(cd.NetworkType)
	if network.Type != "" && network.Type != derived {
		return "", fmt.Errorf(
			"type %q does not match chain selector, which is %q", network.Type, derived,
		)
	}

	return derived, nil
}

// buildNetworksConfig converts the supplied networks into the cfgnet.Config
// that the chain loaders consume, deriving each network type and injecting the
// default RPC. Each network is validated once fully populated.
func buildNetworksConfig(ctx context.Context, nets []cfgnet.Network, rpcBaseURL string) (*cfgnet.Config, error) {
	networks := make([]cfgnet.Network, 0, len(nets))

	for _, network := range nets {
		networkType, err := resolveNetworkType(ctx, network)
		if err != nil {
			return nil, fmt.Errorf("network %d: %w", network.ChainSelector, err)
		}
		network.Type = networkType

		if rpcBaseURL != "" {
			defaultRPC := cfgnet.RPC{
				RPCName:            "default",
				PreferredURLScheme: "http",
				HTTPURL:            fmt.Sprintf("%s/%d", strings.TrimRight(rpcBaseURL, "/"), network.ChainSelector),
			}

			// Prepended, not appended: the chain loaders read RPCs[0] for every
			// non-EVM family, so the derived URL has to win. Copying into a fresh
			// slice also keeps the caller's backing array untouched.
			network.RPCs = append([]cfgnet.RPC{defaultRPC}, network.RPCs...)
		}

		// Validate after the default RPC is injected, since it may be the only one.
		if err := network.Validate(); err != nil {
			return nil, fmt.Errorf("network %d: %w", network.ChainSelector, err)
		}

		networks = append(networks, network)
	}

	return cfgnet.NewConfig(networks), nil
}

// parseBuildParams decodes a YAML encoded BuildParams document, rejecting keys
// that do not correspond to a field.
func parseBuildParams(data []byte) (BuildParams, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	var params BuildParams
	if err := dec.Decode(&params); err != nil {
		if errors.Is(err, io.EOF) {
			return BuildParams{}, errors.New("failed to decode build params: document is empty")
		}

		return BuildParams{}, fmt.Errorf("failed to decode build params: %w", err)
	}

	return params, nil
}

// BuildConfig holds the options applied to a Build, each with a default that
// the corresponding BuildWith* option overrides.
type BuildConfig struct {
	// reporter records operations performed by the environment.
	reporter operations.Reporter

	// operationRegistry holds the operations available to the environment.
	operationRegistry *operations.OperationRegistry

	// lggr is used throughout the build and by the resulting environment.
	lggr logger.Logger
}

// Configure applies the given options, overriding the defaults.
func (c *BuildConfig) Configure(opts []BuildOption) {
	for _, opt := range opts {
		opt(c)
	}
}

// BuildOption is a functional option type for configuring an environment build.
type BuildOption func(*BuildConfig)

// newBuildConfig returns a BuildConfig populated with defaults.
func newBuildConfig() (*BuildConfig, error) {
	lggr, err := logger.New()
	if err != nil {
		return nil, err
	}

	return &BuildConfig{
		reporter:          operations.NewMemoryReporter(),
		operationRegistry: operations.NewOperationRegistry(),
		lggr:              lggr,
	}, nil
}

// BuildWithOperationRegistry supplies a pre-populated operation registry.
// Defaults to an empty registry.
func BuildWithOperationRegistry(registry *operations.OperationRegistry) BuildOption {
	return func(o *BuildConfig) {
		o.operationRegistry = registry
	}
}

// BuildWithLogger supplies the logger used during the build and carried on the
// resulting environment. Defaults to a new logger.
func BuildWithLogger(lggr logger.Logger) BuildOption {
	return func(o *BuildConfig) {
		o.lggr = lggr
	}
}

// BuildWithReporter supplies the operations reporter. Defaults to an in-memory
// reporter.
func BuildWithReporter(reporter operations.Reporter) BuildOption {
	return func(o *BuildConfig) {
		o.reporter = reporter
	}
}

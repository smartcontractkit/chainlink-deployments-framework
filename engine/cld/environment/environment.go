package environment

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	cldf_datastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"

	cldf_engine_catalog "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/catalog"
	cldf_chains "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/chains"
	cldf_config "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	cldf_domain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	cldf_engine_offchain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/offchain"
	cldf_offchain "github.com/smartcontractkit/chainlink-deployments-framework/offchain"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
)

// LoadEnvironmentOptions contains configuration options for LoadEnvironment.
type LoadEnvironmentOptions struct {
	reporter             operations.Reporter
	migrationString      string
	withoutJD            bool
	chainSelectorsToLoad []uint64
	operationRegistry    *operations.OperationRegistry
	anvilKeyAsDeployer   bool
}

// LoadEnvironmentOption is a function that modifies LoadEnvironmentOptions.
type LoadEnvironmentOption func(*LoadEnvironmentOptions)

// WithAnvilKeyAsDeployer sets the private key of the forked environment to use the Anvil key as the deployer key.
func WithAnvilKeyAsDeployer() LoadEnvironmentOption {
	return func(o *LoadEnvironmentOptions) {
		o.anvilKeyAsDeployer = true
	}
}

// WithReporter sets the reporter for LoadEnvironment.
func WithReporter(reporter operations.Reporter) LoadEnvironmentOption {
	return func(o *LoadEnvironmentOptions) {
		o.reporter = reporter
	}
}

// WithoutJD will configure the environment to not load Job Distributor.
// By default, if option is not specified, Job Distributor is loaded.
// This is useful for migrations that do not require Job Distributor to be loaded.
// WARNING: This will set env.Offchain to nil. Ensure that you do not use env.Offchain.
func WithoutJD() LoadEnvironmentOption {
	return func(o *LoadEnvironmentOptions) {
		o.withoutJD = true
	}
}

// OnlyLoadChainsFor will configure the environment to load only the specified chains
// for the given migration key.
// By default, if option is not specified, all chains are loaded.
// This is useful for migrations that are only applicable to a subset of chains.
func OnlyLoadChainsFor(migrationKey string, chainsSelectors []uint64) LoadEnvironmentOption {
	return func(o *LoadEnvironmentOptions) {
		o.migrationString = migrationKey
		o.chainSelectorsToLoad = chainsSelectors
	}
}

// WithOperationRegistry will configure the bundle in environment to use the specified operation registry.
func WithOperationRegistry(registry *operations.OperationRegistry) LoadEnvironmentOption {
	return func(o *LoadEnvironmentOptions) {
		o.operationRegistry = registry
	}
}

func Load(
	getCtx func() context.Context,
	lggr logger.Logger,
	env string,
	domain cldf_domain.Domain,
	useRealBackends bool,
	opts ...LoadEnvironmentOption,
) (cldf.Environment, error) {
	// Default options
	options := &LoadEnvironmentOptions{
		reporter:          operations.NewMemoryReporter(),
		operationRegistry: operations.NewOperationRegistry(),
	}
	for _, opt := range opts {
		opt(options)
	}

	envdir := domain.EnvDir(env)

	config, err := cldf_config.Load(domain, env, lggr)
	if err != nil {
		return cldf.Environment{}, err
	}

	ab, err := envdir.AddressBook()
	if err != nil {
		return cldf.Environment{}, err
	}

	// Note: Currently, if no datastore is present in the envdir, no error is returned.
	// This behavior is temporary and will be updated to return an error once all environments
	// are guaranteed to have a datastore.
	ds, err := envdir.DataStore()
	if err != nil {
		lggr.Warn("Unable to load datastore, skipping")
	}

	addressesByChain, err := ab.Addresses()
	if err != nil {
		return cldf.Environment{}, err
	}

	// default - loads all chains
	chainSelectorsToLoad := slices.Collect(maps.Keys(addressesByChain))

	if options.migrationString != "" && len(options.chainSelectorsToLoad) > 0 {
		lggr.Infow("Override: loading migration chains", "migration", options.migrationString, "chains", options.chainSelectorsToLoad)
		chainSelectorsToLoad = options.chainSelectorsToLoad
	}

	blockChains, err := cldf_chains.LoadChains(getCtx(), lggr, config, chainSelectorsToLoad)
	if err != nil {
		return cldf.Environment{}, err
	}

	nodes, err := envdir.LoadNodes()
	if err != nil {
		return cldf.Environment{}, err
	}

	var jd cldf_offchain.Client
	if !options.withoutJD {
		jd, err = cldf_engine_offchain.LoadOffchainClient(
			getCtx(),
			domain,
			env,
			config.Env,
			lggr,
			useRealBackends,
		)
		if err != nil {
			return cldf.Environment{},
				fmt.Errorf("failed to load offchain client for environment %s: %w", env, err)
		}
	} else {
		lggr.Info("Override: skipping JD initialization")
	}

	lggr.Debugw("Loaded environment", "env", env, "addressBook", ab)

	sharedSecrets, err := cldf.GenerateSharedSecrets(
		config.Env.Offchain.OCR.XSigners, config.Env.Offchain.OCR.XProposers,
	)
	if err != nil {
		if errors.Is(err, cldf.ErrMnemonicRequired) {
			lggr.Warn("No OCR secrets found in environment, proceeding without them")
		} else {
			return cldf.Environment{}, err
		}
	}

	var catalogDataStore cldf_datastore.CatalogStore
	if config.Env.Catalog.GRPC != "" {
		lggr.Infow("Initializing Catalog client", "url", config.Env.Catalog.GRPC)
		catalogDataStore, err = cldf_engine_catalog.LoadCatalog(getCtx(), env, config, domain)
		if err != nil {
			return cldf.Environment{}, err
		}
	} else {
		lggr.Info("Skipping Catalog client initialization, no Catalog config found")
	}

	return cldf.Environment{
		Name:              env,
		Logger:            lggr,
		ExistingAddresses: ab,
		DataStore:         ds,
		NodeIDs:           nodes.Keys(),
		Offchain:          jd,
		GetContext:        getCtx,
		OCRSecrets:        sharedSecrets,
		OperationsBundle:  operations.NewBundle(getCtx, lggr, options.reporter, operations.WithOperationRegistry(options.operationRegistry)),
		BlockChains:       blockChains,
		Catalog:           catalogDataStore,
	}, nil
}

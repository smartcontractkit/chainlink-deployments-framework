package environment

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"

	fcatalog "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/catalog"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/chains"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/offchain"
	foffchain "github.com/smartcontractkit/chainlink-deployments-framework/offchain"
	focr "github.com/smartcontractkit/chainlink-deployments-framework/offchain/ocr"

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
	domain fdomain.Domain,
	useRealBackends bool,
	opts ...LoadEnvironmentOption,
) (fdeployment.Environment, error) {
	// Default options
	options := &LoadEnvironmentOptions{
		reporter:          operations.NewMemoryReporter(),
		operationRegistry: operations.NewOperationRegistry(),
	}
	for _, opt := range opts {
		opt(options)
	}

	envdir := domain.EnvDir(env)

	cfg, err := config.Load(domain, env, lggr)
	if err != nil {
		return fdeployment.Environment{}, err
	}

	ab, err := envdir.AddressBook()
	if err != nil {
		return fdeployment.Environment{}, err
	}

	// Note: Currently, if no datastore is present in the envdir, no error is returned.
	// This behavior is temporary and will be updated to return an error once all environments
	// are guaranteed to have a fdatastore.
	ds, err := envdir.DataStore()
	if err != nil {
		lggr.Warn("Unable to load datastore, skipping")
	}

	addressesByChain, err := ab.Addresses()
	if err != nil {
		return fdeployment.Environment{}, err
	}

	// default - loads all chains
	chainSelectorsToLoad := slices.Collect(maps.Keys(addressesByChain))

	if options.migrationString != "" && len(options.chainSelectorsToLoad) > 0 {
		lggr.Infow("Override: loading migration chains", "migration", options.migrationString, "chains", options.chainSelectorsToLoad)
		chainSelectorsToLoad = options.chainSelectorsToLoad
	}

	blockChains, err := chains.LoadChains(getCtx(), lggr, cfg, chainSelectorsToLoad)
	if err != nil {
		return fdeployment.Environment{}, err
	}

	nodes, err := envdir.LoadNodes()
	if err != nil {
		return fdeployment.Environment{}, err
	}

	var jd foffchain.Client
	if !options.withoutJD {
		jd, err = offchain.LoadOffchainClient(
			getCtx(),
			domain,
			env,
			cfg.Env,
			lggr,
			useRealBackends,
		)
		if err != nil {
			return fdeployment.Environment{},
				fmt.Errorf("failed to load offchain client for environment %s: %w", env, err)
		}
	} else {
		lggr.Info("Override: skipping JD initialization")
	}

	lggr.Debugw("Loaded environment", "env", env, "addressBook", ab)

	sharedSecrets, err := focr.GenerateSharedSecrets(
		cfg.Env.Offchain.OCR.XSigners, cfg.Env.Offchain.OCR.XProposers,
	)
	if err != nil {
		if errors.Is(err, focr.ErrMnemonicRequired) {
			lggr.Warn("No OCR secrets found in environment, proceeding without them")
		} else {
			return fdeployment.Environment{}, err
		}
	}

	var catalogDataStore fdatastore.CatalogStore
	if cfg.Env.Catalog.GRPC != "" {
		lggr.Infow("Initializing Catalog client", "url", cfg.Env.Catalog.GRPC)
		catalogDataStore, err = fcatalog.LoadCatalog(getCtx(), env, cfg, domain)
		if err != nil {
			return fdeployment.Environment{}, err
		}
	} else {
		lggr.Info("Skipping Catalog client initialization, no Catalog config found")
	}

	return fdeployment.Environment{
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

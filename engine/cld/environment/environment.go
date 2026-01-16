package environment

import (
	"context"
	"errors"
	"fmt"

	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	cldcatalog "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/catalog"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/chains"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	cfgdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/domain"
	clddomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/internal/credentials"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/offchain"
	foffchain "github.com/smartcontractkit/chainlink-deployments-framework/offchain"
	focr "github.com/smartcontractkit/chainlink-deployments-framework/offchain/ocr"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
)

func Load(
	ctx context.Context,
	domain clddomain.Domain,
	envKey string,
	opts ...LoadEnvironmentOption,
) (fdeployment.Environment, error) {
	loadcfg, err := newLoadConfig()
	if err != nil {
		return fdeployment.Environment{}, err
	}
	loadcfg.Configure(opts)

	var (
		lggr   = loadcfg.lggr
		envdir = domain.EnvDir(envKey)
	)

	cfg, err := config.Load(domain, envKey, lggr)
	if err != nil {
		return fdeployment.Environment{}, err
	}

	ab, err := envdir.AddressBook()
	if err != nil {
		return fdeployment.Environment{}, err
	}

	var ds fdatastore.DataStore

	if cfg.DatastoreType == cfgdomain.DatastoreTypeCatalog || cfg.DatastoreType == cfgdomain.DatastoreTypeAll {
		if cfg.Env.Catalog.GRPC != "" {
			lggr.Infow("Fetching data from Catalog", "url", cfg.Env.Catalog.GRPC)
			catalogStore, catalogErr := cldcatalog.LoadCatalog(ctx, envKey, cfg, domain)
			if catalogErr != nil {
				return fdeployment.Environment{}, catalogErr
			}

			// Load all data from the catalog into a local datastore
			// After this, all operations happen locally without remote calls
			ds, err = fdatastore.LoadDataStoreFromCatalog(ctx, catalogStore)
			if err != nil {
				return fdeployment.Environment{}, fmt.Errorf("failed to load data from catalog: %w", err)
			}
			lggr.Infow("Loaded catalog data into local datastore for deployment operations")
		} else {
			return fdeployment.Environment{}, fmt.Errorf("catalog GRPC endpoint is required when datastore location is set to '%s'", cfgdomain.DatastoreTypeCatalog)
		}
	} else {
		// Load datastore from file system (default behavior)
		ds, err = envdir.DataStore()
		if err != nil {
			return fdeployment.Environment{}, err
		}
		lggr.Infow("Using file-based datastore")
	}

	// default - loads all chains from the networks config
	chainSelectorsToLoad := cfg.Networks.ChainSelectors()

	if loadcfg.chainSelectorsToLoad != nil {
		lggr.Infow("Override: loading chains", "chains", loadcfg.chainSelectorsToLoad)
		chainSelectorsToLoad = loadcfg.chainSelectorsToLoad
	}

	blockChains, err := chains.LoadChains(ctx, lggr, cfg, chainSelectorsToLoad)
	if err != nil {
		return fdeployment.Environment{}, err
	}

	nodes, err := envdir.LoadNodes()
	if err != nil {
		return fdeployment.Environment{}, err
	}

	var jd foffchain.Client
	if !loadcfg.withoutJD {
		jd, err = offchain.LoadOffchainClient(
			ctx,
			domain,
			cfg.Env.Offchain.JobDistributor,
			offchain.WithLogger(lggr),
			offchain.WithDryRun(loadcfg.useDryRunJobDistributor),
			offchain.WithCredentials(credentials.GetCredsForEnv(envKey)),
		)
		if err != nil {
			if errors.Is(err, offchain.ErrEndpointsRequired) {
				lggr.Warn("Skipping JD initialization: gRPC endpoint is not set in config")
			} else {
				return fdeployment.Environment{},
					fmt.Errorf("failed to load offchain client for environment %s: %w", envKey, err)
			}
		}
	} else {
		lggr.Info("Override: skipping JD initialization")
	}

	lggr.Debugw("Loaded environment", "env", envKey, "addressBook", ab)

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

	exceptions, err := envdir.LoadExceptions()
	if err != nil {
		return fdeployment.Environment{}, fmt.Errorf("failed to load exceptions: %w", err)
	}

	getCtx := func() context.Context { return ctx }

	return fdeployment.Environment{
		Name:              envKey,
		Logger:            lggr,
		ExistingAddresses: ab,
		DataStore:         ds,
		NodeIDs:           nodes.Keys(),
		Offchain:          jd,
		GetContext:        getCtx,
		OCRSecrets:        sharedSecrets,
		OperationsBundle:  operations.NewBundle(getCtx, lggr, loadcfg.reporter, operations.WithOperationRegistry(loadcfg.operationRegistry)),
		BlockChains:       blockChains,
		Exceptions:        exceptions,
	}, nil
}

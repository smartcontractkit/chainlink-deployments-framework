package environment

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

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

func Load(
	ctx context.Context,
	domain fdomain.Domain,
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

	ds, err := envdir.DataStore()
	if err != nil {
		return fdeployment.Environment{}, err
	}

	var catalog fdatastore.CatalogStore
	if cfg.Env.Catalog.GRPC != "" {
		lggr.Infow("Initializing Catalog client", "url", cfg.Env.Catalog.GRPC)
		catalog, err = fcatalog.LoadCatalog(ctx, envKey, cfg, domain)
		if err != nil {
			return fdeployment.Environment{}, err
		}
	} else {
		lggr.Info("Skipping Catalog client initialization, no Catalog config found")
	}

	addressesByChain, err := ab.Addresses()
	if err != nil {
		return fdeployment.Environment{}, err
	}

	// default - loads all chains
	chainSelectorsToLoad := slices.Collect(maps.Keys(addressesByChain))

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
			envKey,
			cfg.Env,
			lggr,
			!loadcfg.useDryRunJobDistributor,
		)
		if err != nil {
			return fdeployment.Environment{},
				fmt.Errorf("failed to load offchain client for environment %s: %w", envKey, err)
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
		Catalog:           catalog,
	}, nil
}

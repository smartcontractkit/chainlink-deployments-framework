package contract

import (
	"context"
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldcatalog "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/catalog"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	cfgdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/domain"
	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// NetworkLoaderFunc loads network configuration for an environment.
type NetworkLoaderFunc func(env string, dom domain.Domain) (*cfgnet.Config, error)

// DataStoreLoadOptions configures how the datastore is loaded.
type DataStoreLoadOptions struct {
	// FromLocal, when true, always use local files (envdir.DataStore) and ignore domain config.
	// Use for local runs when you want to verify against local datastore only.
	FromLocal bool
}

// DataStoreLoaderFunc returns a datastore for the given env directory.
// When opts.FromLocal is false and domain uses catalog datastore, loads from the remote catalog (CI-friendly).
type DataStoreLoaderFunc func(ctx context.Context, envdir domain.EnvDir, lggr logger.Logger, opts DataStoreLoadOptions) (datastore.DataStore, error)

// Deps holds injectable dependencies.
type Deps struct {
	NetworkLoader   NetworkLoaderFunc
	DataStoreLoader DataStoreLoaderFunc
}

func (d *Deps) applyDefaults() {
	if d.NetworkLoader == nil {
		d.NetworkLoader = defaultNetworkLoader
	}
	if d.DataStoreLoader == nil {
		d.DataStoreLoader = defaultDataStoreLoader
	}
}

func defaultNetworkLoader(env string, dom domain.Domain) (*cfgnet.Config, error) {
	return config.LoadNetworks(env, dom, logger.Nop())
}

func defaultDataStoreLoader(ctx context.Context, envdir domain.EnvDir, lggr logger.Logger, opts DataStoreLoadOptions) (datastore.DataStore, error) {
	dom := domain.NewDomain(envdir.RootPath(), envdir.DomainKey())
	envKey := envdir.Key()

	cfg, err := config.Load(dom, envKey, lggr)
	if err != nil {
		if opts.FromLocal {
			lggr.Infow("Loading datastore from local files")
			return envdir.DataStore()
		}

		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if opts.FromLocal || cfg.DatastoreType == cfgdomain.DatastoreTypeFile {
		lggr.Infow("Loading datastore from local files")
		return envdir.DataStore()
	}

	if cfg.Env.Catalog.GRPC == "" {
		return nil, fmt.Errorf("catalog GRPC endpoint is required when datastore is set to %q", cfg.DatastoreType)
	}
	lggr.Infow("Loading datastore from catalog", "url", cfg.Env.Catalog.GRPC)
	catalogStore, err := cldcatalog.LoadCatalog(ctx, envKey, cfg, dom)
	if err != nil {
		return nil, fmt.Errorf("failed to load catalog: %w", err)
	}
	ds, err := datastore.LoadDataStoreFromCatalog(ctx, catalogStore)
	if err != nil {
		return nil, fmt.Errorf("failed to load data from catalog: %w", err)
	}
	lggr.Infow("Loaded datastore from catalog")

	return ds, nil
}

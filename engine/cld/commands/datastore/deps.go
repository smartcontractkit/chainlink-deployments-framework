// Package datastore provides CLI commands for datastore management operations.
package datastore

import (
	"context"

	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldcatalog "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/catalog"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// ConfigLoaderFunc loads the configuration for a given domain and environment.
type ConfigLoaderFunc func(dom domain.Domain, envKey string, lggr logger.Logger) (*config.Config, error)

// CatalogLoaderFunc loads the catalog store for a given environment.
type CatalogLoaderFunc func(ctx context.Context, envKey string, cfg *config.Config, dom domain.Domain) (fdatastore.CatalogStore, error)

// FileMergerFunc merges changeset datastore to local files.
type FileMergerFunc func(envDir domain.EnvDir, name, timestamp string) error

// CatalogMergerFunc merges changeset datastore to catalog.
type CatalogMergerFunc func(ctx context.Context, envDir domain.EnvDir, name, timestamp string, catalog fdatastore.CatalogStore) error

// CatalogSyncerFunc syncs the entire local datastore to catalog.
type CatalogSyncerFunc func(ctx context.Context, envDir domain.EnvDir, catalog fdatastore.CatalogStore) error

// defaultConfigLoader is the production implementation that loads config.
func defaultConfigLoader(dom domain.Domain, envKey string, lggr logger.Logger) (*config.Config, error) {
	return config.Load(dom, envKey, lggr)
}

// defaultCatalogLoader is the production implementation that loads catalog.
func defaultCatalogLoader(ctx context.Context, envKey string, cfg *config.Config, dom domain.Domain) (fdatastore.CatalogStore, error) {
	return cldcatalog.LoadCatalog(ctx, envKey, cfg, dom)
}

// defaultFileMerger is the production implementation that merges to files.
func defaultFileMerger(envDir domain.EnvDir, name, timestamp string) error {
	return envDir.MergeChangesetDataStore(name, timestamp)
}

// defaultCatalogMerger is the production implementation that merges to catalog.
func defaultCatalogMerger(ctx context.Context, envDir domain.EnvDir, name, timestamp string, catalog fdatastore.CatalogStore) error {
	return envDir.MergeChangesetDataStoreCatalog(ctx, name, timestamp, catalog)
}

// defaultCatalogSyncer is the production implementation that syncs to catalog.
func defaultCatalogSyncer(ctx context.Context, envDir domain.EnvDir, catalog fdatastore.CatalogStore) error {
	return envDir.SyncDataStoreToCatalog(ctx, catalog)
}

// Deps holds the injectable dependencies for datastore commands.
// All fields are optional; nil values will use production defaults.
type Deps struct {
	// ConfigLoader loads the configuration for a domain and environment.
	// Default: config.Load
	ConfigLoader ConfigLoaderFunc

	// CatalogLoader loads the catalog store.
	// Default: cldcatalog.LoadCatalog
	CatalogLoader CatalogLoaderFunc

	// FileMerger merges changeset datastore to local files.
	// Default: envDir.MergeChangesetDataStore
	FileMerger FileMergerFunc

	// CatalogMerger merges changeset datastore to catalog.
	// Default: envDir.MergeChangesetDataStoreCatalog
	CatalogMerger CatalogMergerFunc

	// CatalogSyncer syncs the entire local datastore to catalog.
	// Default: envDir.SyncDataStoreToCatalog
	CatalogSyncer CatalogSyncerFunc
}

// applyDefaults fills in nil dependencies with production defaults.
func (d *Deps) applyDefaults() {
	if d.ConfigLoader == nil {
		d.ConfigLoader = defaultConfigLoader
	}
	if d.CatalogLoader == nil {
		d.CatalogLoader = defaultCatalogLoader
	}
	if d.FileMerger == nil {
		d.FileMerger = defaultFileMerger
	}
	if d.CatalogMerger == nil {
		d.CatalogMerger = defaultCatalogMerger
	}
	if d.CatalogSyncer == nil {
		d.CatalogSyncer = defaultCatalogSyncer
	}
}

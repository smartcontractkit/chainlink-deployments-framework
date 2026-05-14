package catalog

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore/catalog/remote"
)

// Types and functions aliased (and delegated) here for backwards compatibility.

type CatalogClient = remote.CatalogClient //nolint:revive // renaming would be a breaking change

type CatalogDataStoreConfig = remote.CatalogDataStoreConfig //nolint:revive // renaming would be a breaking change

func NewCatalogDataStore(config CatalogDataStoreConfig) datastore.CatalogStore {
	return remote.NewCatalogDataStore(config)
}

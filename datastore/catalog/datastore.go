package catalog

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

type CatalogDataStoreConfig struct {
	Domain      string
	Environment string
	Client      CatalogClient
}

var _ datastore.CatalogStore = &CatalogDataStore{}

type CatalogDataStore struct {
	addressRefStore       *CatalogAddressRefStore
	chainMetadataStore    *CatalogChainMetadataStore
	contractMetadataStore *CatalogContractMetadataStore
	envMetadataStore      *CatalogEnvMetadataStore
}

func NewCatalogDataStore(config CatalogDataStoreConfig) *CatalogDataStore {
	return &CatalogDataStore{
		addressRefStore:       NewCatalogAddressRefStore(CatalogAddressRefStoreConfig(config)),
		chainMetadataStore:    NewCatalogChainMetadataStore(CatalogChainMetadataStoreConfig(config)),
		contractMetadataStore: NewCatalogContractMetadataStore(CatalogContractMetadataStoreConfig(config)),
		envMetadataStore:      NewCatalogEnvMetadataStore(CatalogEnvMetadataStoreConfig(config)),
	}
}

func (s *CatalogDataStore) Addresses() datastore.MutableRefStoreV2[datastore.AddressRefKey, datastore.AddressRef] {
	return s.addressRefStore
}

func (s *CatalogDataStore) ChainMetadata() datastore.MutableStoreV2[datastore.ChainMetadataKey, datastore.ChainMetadata] {
	return s.chainMetadataStore
}

func (s *CatalogDataStore) ContractMetadata() datastore.MutableStoreV2[datastore.ContractMetadataKey, datastore.ContractMetadata] {
	return s.contractMetadataStore
}

func (s *CatalogDataStore) EnvMetadata() datastore.MutableUnaryStoreV2[datastore.EnvMetadata] {
	return s.envMetadataStore
}

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
	AddressRefStore       *CatalogAddressRefStore
	ChainMetadataStore    *CatalogChainMetadataStore
	ContractMetadataStore *CatalogContractMetadataStore
	EnvMetadataStore      *CatalogEnvMetadataStore
}

func NewCatalogDataStore(config CatalogDataStoreConfig) *CatalogDataStore {
	return &CatalogDataStore{
		AddressRefStore:       NewCatalogAddressRefStore(CatalogAddressRefStoreConfig(config)),
		ChainMetadataStore:    NewCatalogChainMetadataStore(CatalogChainMetadataStoreConfig(config)),
		ContractMetadataStore: NewCatalogContractMetadataStore(CatalogContractMetadataStoreConfig(config)),
		EnvMetadataStore:      NewCatalogEnvMetadataStore(CatalogEnvMetadataStoreConfig(config)),
	}
}

func (s *CatalogDataStore) Addresses() datastore.MutableRefStoreV2[datastore.AddressRefKey, datastore.AddressRef] {
	return s.AddressRefStore
}

func (s *CatalogDataStore) ChainMetadata() datastore.MutableStoreV2[datastore.ChainMetadataKey, datastore.ChainMetadata] {
	return s.ChainMetadataStore
}

func (s *CatalogDataStore) ContractMetadata() datastore.MutableStoreV2[datastore.ContractMetadataKey, datastore.ContractMetadata] {
	return s.ContractMetadataStore
}

func (s *CatalogDataStore) EnvMetadata() datastore.MutableUnaryStoreV2[datastore.EnvMetadata] {
	return s.EnvMetadataStore
}

package catalog

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

type CatalogDataStoreConfig struct {
	Domain      string
	Environment string
	Client      CatalogClient
}

var _ datastore.CatalogStore = &catalogDataStore{}

type catalogDataStore struct {
	addressRefStore       *catalogAddressRefStore
	chainMetadataStore    *catalogChainMetadataStore
	contractMetadataStore *catalogContractMetadataStore
	envMetadataStore      *catalogEnvMetadataStore
}

func NewCatalogDataStore(config CatalogDataStoreConfig) *catalogDataStore {
	return &catalogDataStore{
		addressRefStore:       newCatalogAddressRefStore(catalogAddressRefStoreConfig(config)),
		chainMetadataStore:    newCatalogChainMetadataStore(catalogChainMetadataStoreConfig(config)),
		contractMetadataStore: newCatalogContractMetadataStore(catalogContractMetadataStoreConfig(config)),
		envMetadataStore:      newCatalogEnvMetadataStore(catalogEnvMetadataStoreConfig(config)),
	}
}

func (s *catalogDataStore) Addresses() datastore.MutableRefStoreV2[datastore.AddressRefKey, datastore.AddressRef] {
	return s.addressRefStore
}

func (s *catalogDataStore) ChainMetadata() datastore.MutableStoreV2[datastore.ChainMetadataKey, datastore.ChainMetadata] {
	return s.chainMetadataStore
}

func (s *catalogDataStore) ContractMetadata() datastore.MutableStoreV2[datastore.ContractMetadataKey, datastore.ContractMetadata] {
	return s.contractMetadataStore
}

func (s *catalogDataStore) EnvMetadata() datastore.MutableUnaryStoreV2[datastore.EnvMetadata] {
	return s.envMetadataStore
}

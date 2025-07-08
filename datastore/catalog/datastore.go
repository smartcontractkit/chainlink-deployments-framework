package catalog

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

type CatalogDataStoreConfig struct {
	Domain      string `json:"domain"`
	Environment string `json:"environment"`
}

var _ datastore.MutableDataStore = &CatalogDataStore{}

type CatalogDataStore struct {
	AddressRefStore       *CatalogAddressRefStore
	ChainMetadataStore    *CatalogChainMetadataStore
	ContractMetadataStore *CatalogContractMetadataStore
	EnvMetadataStore      *CatalogEnvMetadataStore

	domain      string
	environment string
}

func NewCatalogDataStore(config CatalogDataStoreConfig) *CatalogDataStore {
	return &CatalogDataStore{
		domain:      config.Domain,
		environment: config.Environment,

		AddressRefStore: NewCatalogAddressRefStore(
			CatalogAddressRefStoreConfig{
				Domain:      config.Domain,
				Environment: config.Environment,
			}),
		ChainMetadataStore: NewCatalogChainMetadataStore(
			CatalogChainMetadataStoreConfig{
				Domain:      config.Domain,
				Environment: config.Environment,
			},
		),
		ContractMetadataStore: NewCatalogContractMetadataStore(
			CatalogContractMetadataStoreConfig{
				Domain:      config.Domain,
				Environment: config.Environment,
			},
		),
		EnvMetadataStore: NewCatalogEnvMetadataStore(
			CatalogEnvMetadataStoreConfig{
				Domain:      config.Domain,
				Environment: config.Environment,
			},
		),
	}
}

func (s *CatalogDataStore) Addresses() datastore.MutableAddressRefStore {
	return s.AddressRefStore
}

func (s *CatalogDataStore) ChainMetadata() datastore.MutableChainMetadataStore {
	return s.ChainMetadataStore
}

func (s *CatalogDataStore) ContractMetadata() datastore.MutableContractMetadataStore {
	return s.ContractMetadataStore
}

func (s *CatalogDataStore) EnvMetadata() datastore.MutableEnvMetadataStore {
	return s.EnvMetadataStore
}

func (s *CatalogDataStore) Seal() datastore.DataStore {
	return &sealedCatalogDataStore{
		AddressRefStore:       s.AddressRefStore,
		ChainMetadataStore:    s.ChainMetadataStore,
		ContractMetadataStore: s.ContractMetadataStore,
		EnvMetadataStore:      s.EnvMetadataStore,
		domain:                s.domain,
		environment:           s.environment,
	}
}

func (s *CatalogDataStore) Merge(other datastore.DataStore) error {
	// enpty implementation for now
	return nil
}

var _ datastore.DataStore = &sealedCatalogDataStore{}

type sealedCatalogDataStore struct {
	AddressRefStore       *CatalogAddressRefStore
	ChainMetadataStore    *CatalogChainMetadataStore
	ContractMetadataStore *CatalogContractMetadataStore
	EnvMetadataStore      *CatalogEnvMetadataStore

	domain      string
	environment string
}

func (s *sealedCatalogDataStore) Addresses() datastore.AddressRefStore {
	return s.AddressRefStore
}

func (s *sealedCatalogDataStore) ChainMetadata() datastore.ChainMetadataStore {
	return s.ChainMetadataStore
}

func (s *sealedCatalogDataStore) ContractMetadata() datastore.ContractMetadataStore {
	return s.ContractMetadataStore
}

func (s *sealedCatalogDataStore) EnvMetadata() datastore.EnvMetadataStore {
	return s.EnvMetadataStore
}

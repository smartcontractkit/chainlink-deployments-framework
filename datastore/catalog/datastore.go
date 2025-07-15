package catalog

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	pb "github.com/smartcontractkit/chainlink-deployments-framework/datastore/catalog/internal/protos"
)

type CatalogDataStoreConfig struct {
	Domain      string                        `json:"domain"`
	Environment string                        `json:"environment"`
	Client      pb.DeploymentsDatastoreClient `json:"-"`
}

var _ datastore.CatalogStore = &CatalogDataStore{}

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

		AddressRefStore:       NewCatalogAddressRefStore(CatalogAddressRefStoreConfig(config)),
		ChainMetadataStore:    NewCatalogChainMetadataStore(CatalogChainMetadataStoreConfig(config)),
		ContractMetadataStore: NewCatalogContractMetadataStore(CatalogContractMetadataStoreConfig(config)),
		EnvMetadataStore:      NewCatalogEnvMetadataStore(CatalogEnvMetadataStoreConfig(config)),
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

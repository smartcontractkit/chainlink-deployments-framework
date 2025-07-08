package catalog

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

type CatalogContractMetadataStoreConfig struct {
	Domain      string `json:"domain"`
	Environment string `json:"environment"`
}

type CatalogContractMetadataStore struct {
	domain      string
	environment string
}

var _ datastore.ContractMetadataStore = &CatalogContractMetadataStore{}

var _ datastore.MutableContractMetadataStore = &CatalogContractMetadataStore{}

func NewCatalogContractMetadataStore(cfg CatalogContractMetadataStoreConfig) *CatalogContractMetadataStore {
	return &CatalogContractMetadataStore{
		domain:      cfg.Domain,
		environment: cfg.Environment,
	}
}

func (s *CatalogContractMetadataStore) Get(key datastore.ContractMetadataKey) (datastore.ContractMetadata, error) {
	// Implementation for fetching an ContractMetadata from the catalog
	return datastore.ContractMetadata{}, nil // Placeholder return
}

// Fetch returns a copy of all ContractMetadata in the catalog.
func (s *CatalogContractMetadataStore) Fetch() ([]datastore.ContractMetadata, error) {
	// Implementation for fetching all ContractMetadatas from the catalog
	return []datastore.ContractMetadata{}, nil // Placeholder return
}

// Filter returns a copy of all ContractMetadata in the catalog that match the provided filter.
// Filters are applied in the order they are provided.
// If no filters are provided, all records are returned.
func (s *CatalogContractMetadataStore) Filter(filters ...datastore.FilterFunc[datastore.ContractMetadataKey, datastore.ContractMetadata]) []datastore.ContractMetadata {
	// Implementation for filtering ContractMetadatas in the catalog
	records := []datastore.ContractMetadata{} // Placeholder for fetched records
	for _, filter := range filters {
		records = filter(records)
	}
	return records
}

func (s *CatalogContractMetadataStore) Add(record datastore.ContractMetadata) error {
	// Implementation for adding an ContractMetadata to the catalog
	return nil // Placeholder return
}

func (s *CatalogContractMetadataStore) Upsert(record datastore.ContractMetadata) error {
	// Implementation for upserting an ContractMetadata in the catalog
	return nil // Placeholder return
}

func (s *CatalogContractMetadataStore) Update(record datastore.ContractMetadata) error {
	// Implementation for updating an ContractMetadata in the catalog
	return nil // Placeholder return
}

func (s *CatalogContractMetadataStore) Delete(key datastore.ContractMetadataKey) error {
	// Implementation for deleting an ContractMetadata from the catalog
	return nil // Placeholder return
}

package catalog

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

type CatalogChainMetadataStoreConfig struct {
	Domain      string `json:"domain"`
	Environment string `json:"environment"`
}

type CatalogChainMetadataStore struct {
	domain      string
	environment string
}

var _ datastore.ChainMetadataStore = &CatalogChainMetadataStore{}

var _ datastore.MutableChainMetadataStore = &CatalogChainMetadataStore{}

func NewCatalogChainMetadataStore(cfg CatalogChainMetadataStoreConfig) *CatalogChainMetadataStore {
	return &CatalogChainMetadataStore{
		domain:      cfg.Domain,
		environment: cfg.Environment,
	}
}

func (s *CatalogChainMetadataStore) Get(key datastore.ChainMetadataKey) (datastore.ChainMetadata, error) {
	// Implementation for fetching an ChainMetadata from the catalog
	return datastore.ChainMetadata{}, nil // Placeholder return
}

// Fetch returns a copy of all ChainMetadata in the catalog.
func (s *CatalogChainMetadataStore) Fetch() ([]datastore.ChainMetadata, error) {
	// Implementation for fetching all ChainMetadatas from the catalog
	return []datastore.ChainMetadata{}, nil // Placeholder return
}

// Filter returns a copy of all ChainMetadata in the catalog that match the provided filter.
// Filters are applied in the order they are provided.
// If no filters are provided, all records are returned.
func (s *CatalogChainMetadataStore) Filter(filters ...datastore.FilterFunc[datastore.ChainMetadataKey, datastore.ChainMetadata]) []datastore.ChainMetadata {
	// Implementation for filtering ChainMetadatas in the catalog
	records := []datastore.ChainMetadata{} // Placeholder for fetched records
	for _, filter := range filters {
		records = filter(records)
	}
	return records
}

func (s *CatalogChainMetadataStore) Add(record datastore.ChainMetadata) error {
	// Implementation for adding an ChainMetadata to the catalog
	return nil // Placeholder return
}

func (s *CatalogChainMetadataStore) Upsert(record datastore.ChainMetadata) error {
	// Implementation for upserting an ChainMetadata in the catalog
	return nil // Placeholder return
}

func (s *CatalogChainMetadataStore) Update(record datastore.ChainMetadata) error {
	// Implementation for updating an ChainMetadata in the catalog
	return nil // Placeholder return
}

func (s *CatalogChainMetadataStore) Delete(key datastore.ChainMetadataKey) error {
	// Implementation for deleting an ChainMetadata from the catalog
	return nil // Placeholder return
}

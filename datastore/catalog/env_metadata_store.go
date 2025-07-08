package catalog

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

type CatalogEnvMetadataStoreConfig struct {
	Domain      string `json:"domain"`
	Environment string `json:"environment"`
}

type CatalogEnvMetadataStore struct {
	domain      string
	environment string
}

var _ datastore.EnvMetadataStore = &CatalogEnvMetadataStore{}

var _ datastore.MutableEnvMetadataStore = &CatalogEnvMetadataStore{}

// NewCatalogEnvMetadataStore creates a new CatalogEnvMetadataStore instance.
func NewCatalogEnvMetadataStore(cfg CatalogEnvMetadataStoreConfig) *CatalogEnvMetadataStore {
	return &CatalogEnvMetadataStore{
		domain:      cfg.Domain,
		environment: cfg.Environment,
	}
}

func (s *CatalogEnvMetadataStore) Get() (datastore.EnvMetadata, error) {
	// Implementation for fetching an EnvMetadata from the catalog
	return datastore.EnvMetadata{}, nil // Placeholder return
}

func (s *CatalogEnvMetadataStore) Set(record datastore.EnvMetadata) error {
	// Implementation for setting an EnvMetadata in the catalog
	return nil // Placeholder return
}

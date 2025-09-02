package catalog

import (
	"context"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore/catalog/remote"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	credentials "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/internal/credentials"
)

// LoadCatalog loads a catalog data store for the specified domain and environment.
func LoadCatalog(ctx context.Context, env string,
	config *config.Config, domain domain.Domain) (datastore.CatalogStore, error) {
	catalogClient, err := loadCatalogClient(ctx, env, config.Env.Catalog.GRPC)
	if err != nil {
		return nil, err
	}

	catalogDataStore := remote.NewCatalogDataStore(remote.CatalogDataStoreConfig{
		Domain:      domain.Key(),
		Environment: env,
		Client:      catalogClient,
	})

	return catalogDataStore, nil
}

// loadCatalogClient initializes a Catalogue client using the grpc and gap config.
func loadCatalogClient(
	ctx context.Context, env string, url string,
) (*remote.CatalogClient, error) {
	creds := credentials.GetCredsForEnv(env)

	client, err := remote.NewCatalogClient(ctx, remote.CatalogConfig{
		GRPC:  url,
		Creds: creds,
	})
	if err != nil {
		return nil, err
	}

	return client, nil
}

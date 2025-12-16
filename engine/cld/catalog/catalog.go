package catalog

import (
	"context"
	"strings"

	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	catalogremote "github.com/smartcontractkit/chainlink-deployments-framework/datastore/catalog/remote"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	cfgenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	credentials "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/internal/credentials"
)

// LoadCatalog loads a catalog data store for the specified domain and environment.
func LoadCatalog(ctx context.Context, env string,
	config *config.Config, domain domain.Domain) (fdatastore.CatalogStore, error) {
	catalogClient, err := loadCatalogClient(ctx, env, &config.Env.Catalog)
	if err != nil {
		return nil, err
	}

	catalogDatastore := catalogremote.NewCatalogDataStore(catalogremote.CatalogDataStoreConfig{
		Domain:      domain.Key(),
		Environment: env,
		Client:      catalogClient,
	})

	return catalogDatastore, nil
}

// loadCatalogClient initializes a Catalogue client using the grpc config.
func loadCatalogClient(
	ctx context.Context, env string, cfg *cfgenv.CatalogConfig,
) (*catalogremote.CatalogClient, error) {
	creds := credentials.GetCredsForEnv(env)

	catalogCfg := catalogremote.CatalogConfig{
		GRPC:  cfg.GRPC,
		Creds: creds,
	}

	// Configure HMAC authentication if KMS key is provided
	if cfg.Auth != nil && cfg.Auth.KMSKeyID != "" {
		// Extract authority from GRPC URL (hostname without port)
		authority := extractAuthority(cfg.GRPC)

		catalogCfg.HMACConfig = &catalogremote.HMACAuthConfig{
			KeyID:     cfg.Auth.KMSKeyID,
			KeyRegion: cfg.Auth.KMSKeyRegion,
			Authority: authority,
		}
	}

	client, err := catalogremote.NewCatalogClient(ctx, catalogCfg)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// extractAuthority extracts the authority (hostname without scheme and port) from a gRPC URL.
// Examples:
//   - "grpc.example.com:443" -> "grpc.example.com"
//   - "https://grpc.example.com:443" -> "grpc.example.com"
//   - "grpc.example.com" -> "grpc.example.com"
func extractAuthority(grpcURL string) string {
	// Remove scheme if present
	authority := strings.TrimPrefix(grpcURL, "https://")
	authority = strings.TrimPrefix(authority, "http://")

	// Remove port if present
	if idx := strings.LastIndex(authority, ":"); idx != -1 {
		authority = authority[:idx]
	}

	return authority
}

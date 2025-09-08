package catalog

import (
	"testing"

	"github.com/stretchr/testify/require"

	catalogremote "github.com/smartcontractkit/chainlink-deployments-framework/datastore/catalog/remote"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	cfgenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

func TestLoadCatalog(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		env     string
		config  *config.Config
		domain  domain.Domain
		wantErr string
	}{
		{
			name: "successful catalog loading",
			env:  "testnet",
			config: &config.Config{
				Env: &cfgenv.Config{
					Catalog: cfgenv.CatalogConfig{
						GRPC: "localhost:50051",
					},
				},
			},
			domain: domain.NewDomain("test-root", "test-domain"),
		},
		{
			name: "valid config with different grpc url",
			env:  "testnet",
			config: &config.Config{
				Env: &cfgenv.Config{
					Catalog: cfgenv.CatalogConfig{
						GRPC: "grpc.example.com:443",
					},
				},
			},
			domain: domain.NewDomain("test-root", "test-domain"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			result, err := LoadCatalog(ctx, tt.env, tt.config, tt.domain)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
				require.Nil(t, result)
			} else {
				// For successful cases, we expect the function to create a catalog store
				// even if it can't connect to the actual gRPC service
				require.NoError(t, err)
				require.NotNil(t, result)
			}
		})
	}
}

func TestLoadCatalogClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		env     string
		url     string
		wantErr string
	}{
		{
			name: "successful client creation with local env",
			env:  "local",
			url:  "localhost:50051",
		},
		{
			name: "successful client creation with non-local env",
			env:  "testnet",
			url:  "grpc.example.com:443",
		},
		{
			name: "empty url - should still create client",
			env:  "testnet",
			url:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			result, err := loadCatalogClient(ctx, tt.env, tt.url)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
				require.Nil(t, result)
			} else {
				// For successful cases, we expect a client to be created
				// even if it can't actually connect to the service
				require.NoError(t, err)
				require.NotNil(t, result)
				require.IsType(t, &catalogremote.CatalogClient{}, result)
			}
		})
	}
}

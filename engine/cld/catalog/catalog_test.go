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
		cfg     *cfgenv.CatalogConfig
		wantErr string
	}{
		{
			name: "successful client creation with local env",
			env:  "local",
			cfg: &cfgenv.CatalogConfig{
				GRPC: "localhost:50051",
			},
		},
		{
			name: "successful client creation with non-local env",
			env:  "testnet",
			cfg: &cfgenv.CatalogConfig{
				GRPC: "grpc.example.com:443",
			},
		},
		{
			name: "empty url - should still create client",
			env:  "testnet",
			cfg: &cfgenv.CatalogConfig{
				GRPC: "",
			},
		},
		{
			name: "client with HMAC authentication",
			env:  "testnet",
			cfg: &cfgenv.CatalogConfig{
				GRPC: "grpc.example.com:443",
				Auth: &cfgenv.CatalogAuthConfig{
					KMSKeyID:     "test-key-id",
					KMSKeyRegion: "us-west-2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			result, err := loadCatalogClient(ctx, tt.env, tt.cfg)

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

func TestExtractAuthority(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		grpcURL  string
		expected string
	}{
		{
			name:     "hostname with port",
			grpcURL:  "grpc.example.com:443",
			expected: "grpc.example.com",
		},
		{
			name:     "hostname without port",
			grpcURL:  "grpc.example.com",
			expected: "grpc.example.com",
		},
		{
			name:     "https scheme with port",
			grpcURL:  "https://grpc.example.com:443",
			expected: "grpc.example.com",
		},
		{
			name:     "http scheme with port",
			grpcURL:  "http://grpc.example.com:8080",
			expected: "grpc.example.com",
		},
		{
			name:     "localhost with port",
			grpcURL:  "localhost:50051",
			expected: "localhost",
		},
		{
			name:     "localhost without port",
			grpcURL:  "localhost",
			expected: "localhost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := extractAuthority(tt.grpcURL)
			require.Equal(t, tt.expected, result)
		})
	}
}

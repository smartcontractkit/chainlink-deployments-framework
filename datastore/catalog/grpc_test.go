package catalog_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore/catalog"
)

func TestNewCatalogClient_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		config        catalog.CatalogConfig
		expectError   bool
		errorContains string
	}{
		{
			name: "config_with_insecure_credentials",
			config: catalog.CatalogConfig{
				GRPC:  "localhost:9090",
				Creds: insecure.NewCredentials(),
			},
			expectError: false,
		},
		{
			name: "config_with_gap_token_only",
			config: catalog.CatalogConfig{
				GRPC:  "localhost:9090",
				Creds: insecure.NewCredentials(),
				GAP: &catalog.GAPConfig{
					Token:      "test-token",
					Repository: "",
				},
			},
			expectError: false,
		},
		{
			name: "config_with_gap_repository_only",
			config: catalog.CatalogConfig{
				GRPC:  "localhost:9090",
				Creds: insecure.NewCredentials(),
				GAP: &catalog.GAPConfig{
					Token:      "",
					Repository: "test-repo",
				},
			},
			expectError: false,
		},
		{
			name: "full_config",
			config: catalog.CatalogConfig{
				GRPC:  "localhost:9090",
				Creds: insecure.NewCredentials(),
				GAP: &catalog.GAPConfig{
					Token:      "test-token",
					Repository: "test-repo",
				},
			},
			expectError: false,
		},
		{
			name: "no_transport_credentials",
			config: catalog.CatalogConfig{
				GRPC: "localhost:9090",
				// No Creds field set
			},
			expectError:   true,
			errorContains: "no transport security set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Execute
			client, err := catalog.NewCatalogClient(tt.config)

			// Verify
			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorContains)
				require.Equal(t, catalog.CatalogClient{}, client)
			} else {
				require.NoError(t, err)
				require.NotNil(t, client)
				require.Equal(t, tt.config.GRPC, client.GRPC)
				require.NotNil(t, client.DeploymentsDatastoreClient)
			}
		})
	}
}

package remote_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"google.golang.org/grpc/credentials/insecure"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore/catalog/remote"
)

func TestNewCatalogClient_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		config        remote.CatalogConfig
		expectError   bool
		errorContains string
	}{
		{
			name: "config_with_insecure_credentials",
			config: remote.CatalogConfig{
				GRPC:  "localhost:9090",
				Creds: insecure.NewCredentials(),
			},
			expectError: false,
		},
		{
			name: "no_transport_credentials",
			config: remote.CatalogConfig{
				GRPC: "localhost:9090",
				// No Creds field set
			},
			expectError:   true,
			errorContains: "no transport security set",
		},
		{
			name: "config_with_hmac_auth",
			config: remote.CatalogConfig{
				GRPC:  "localhost:9090",
				Creds: insecure.NewCredentials(),
				HMACConfig: &remote.HMACAuthConfig{
					KeyID:     "test-key-id",
					KeyRegion: "us-west-2",
					Authority: "catalog.example.com",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Execute
			client, err := remote.NewCatalogClient(t.Context(), tt.config)

			// Verify
			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorContains)
				require.Nil(t, client)
			} else {
				require.NoError(t, err)
				require.NotNil(t, client)
			}
		})
	}
}

func TestCatalogClient_Close(t *testing.T) {
	t.Parallel()

	client, err := remote.NewCatalogClient(t.Context(), remote.CatalogConfig{
		GRPC:  "localhost:9090",
		Creds: insecure.NewCredentials(),
	})

	require.NoError(t, err)
	require.NotNil(t, client)

	require.NoError(t, client.Close())
}

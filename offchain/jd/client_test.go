package jd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/credentials/insecure"
)

func TestNewJDClient_ConfigurationScenarios(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      JDConfig
		description string
	}{
		{
			name: "basic config with credentials",
			config: JDConfig{
				GRPC:  "localhost:9090",
				Creds: insecure.NewCredentials(),
			},
			description: "Basic configuration with insecure credentials",
		},
		{
			name: "config with WSRPC",
			config: JDConfig{
				GRPC:  "localhost:9090",
				WSRPC: "ws://localhost:9091",
				Creds: insecure.NewCredentials(),
			},
			description: "Configuration with WebSocket RPC endpoint",
		},
		{
			name: "config with OAuth2 auth",
			config: JDConfig{
				GRPC:  "localhost:9090",
				Creds: insecure.NewCredentials(),
				Auth:  oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
			},
			description: "Configuration with OAuth2 authentication",
		},

		{
			name: "complete config",
			config: JDConfig{
				GRPC:  "localhost:9090",
				WSRPC: "ws://localhost:9091",
				Creds: insecure.NewCredentials(),
				Auth:  oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
			},
			description: "Complete configuration with all options",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, err := NewJDClient(tt.config)

			// gRPC connection creation typically succeeds even without server
			// The actual connection failure happens on first RPC call
			if err != nil {
				t.Logf("Connection failed for %s: %v", tt.description, err)
				assert.Contains(t, err.Error(), "failed to connect Job Distributor service")
			} else {
				require.NotNil(t, client, "Client should not be nil for %s", tt.description)

				// Verify fields are set correctly
				assert.Equal(t, tt.config.WSRPC, client.WSRPC)
				assert.NotNil(t, client.NodeServiceClient)
				assert.NotNil(t, client.JobServiceClient)
				assert.NotNil(t, client.CSAServiceClient)
			}
		})
	}
}

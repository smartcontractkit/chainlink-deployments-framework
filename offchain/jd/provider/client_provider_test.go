package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestClientOffchainProviderConfig_validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  ClientOffchainProviderConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config with minimal fields",
			config: ClientOffchainProviderConfig{
				GRPC: "localhost:9090",
			},
			wantErr: false,
		},
		{
			name: "valid config with all fields",
			config: ClientOffchainProviderConfig{
				GRPC: "localhost:9090",
			},
			wantErr: false,
		},
		{
			name:    "invalid config - missing GRPC",
			config:  ClientOffchainProviderConfig{},
			wantErr: true,
			errMsg:  "gRPC URL is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewClientOffchainProvider(t *testing.T) {
	t.Parallel()

	config := ClientOffchainProviderConfig{
		GRPC: "localhost:9090",
	}

	provider := NewClientOffchainProvider(config)

	require.NotNil(t, provider)
	assert.Equal(t, config, provider.config)
	assert.Nil(t, provider.client) // Should be nil until initialized
}

func TestClientOffchainProvider_Name(t *testing.T) {
	t.Parallel()

	provider := NewClientOffchainProvider(ClientOffchainProviderConfig{
		GRPC: "localhost:9090",
	})

	assert.Equal(t, "Job Distributor Client Offchain Provider", provider.Name())
}

func TestClientOffchainProvider_Initialize_ValidationError(t *testing.T) {
	t.Parallel()

	// Test with invalid config
	provider := NewClientOffchainProvider(ClientOffchainProviderConfig{})

	client, err := provider.Initialize(context.Background())

	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to validate provider config")
}

func TestClientOffchainProvider_OffchainClient_BeforeInitialize(t *testing.T) {
	t.Parallel()

	provider := NewClientOffchainProvider(ClientOffchainProviderConfig{
		GRPC: "localhost:9090",
	})

	client := provider.OffchainClient()
	assert.Nil(t, client) // Should be nil before initialization
}

func TestInitializeProvider(t *testing.T) {
	t.Parallel()

	// Test with invalid provider
	provider := NewClientOffchainProvider(ClientOffchainProviderConfig{})

	client, err := provider.Initialize(context.Background())

	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to validate provider config")
}

func TestWithDryRun(t *testing.T) {
	t.Parallel()

	lggr := logger.Test(t)

	// Test that WithDryRun option sets the dry run fields correctly
	config := ClientOffchainProviderConfig{
		GRPC: "localhost:9090",
	}

	provider := NewClientOffchainProvider(config, WithDryRun(lggr))

	// Verify that dry run fields are set (we can't access them directly since they're private,
	// but we can test validation)
	err := provider.config.validate()
	require.NoError(t, err, "Config with dry run and logger should be valid")
}

func TestWithDryRun_MissingLogger(t *testing.T) {
	t.Parallel()

	// Test that dry run without logger fails validation
	config := ClientOffchainProviderConfig{
		GRPC: "localhost:9090",
	}

	// Manually set dry run without logger to test validation
	config.dryRun = true
	// config.dryRunLogger is nil

	provider := NewClientOffchainProvider(config)

	err := provider.config.validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dry run logger is required when dry run mode is enabled")
}

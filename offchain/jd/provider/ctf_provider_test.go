package provider

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCTFOffchainProviderConfig_validate(t *testing.T) { //nolint:paralleltest // Cannot run in parallel due to global env var manipulation
	// Save original env var and restore at the end
	originalEnv := os.Getenv("CTF_JD_IMAGE")
	defer func() {
		if originalEnv != "" {
			os.Setenv("CTF_JD_IMAGE", originalEnv)
		} else {
			os.Unsetenv("CTF_JD_IMAGE")
		}
	}()

	tests := []struct {
		name    string
		config  CTFOffchainProviderConfig
		envVar  string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config with Image provided",
			config: CTFOffchainProviderConfig{
				Image: "my-jd-image:latest",
			},
			wantErr: false,
		},
		{
			name:    "valid config with CTF_JD_IMAGE env var",
			config:  CTFOffchainProviderConfig{},
			envVar:  "env-jd-image:latest",
			wantErr: false,
		},
		{
			name:    "missing both Image and CTF_JD_IMAGE",
			config:  CTFOffchainProviderConfig{},
			wantErr: true,
			errMsg:  "either Image must be provided in config or CTF_JD_IMAGE environment variable must be set",
		},
		{
			name: "both Image and CTF_JD_IMAGE provided (should prefer config)",
			config: CTFOffchainProviderConfig{
				Image: "config-jd-image:latest",
			},
			envVar:  "env-jd-image:latest",
			wantErr: false,
		},
	}

	for _, tt := range tests { //nolint:paralleltest // Cannot run subtests in parallel due to global env var manipulation
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable if provided
			if tt.envVar != "" {
				os.Setenv("CTF_JD_IMAGE", tt.envVar)
			} else {
				os.Unsetenv("CTF_JD_IMAGE")
			}

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

func TestNewCTFOffchainProvider(t *testing.T) {
	t.Parallel()

	config := CTFOffchainProviderConfig{
		Image: "test-jd-image:latest",
	}

	provider := NewCTFOffchainProvider(t, config)

	assert.NotNil(t, provider)
	assert.Equal(t, config, provider.config)
	assert.Equal(t, t, provider.t)
	assert.Nil(t, provider.client) // Should be nil until initialized
}

func TestCTFOffchainProvider_Name(t *testing.T) {
	t.Parallel()

	config := CTFOffchainProviderConfig{
		Image: "test-jd-image:latest",
	}
	provider := NewCTFOffchainProvider(t, config)

	expectedName := "Job Distributor CTF Offchain Provider"
	assert.Equal(t, expectedName, provider.Name())
}

func TestCTFOffchainProvider_OffchainClient_BeforeInitialize(t *testing.T) {
	t.Parallel()

	config := CTFOffchainProviderConfig{
		Image: "test-jd-image:latest",
	}
	provider := NewCTFOffchainProvider(t, config)

	// Should return nil before initialization
	assert.Nil(t, provider.OffchainClient())
}

func TestCTFOffchainProvider_Initialize_ValidationError(t *testing.T) {
	t.Parallel()

	// Test that Initialize properly validates configuration
	// Clear any existing CTF_JD_IMAGE environment variable
	os.Unsetenv("CTF_JD_IMAGE")

	config := CTFOffchainProviderConfig{
		// Missing both Image field and CTF_JD_IMAGE env var
	}
	provider := NewCTFOffchainProvider(t, config)

	ctx := context.Background()
	client, err := provider.Initialize(ctx)

	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "either Image must be provided in config or CTF_JD_IMAGE environment variable must be set")
}

// TestCTFOffchainProvider_CustomConfiguration tests that custom configuration is properly handled
func TestCTFOffchainProvider_CustomConfiguration(t *testing.T) {
	t.Parallel()

	config := CTFOffchainProviderConfig{
		Image:            "custom-jd-image:latest",
		GRPCPort:         "15000",
		WSRPCPort:        "9000",
		CSAEncryptionKey: "custom-key-123",
		DockerFilePath:   "./custom/Dockerfile",
		DockerContext:    "./custom/context",
		JDSQLDumpPath:    "./custom/dump.sql",
	}

	provider := NewCTFOffchainProvider(t, config)

	assert.Equal(t, "custom-jd-image:latest", provider.config.Image)
	assert.Equal(t, "15000", provider.config.GRPCPort)
	assert.Equal(t, "9000", provider.config.WSRPCPort)
	assert.Equal(t, "custom-key-123", provider.config.CSAEncryptionKey)
	assert.Equal(t, "./custom/Dockerfile", provider.config.DockerFilePath)
	assert.Equal(t, "./custom/context", provider.config.DockerContext)
	assert.Equal(t, "./custom/dump.sql", provider.config.JDSQLDumpPath)
}

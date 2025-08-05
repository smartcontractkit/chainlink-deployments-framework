package network

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_AnvilConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  AnvilConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: AnvilConfig{
				Image:          "anvil:latest",
				Port:           8545,
				ArchiveHTTPURL: "https://archive.example.com",
			},
			wantErr: false,
		},
		{
			name: "missing image",
			config: AnvilConfig{
				Port:           8545,
				ArchiveHTTPURL: "https://archive.example.com",
			},
			wantErr: true,
		},
		{
			name: "missing port",
			config: AnvilConfig{
				Image:          "anvil:latest",
				ArchiveHTTPURL: "https://archive.example.com",
			},
			wantErr: true,
		},
		{
			name: "missing archive HTTP URL",
			config: AnvilConfig{
				Image: "anvil:latest",
				Port:  8545,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDecodeMetadata(t *testing.T) {
	t.Parallel()

	// Define test structs
	type SimpleStruct struct {
		Name    string `yaml:"name"`
		Value   int    `yaml:"value"`
		LongInt uint64 `yaml:"long_int"`
	}

	longInt := uint64(13264668187771770619)

	tests := []struct {
		name          string
		metadata      any
		wantErr       bool
		expectedValue any
	}{
		{
			name: "successful conversion to EVMMetadata",
			metadata: map[string]any{
				"anvil_config": map[string]any{
					"image":            "my/image:latest",
					"port":             8545,
					"archive_http_url": "https://eth-mainnet.g.alchemy.com/v2/demo",
				},
			},
			wantErr: false,
			expectedValue: EVMMetadata{
				AnvilConfig: &AnvilConfig{
					Image:          "my/image:latest",
					Port:           8545,
					ArchiveHTTPURL: "https://eth-mainnet.g.alchemy.com/v2/demo",
				},
			},
		},
		{
			name: "successful conversion to SimpleStruct",
			metadata: map[string]any{
				"name":     "test",
				"value":    42,
				"long_int": longInt,
			},
			wantErr: false,
			expectedValue: SimpleStruct{
				Name:    "test",
				Value:   42,
				LongInt: longInt,
			},
		},
		{
			name: "partial metadata conversion",
			metadata: map[string]any{
				"name": "partial",
				// value field missing
			},
			wantErr: false,
			expectedValue: SimpleStruct{
				Name:  "partial",
				Value: 0, // zero value
			},
		},
		{
			name:     "nil metadata",
			metadata: nil,
			wantErr:  true,
		},
		{
			name: "type mismatch in metadata",
			metadata: map[string]any{
				"name":  123,            // should be string
				"value": "not_a_number", // should be int
			},
			wantErr: true,
		},
		{
			name:     "empty metadata map",
			metadata: map[string]any{},
			wantErr:  false,
			expectedValue: SimpleStruct{
				Name:  "",
				Value: 0,
			},
		},
		{
			name: "complex nested structure",
			metadata: map[string]any{
				"anvil_config": map[string]any{
					"image": "custom/image:v1.0",
					"port":  9545,
				},
			},
			wantErr: false,
			expectedValue: EVMMetadata{
				AnvilConfig: &AnvilConfig{
					Image:          "custom/image:v1.0",
					Port:           9545,
					ArchiveHTTPURL: "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			switch tt.expectedValue.(type) {
			case EVMMetadata:
				assertStruct[EVMMetadata](t, tt.metadata, tt.expectedValue, tt.wantErr)
			case SimpleStruct:
				assertStruct[SimpleStruct](t, tt.metadata, tt.expectedValue, tt.wantErr)
			}
		})
	}
}

func TestDecodeMetadata_fromYaml(t *testing.T) {
	t.Parallel()

	// Create temporary test files
	tmpDir := t.TempDir()

	// Valid YAML content for first file
	yamlContent1 := `
networks:
- type: "mainnet"
  chain_selector: 1
  metadata:
    anvil_config:
      image: "my/image:latest"
      port: 8545
      archive_http_url: "https://test.url"
  block_explorer:
    type: "Etherscan"
    api_key: "test_key"
    url: "https://etherscan.io"
  rpcs:
  - rpc_name: "test_rpc"
    preferred_url_scheme: "http"
    http_url: "https://test.rpc"
    ws_url: "wss://test.rpc"
  - rpc_name: "test_rpc2"
    preferred_url_scheme: "http"
    http_url: "https://test2.rpc"
    ws_url: "wss://test2.rpc"`

	tmpFile1 := filepath.Join(tmpDir, "test1.yaml")
	err := os.WriteFile(tmpFile1, []byte(yamlContent1), 0600)
	require.NoError(t, err, "Failed to create test file")

	// Expected EVMMetadata for the test
	expectedEVMMetadata := EVMMetadata{
		AnvilConfig: &AnvilConfig{
			Image:          "my/image:latest",
			Port:           8545,
			ArchiveHTTPURL: "https://test.url",
		},
	}

	// Load config from yaml
	manifest, err := Load([]string{tmpFile1})
	require.NoError(t, err, "Failed to load network manifest from YAML")

	// Convert metadata to EVMMetadata
	assertStruct[EVMMetadata](t, manifest.networks[1].Metadata, expectedEVMMetadata, false)
}

// assertStruct is a helper function that evaluates if a struct matches expected values
func assertStruct[T any](t *testing.T, metadata, expectedValue any, wantErr bool) {
	t.Helper()

	result, err := DecodeMetadata[T](metadata)
	if wantErr {
		require.Error(t, err)
	} else {
		require.NoError(t, err)
		assert.Equal(t, expectedValue, result)
	}
}

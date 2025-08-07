package network

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func Test_Config_MarshalYAML(t *testing.T) {
	t.Parallel()

	networks := []Network{
		{
			Type:          NetworkTypeMainnet,
			ChainSelector: 1,
			BlockExplorer: BlockExplorer{
				Type:   "Etherscan",
				APIKey: "test_key",
				URL:    "https://etherscan.io",
			},
			RPCs: []RPC{
				{
					RPCName:            "test_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "https://test.rpc",
					WSURL:              "wss://test.rpc",
				},
			},
			Metadata: map[string]any{
				"test": "test",
			},
		},
	}

	cfg := NewConfig(networks)

	yaml, err := yaml.Marshal(cfg)
	require.NoError(t, err)

	want := `networks:
    - type: mainnet
      chain_selector: 1
      block_explorer:
        type: Etherscan
        api_key: test_key
        url: https://etherscan.io
      rpcs:
        - rpc_name: test_rpc
          preferred_url_scheme: http
          http_url: https://test.rpc
          ws_url: wss://test.rpc
      metadata:
        test: test
`

	assert.YAMLEq(t, want, string(yaml))
}

func Test_Config_UnmarshalYAML(t *testing.T) {
	t.Parallel()

	give := `
networks:
  - type: mainnet
    chain_selector: 1
    block_explorer:
      type: Etherscan
      api_key: test_key
      url: https://etherscan.io
    rpcs:
      - rpc_name: test_rpc
        preferred_url_scheme: http
        http_url: https://test.rpc
        ws_url: wss://test.rpc
    metadata:
      test: test
`

	var cfg Config

	err := yaml.Unmarshal([]byte(give), &cfg)
	require.NoError(t, err)

	assert.Equal(t, Config{
		networks: map[uint64]Network{
			1: {
				Type:          NetworkTypeMainnet,
				ChainSelector: 1,
				BlockExplorer: BlockExplorer{
					Type:   "Etherscan",
					APIKey: "test_key",
					URL:    "https://etherscan.io",
				},
				RPCs: []RPC{
					{
						RPCName:            "test_rpc",
						PreferredURLScheme: "http",
						HTTPURL:            "https://test.rpc",
						WSURL:              "wss://test.rpc",
					},
				},
				Metadata: map[string]any{
					"test": "test",
				},
			},
		},
	}, cfg)

	// Failure case: invalid metadata
	err = yaml.Unmarshal([]byte("invalid"), &cfg)
	require.Error(t, err)
}

func Test_Config_GetBySelector(t *testing.T) {
	t.Parallel()

	network := Network{
		Type:          NetworkTypeMainnet,
		ChainSelector: 1,
		RPCs: []RPC{
			{
				RPCName:            "test_rpc",
				PreferredURLScheme: "http",
				HTTPURL:            "https://test.rpc",
				WSURL:              "wss://test.rpc",
			},
		},
	}

	cfg := NewConfig([]Network{network})

	tests := []struct {
		name         string
		giveSelector uint64
		want         Network
		wantErr      string
	}{
		{
			name:         "valid selector",
			giveSelector: 1,
			want:         network,
		},
		{
			name:         "invalid selector",
			giveSelector: 2,
			wantErr:      "network with selector 2 not found in configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := cfg.NetworkBySelector(tt.giveSelector)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_Config_ChainSelectors(t *testing.T) {
	t.Parallel()

	cfg := NewConfig([]Network{
		{ChainSelector: 1},
		{ChainSelector: 2},
	})

	expectedSelectors := []uint64{1, 2}
	selectors := cfg.ChainSelectors()

	assert.ElementsMatch(t, expectedSelectors, selectors)
}

func Test_Config_Merge(t *testing.T) {
	t.Parallel()

	cfg := NewConfig([]Network{
		{ChainSelector: 1},
	})

	cfg.Merge(NewConfig([]Network{
		{ChainSelector: 2},
	}))

	assert.Equal(t, &Config{
		networks: map[uint64]Network{
			1: {ChainSelector: 1},
			2: {ChainSelector: 2},
		},
	}, cfg)
}

func Test_Config_FilterWith(t *testing.T) {
	t.Parallel()

	// Create test networks
	mainnetNetwork1 := Network{
		Type:          "mainnet",
		ChainSelector: chain_selectors.ETHEREUM_MAINNET.Selector,
		RPCs: []RPC{
			{
				RPCName:            "test_rpc",
				PreferredURLScheme: "http",
				HTTPURL:            "https://test.rpc",
				WSURL:              "wss://test.rpc",
			},
		},
	}

	mainnetNetwork2 := Network{
		Type:          "mainnet",
		ChainSelector: chain_selectors.SOLANA_MAINNET.Selector,
		RPCs: []RPC{
			{
				RPCName:            "test_rpc3",
				PreferredURLScheme: "http",
				HTTPURL:            "https://test3.rpc",
				WSURL:              "wss://test3.rpc",
			},
		},
	}

	testnetNetwork := Network{
		Type:          "testnet",
		ChainSelector: chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector,
		RPCs: []RPC{
			{
				RPCName:            "test_rpc2",
				PreferredURLScheme: "http",
				HTTPURL:            "https://test2.rpc",
				WSURL:              "wss://test2.rpc",
			},
		},
	}

	cfg := NewConfig([]Network{
		mainnetNetwork1,
		testnetNetwork,
		mainnetNetwork2,
	})

	tests := []struct {
		name        string
		giveFilters []NetworkFilter
		want        *Config
	}{
		{
			name:        "filter by mainnet type",
			giveFilters: []NetworkFilter{TypesFilter(NetworkTypeMainnet)},
			want:        NewConfig([]Network{mainnetNetwork1, mainnetNetwork2}),
		},
		{
			name:        "filter by all types",
			giveFilters: []NetworkFilter{TypesFilter(NetworkTypeMainnet, NetworkTypeTestnet)},
			want:        NewConfig([]Network{mainnetNetwork1, testnetNetwork, mainnetNetwork2}),
		},
		{
			name:        "filter by chain selector 1",
			giveFilters: []NetworkFilter{ChainSelectorFilter(chain_selectors.ETHEREUM_MAINNET.Selector)},
			want:        NewConfig([]Network{mainnetNetwork1}),
		},
		{
			name:        "filter by non-existent chain selector",
			giveFilters: []NetworkFilter{ChainSelectorFilter(999)},
			want:        NewConfig([]Network{}),
		},
		{
			name:        "filter by chain family",
			giveFilters: []NetworkFilter{ChainFamilyFilter(chain_selectors.FamilyEVM)},
			want:        NewConfig([]Network{mainnetNetwork1, testnetNetwork}),
		},
		{
			name: "combination: filter by mainnet and chain selector 1",
			giveFilters: []NetworkFilter{
				TypesFilter(NetworkTypeMainnet),
				ChainSelectorFilter(chain_selectors.ETHEREUM_MAINNET.Selector),
			},
			want: NewConfig([]Network{mainnetNetwork1}),
		},
		{
			name: "combination: filter by testnet and chain selector 1 (no match)",
			giveFilters: []NetworkFilter{
				TypesFilter(NetworkTypeTestnet),
				ChainSelectorFilter(chain_selectors.ETHEREUM_MAINNET.Selector),
			},
			want: NewConfig([]Network{}),
		},
		{
			name:        "no filters",
			giveFilters: []NetworkFilter{},
			want:        NewConfig([]Network{mainnetNetwork1, testnetNetwork, mainnetNetwork2}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := cfg.FilterWith(tt.giveFilters...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_TypeFilter(t *testing.T) {
	t.Parallel()

	mainnetNetwork := Network{Type: NetworkTypeMainnet, ChainSelector: 1}
	testnetNetwork := Network{Type: NetworkTypeTestnet, ChainSelector: 2}

	tests := []struct {
		name        string
		giveTypes   []NetworkType
		giveNetwork Network
		want        bool
	}{
		{
			name:        "mainnet filter with mainnet network",
			giveTypes:   []NetworkType{NetworkTypeMainnet},
			giveNetwork: mainnetNetwork,
			want:        true,
		},
		{
			name:        "mainnet filter with testnet network",
			giveTypes:   []NetworkType{NetworkTypeMainnet},
			giveNetwork: testnetNetwork,
			want:        false,
		},
		{
			name:        "testnet filter with testnet network",
			giveTypes:   []NetworkType{NetworkTypeTestnet},
			giveNetwork: testnetNetwork,
			want:        true,
		},
		{
			name:        "testnet filter with mainnet network",
			giveTypes:   []NetworkType{NetworkTypeTestnet},
			giveNetwork: mainnetNetwork,
			want:        false,
		},
		{
			name:        "multiple types with mainnet network",
			giveTypes:   []NetworkType{NetworkTypeMainnet, NetworkTypeTestnet},
			giveNetwork: mainnetNetwork,
			want:        true,
		},
		{
			name:        "multiple types with testnet network",
			giveTypes:   []NetworkType{NetworkTypeMainnet, NetworkTypeTestnet},
			giveNetwork: testnetNetwork,
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filter := TypesFilter(tt.giveTypes...)
			got := filter(tt.giveNetwork)

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_ChainSelectorFilter(t *testing.T) {
	t.Parallel()

	network1 := Network{Type: NetworkTypeMainnet, ChainSelector: 1}
	network2 := Network{Type: NetworkTypeTestnet, ChainSelector: 2}

	tests := []struct {
		name         string
		giveSelector uint64
		giveNetwork  Network
		want         bool
	}{
		{
			name:         "matching chain selector 1",
			giveSelector: 1,
			giveNetwork:  network1,
			want:         true,
		},
		{
			name:         "non-matching chain selector 1",
			giveSelector: 1,
			giveNetwork:  network2,
			want:         false,
		},
		{
			name:         "matching chain selector 2",
			giveSelector: 2,
			giveNetwork:  network2,
			want:         true,
		},
		{
			name:         "non-matching chain selector 2",
			giveSelector: 2,
			giveNetwork:  network1,
			want:         false,
		},
		{
			name:         "non-existent chain selector",
			giveSelector: 999,
			giveNetwork:  network1,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filter := ChainSelectorFilter(tt.giveSelector)
			got := filter(tt.giveNetwork)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_ChainFamilyFilter(t *testing.T) {
	t.Parallel()

	network := Network{
		Type:          NetworkTypeMainnet,
		ChainSelector: chain_selectors.TEST_1000.Selector, // EVM
	}

	tests := []struct {
		name        string
		giveFamily  string
		giveNetwork Network
		want        bool
	}{
		{
			name:        "matching EVM family",
			giveFamily:  chain_selectors.FamilyEVM,
			giveNetwork: network,
			want:        true,
		},
		{
			name:        "does not match EVM family",
			giveFamily:  chain_selectors.FamilySolana,
			giveNetwork: network,
			want:        false,
		},
		{
			name:       "chain selector does not have family",
			giveFamily: chain_selectors.FamilyEVM,
			giveNetwork: Network{
				ChainSelector: 999999999999999999, // Non-existent chain selector
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filter := ChainFamilyFilter(tt.giveFamily)
			got := filter(tt.giveNetwork)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_Config_Load(t *testing.T) {
	t.Parallel()

	// Create temporary test files
	tmpDir := t.TempDir()

	// Valid YAML content for first file
	yamlContent1 := `
networks:
- type: "mainnet"
  chain_selector: 1
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
    ws_url: "wss://test2.rpc"
- type: "testnet"
  chain_selector: 2
  rpcs:
  - rpc_name: "duplicate_test_rpc"
    preferred_url_scheme: "http"
    http_url: "https://dup-test.rpc"
    ws_url: "wss://dup-test.rpc"
- type: "testnet"
  chain_selector: 2
  rpcs:
  - rpc_name: "test_rpc"
    preferred_url_scheme: "http"
    http_url: "https://test.rpc"
    ws_url: "wss://test.rpc"
- type: "mainnet"
  chain_selector: 3
  rpcs:
  - rpc_name: "test_rpc3"
    preferred_url_scheme: "http"
    http_url: "https://dup-test3.rpc"
    ws_url: "wss://dup-test3.rpc"`

	// Valid YAML content for second file
	yamlContent2 := `
networks:
- type: "mainnet"
  chain_selector: 3
  metadata:
    test_config:
      test_field: "value"
      test_another_field: 123
  rpcs:
  - rpc_name: "test_rpc3"
    preferred_url_scheme: "http"
    http_url: "https://test3.rpc"
    ws_url: "wss://test3.rpc"`

	tmpFile1 := filepath.Join(tmpDir, "test1.yaml")
	tmpFile2 := filepath.Join(tmpDir, "test2.yaml")
	invalidFile := filepath.Join(tmpDir, "invalid.yaml")

	network1 := Network{
		Type:          "mainnet",
		ChainSelector: 1,
		BlockExplorer: BlockExplorer{
			Type:   "Etherscan",
			APIKey: "test_key",
			URL:    "https://etherscan.io",
		},
		RPCs: []RPC{
			{
				RPCName:            "test_rpc",
				PreferredURLScheme: "http",
				HTTPURL:            "https://test.rpc",
				WSURL:              "wss://test.rpc",
			},
			{
				RPCName:            "test_rpc2",
				PreferredURLScheme: "http",
				HTTPURL:            "https://test2.rpc",
				WSURL:              "wss://test2.rpc",
			},
		},
	}

	network2 := Network{
		Type:          "testnet",
		ChainSelector: 2,
		RPCs: []RPC{
			{
				RPCName:            "test_rpc",
				PreferredURLScheme: "http",
				HTTPURL:            "https://test.rpc",
				WSURL:              "wss://test.rpc",
			},
		},
	}

	network3 := Network{
		Type:          "mainnet",
		ChainSelector: 3,
		Metadata: map[string]any{
			"test_config": map[string]any{
				"test_field":         "value",
				"test_another_field": 123,
			},
		},
		RPCs: []RPC{
			{
				RPCName:            "test_rpc3",
				PreferredURLScheme: "http",
				HTTPURL:            "https://test3.rpc",
				WSURL:              "wss://test3.rpc",
			},
		},
	}

	network4 := Network{
		Type:          "mainnet",
		ChainSelector: 3,
		RPCs: []RPC{
			{
				RPCName:            "test_rpc3",
				PreferredURLScheme: "http",
				HTTPURL:            "https://dup-test3.rpc",
				WSURL:              "wss://dup-test3.rpc",
			},
		},
	}

	err := os.WriteFile(tmpFile1, []byte(yamlContent1), 0600)
	require.NoError(t, err, "Failed to create test file")

	err = os.WriteFile(tmpFile2, []byte(yamlContent2), 0600)
	require.NoError(t, err, "Failed to create test file")

	err = os.WriteFile(invalidFile, []byte("invalid: yaml: content: ["), 0600)
	require.NoError(t, err, "Failed to create invalid test file")

	tests := []struct {
		name          string
		giveFilePaths []string
		want          *Config
		wantErr       bool
	}{
		{
			name:          "single valid file",
			giveFilePaths: []string{tmpFile1},
			want: NewConfig([]Network{
				network1,
				network2,
				network4,
			}),
		},
		{
			name:          "multiple valid files",
			giveFilePaths: []string{tmpFile1, tmpFile2},
			want: NewConfig([]Network{
				network1,
				network2,
				network3,
			}),
		},
		{
			name:          "non-existent file",
			giveFilePaths: []string{"/non/existent/file.yaml"},
			wantErr:       true,
		},
		{
			name:          "invalid yaml",
			giveFilePaths: []string{invalidFile},
			wantErr:       true,
		},
		{
			name:          "empty file paths",
			giveFilePaths: []string{},
			want:          NewConfig([]Network{}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Load(tt.giveFilePaths)
			if tt.wantErr {
				require.Error(t, err, "Expected %s error", tt.name)
			} else {
				require.NoError(t, err, "Load() should not return an error, got %v", err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_Load_WithURLTransformer(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	yamlContent := `
networks:
- type: "mainnet"
  chain_selector: 1
  rpcs:
  - rpc_name: "test_rpc"
    preferred_url_scheme: "http"
    http_url: "https://test.rpc"
    ws_url: "wss://test.rpc"`

	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(yamlContent), 0600)
	require.NoError(t, err, "Failed to create test file")

	// Test URL transformer
	transformer := func(cfg *Config) {
		for _, network := range cfg.Networks() {
			for j, rpc := range network.RPCs {
				rpc.HTTPURL = strings.Replace(rpc.HTTPURL, "test", "test2", 1)
				rpc.WSURL = strings.Replace(rpc.WSURL, "test", "test2", 1)
				network.RPCs[j] = rpc
			}
		}
	}

	got, err := Load([]string{tmpFile}, WithURLTransform(transformer))
	require.NoError(t, err, "Load() should not return an error")

	want := NewConfig([]Network{
		{
			Type:          "mainnet",
			ChainSelector: 1,
			RPCs: []RPC{
				{
					RPCName:            "test_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "https://test2.rpc",
					WSURL:              "wss://test2.rpc",
				},
			},
		},
	})

	assert.Equal(t, want, got)
}

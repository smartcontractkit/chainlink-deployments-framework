package environment

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	config_network "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

// These tests have been copied from the networks package but all network loading tests should be refactored once
// we remove the legacy network loading logic.

var (
	dummyDomain = domain.NewDomain(domain.DomainsRoot, "dummy")
)

func Test_LoadNetworks(t *testing.T) {
	t.Parallel()

	var (
		networks = []config_network.Network{
			{
				Type:          config_network.NetworkTypeMainnet,
				ChainSelector: 1,
				RPCs: []config_network.RPC{
					{
						RPCName:            "test_rpc",
						PreferredURLScheme: "http",
						HTTPURL:            "https://test.rpc",
						WSURL:              "wss://test.rpc",
					},
				},
			},
			{
				Type:          config_network.NetworkTypeTestnet,
				ChainSelector: 2,
				RPCs: []config_network.RPC{
					{
						RPCName:            "test_rpc",
						PreferredURLScheme: "http",
						HTTPURL:            "https://test.rpc",
						WSURL:              "wss://test.rpc",
					},
				},
			},
		}

		cfg        = config_network.NewConfig(networks)
		mainnetCfg = config_network.NewConfig([]config_network.Network{networks[0]})
		testnetCfg = config_network.NewConfig([]config_network.Network{networks[1]})
	)

	fixture, err := yaml.Marshal(cfg)
	require.NoError(t, err)

	dom, _ := setupConfigDirs(t)

	// Create the network config file
	err = os.WriteFile(dom.ConfigNetworksFilePath("networks.yaml"), fixture, 0600)
	require.NoError(t, err)

	// Create domain config for main test domain
	mainDomainConfig := `environments:
  local:
    network_types:
      - testnet
  staging:
    network_types:
      - testnet
  prod:
    network_types:
      - testnet
      - mainnet
  testnet:
    network_types:
      - testnet
  mainnet:
    network_types:
      - mainnet`
	err = os.WriteFile(dom.ConfigDomainFilePath(), []byte(mainDomainConfig), 0600)
	require.NoError(t, err)

	// Temporary domain for testing the Data Streams domain exception
	var (
		streamsDomainDir = filepath.Join(dom.RootPath(), "data-streams")
		streamsConfigDir = filepath.Join(streamsDomainDir, ".config", "networks")
	)

	err = os.MkdirAll(streamsConfigDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(
		filepath.Join(streamsConfigDir, "networks.yaml"), fixture, 0600,
	)
	require.NoError(t, err)

	tests := []struct {
		name       string
		giveEnv    string
		giveDomain domain.Domain
		want       *config_network.Config
		wantErr    string
	}{
		{
			name:       "Local",
			giveEnv:    Testnet,
			giveDomain: dom,
			want:       testnetCfg,
		},
		{
			name:       "Prod",
			giveEnv:    Prod,
			giveDomain: dom,
			want:       cfg,
		},
		{
			name:       "Testnet",
			giveEnv:    Testnet,
			giveDomain: dom,
			want:       testnetCfg,
		},
		{
			name:       "Staging",
			giveEnv:    Staging,
			giveDomain: dom,
			want:       testnetCfg,
		},
		{
			name:       "Mainnet",
			giveEnv:    Mainnet,
			giveDomain: dom,
			want:       mainnetCfg,
		},
		{
			name:       "failed to load network config",
			giveEnv:    StagingTestnet,
			giveDomain: domain.NewDomain("nonexistent", "dummy"),
			wantErr:    "failed to load network config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := LoadNetworks(tt.giveEnv, tt.giveDomain, logger.Test(t))
			if tt.wantErr != "" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_loadNetworkConfig(t *testing.T) {
	t.Parallel()
	yamlFile1 := `
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
        http_url: "https://dummy.rpcs.cll/ethereum-mainnet"
        ws_url: "wss://dummy.rpcs.cll/ethereum-mainnet"
      - rpc_name: "test_rpc2"
        preferred_url_scheme: "http"
        http_url: "https://test2.rpc"
        ws_url: "wss://test2.rpc"
    `

	yamlFile2 := `
  networks:
    - type: "testnet"
      chain_selector: 2
      rpcs:
      - rpc_name: "test_rpc"
        preferred_url_scheme: "http"
        http_url: "https://test.rpc"
        ws_url: "wss://test.rpc"
  `

	// Yaml with error
	yamlFile3 := `
    - type: "testnet"
      chain_selector: "2
  `

	network1 := config_network.Network{
		Type:          "mainnet",
		ChainSelector: 1,
		BlockExplorer: config_network.BlockExplorer{
			Type:   "Etherscan",
			APIKey: "test_key",
			URL:    "https://etherscan.io",
		},
		RPCs: []config_network.RPC{
			{
				RPCName:            "test_rpc",
				PreferredURLScheme: "http",
				HTTPURL:            "https://dummy.rpcs.cll/ethereum-mainnet",
				WSURL:              "wss://dummy.rpcs.cll/ethereum-mainnet",
			},
			{
				RPCName:            "test_rpc2",
				PreferredURLScheme: "http",
				HTTPURL:            "https://test2.rpc",
				WSURL:              "wss://test2.rpc",
			},
		},
	}

	network2 := config_network.Network{
		Type:          "testnet",
		ChainSelector: 2,
		RPCs: []config_network.RPC{
			{
				RPCName:            "test_rpc",
				PreferredURLScheme: "http",
				HTTPURL:            "https://test.rpc",
				WSURL:              "wss://test.rpc",
			},
		},
	}

	tests := []struct {
		name         string
		domain       domain.Domain
		filetoCreate []string
		want         *config_network.Config
		wantErr      bool
	}{
		{
			name:         "Loading single file",
			domain:       dummyDomain,
			filetoCreate: []string{yamlFile1},
			wantErr:      false,
			want:         config_network.NewConfig([]config_network.Network{network1}),
		},
		{
			name:         "Loading multiple files",
			domain:       dummyDomain,
			filetoCreate: []string{yamlFile1, yamlFile2},
			wantErr:      false,
			want:         config_network.NewConfig([]config_network.Network{network1, network2}),
		},
		{
			name:         "No config files found",
			domain:       dummyDomain,
			filetoCreate: []string{},
			wantErr:      true,
			want:         nil,
		},
		{
			name:         "Error loading YAML file",
			domain:       dummyDomain,
			filetoCreate: []string{yamlFile3},
			wantErr:      true,
			want:         nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Prepare a temp project root with both testnet and mainnet RPC files in YAML format
			var (
				rootDir   = t.TempDir()
				domainDir = filepath.Join(rootDir, "dummy")
				configDir = filepath.Join(domainDir, ".config", "networks")
			)

			// Create the domain and config directory
			err := os.MkdirAll(configDir, 0755)
			require.NoError(t, err)

			// Create the YAML files specific to the test case
			for i, file := range tt.filetoCreate {
				// Alternate between .yaml and .yml extensions for testing
				fileExt := "yaml"
				if i%2 == 0 {
					fileExt = "yml"
				}
				err = os.WriteFile(
					filepath.Join(configDir, fmt.Sprintf("file-%d.%s", i, fileExt)),
					[]byte(file), 0600,
				)
				require.NoError(t, err)
			}

			domain := domain.NewDomain(rootDir, "dummy")

			got, err := loadNetworkConfig(domain)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, got)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_loadDomainConfigNetworkTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		giveEnv      string
		domainConfig string
		want         []config_network.NetworkType
		wantErr      string
	}{
		{
			name:    "Valid domain config with testnet",
			giveEnv: "development",
			domainConfig: `environments:
  development:
    network_types:
      - testnet`,
			want: []config_network.NetworkType{config_network.NetworkTypeTestnet},
		},
		{
			name:    "Valid domain config with mainnet",
			giveEnv: "production",
			domainConfig: `environments:
  production:
    network_types:
      - mainnet`,
			want: []config_network.NetworkType{config_network.NetworkTypeMainnet},
		},
		{
			name:    "Valid domain config with both types",
			giveEnv: "staging",
			domainConfig: `environments:
  staging:
    network_types:
      - testnet
      - mainnet`,
			want: []config_network.NetworkType{
				config_network.NetworkTypeTestnet,
				config_network.NetworkTypeMainnet,
			},
		},
		{
			name:    "Environment not found",
			giveEnv: "nonexistent",
			domainConfig: `environments:
  development:
    network_types:
      - testnet`,
			wantErr: "environment nonexistent not found in domain config",
		},
		{
			name:         "Invalid YAML",
			giveEnv:      "development",
			domainConfig: `invalid yaml: [}`,
			wantErr:      "failed to load domain config",
		},
		{
			name:    "Empty network_types - validation error",
			giveEnv: "development",
			domainConfig: `environments:
  development:
    network_types: []`,
			wantErr: "network_types is required and cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Prepare a temp domain with domain config
			var (
				rootDir   = t.TempDir()
				domainDir = filepath.Join(rootDir, "dummy")
				configDir = filepath.Join(domainDir, ".config")
			)

			// Create the domain and config directory
			err := os.MkdirAll(configDir, 0755)
			require.NoError(t, err)

			// Create the domain config file
			err = os.WriteFile(
				filepath.Join(configDir, "domain.yaml"), []byte(tt.domainConfig), 0600,
			)
			require.NoError(t, err)

			domain := domain.NewDomain(rootDir, "dummy")

			got, err := loadDomainConfigNetworkTypes(tt.giveEnv, domain)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

package network

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	cldf_domain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"

	cldf_config_network "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
)

var (
	dummyDomain = cldf_domain.NewDomain(cldf_domain.DomainsRoot, "dummy")
)

func Test_LoadNetworks(t *testing.T) {
	t.Parallel()

	var (
		networks = []cldf_config_network.Network{
			{
				Type:          cldf_config_network.NetworkTypeMainnet,
				ChainSelector: 1,
				RPCs: []cldf_config_network.RPC{
					{
						RPCName:            "test_rpc",
						PreferredURLScheme: "http",
						HTTPURL:            "https://test.rpc",
						WSURL:              "wss://test.rpc",
					},
				},
			},
			{
				Type:          cldf_config_network.NetworkTypeTestnet,
				ChainSelector: 2,
				RPCs: []cldf_config_network.RPC{
					{
						RPCName:            "test_rpc",
						PreferredURLScheme: "http",
						HTTPURL:            "https://test.rpc",
						WSURL:              "wss://test.rpc",
					},
				},
			},
		}

		cfg        = cldf_config_network.NewConfig(networks)
		mainnetCfg = cldf_config_network.NewConfig([]cldf_config_network.Network{networks[0]})
		testnetCfg = cldf_config_network.NewConfig([]cldf_config_network.Network{networks[1]})
	)

	fixture, err := yaml.Marshal(cfg)
	require.NoError(t, err)

	// Prepare a temp domain with network config files
	var (
		rootDir   = t.TempDir()
		domainDir = filepath.Join(rootDir, "dummy")
		configDir = filepath.Join(domainDir, ".config", "networks")
	)

	// Create the domain and config directory
	err = os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	// Create the network config file
	err = os.WriteFile(
		filepath.Join(configDir, "networks.yaml"), fixture, 0600,
	)
	require.NoError(t, err)

	// Temporary domain for testing the Data Streams domain exception
	var (
		streamsDomainDir = filepath.Join(rootDir, "data-streams")
		streamsConfigDir = filepath.Join(streamsDomainDir, ".config", "networks")
	)

	err = os.MkdirAll(streamsConfigDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(
		filepath.Join(streamsConfigDir, "networks.yaml"), fixture, 0600,
	)
	require.NoError(t, err)

	domain := cldf_domain.NewDomain(rootDir, "dummy")

	tests := []struct {
		name       string
		giveEnv    string
		giveDomain cldf_domain.Domain
		want       *cldf_config_network.Config
		wantErr    string
	}{
		{
			name:       "Local",
			giveEnv:    environment.Testnet,
			giveDomain: domain,
			want:       testnetCfg,
		},
		{
			name:       "Staging Testnet",
			giveEnv:    environment.StagingTestnet,
			giveDomain: domain,
			want:       testnetCfg,
		},
		{
			name:       "Prod Testnet",
			giveEnv:    environment.ProdTestnet,
			giveDomain: domain,
			want:       testnetCfg,
		},
		{
			name:       "Staging Mainnet",
			giveEnv:    environment.StagingMainnet,
			giveDomain: domain,
			want:       mainnetCfg,
		},
		{
			name:       "Prod Mainnet",
			giveEnv:    environment.ProdMainnet,
			giveDomain: domain,
			want:       mainnetCfg,
		},
		{
			name:       "Prod",
			giveEnv:    environment.Prod,
			giveDomain: domain,
			want:       cfg,
		},
		{
			name:       "Testnet",
			giveEnv:    environment.Testnet,
			giveDomain: domain,
			want:       testnetCfg,
		},
		{
			name:       "Sol Staging",
			giveEnv:    environment.SolStaging,
			giveDomain: domain,
			want:       testnetCfg,
		},
		{
			name:       "Staging",
			giveEnv:    environment.Staging,
			giveDomain: domain,
			want:       testnetCfg,
		},
		{
			name:       "Staging",
			giveEnv:    environment.Staging,
			giveDomain: cldf_domain.NewDomain(rootDir, "data-streams"),
			want:       cfg,
		},
		{
			name:       "Mainnet",
			giveEnv:    environment.Mainnet,
			giveDomain: domain,
			want:       mainnetCfg,
		},
		{
			name:       "Unknown Environment",
			giveEnv:    "unknown",
			giveDomain: domain,
			wantErr:    "unknown env: unknown",
		},
		{
			name:       "failed to load network config",
			giveEnv:    environment.StagingTestnet,
			giveDomain: cldf_domain.NewDomain("nonexistent", "dummy"),
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

	network1 := cldf_config_network.Network{
		Type:          "mainnet",
		ChainSelector: 1,
		BlockExplorer: cldf_config_network.BlockExplorer{
			Type:   "Etherscan",
			APIKey: "test_key",
			URL:    "https://etherscan.io",
		},
		RPCs: []cldf_config_network.RPC{
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

	network2 := cldf_config_network.Network{
		Type:          "testnet",
		ChainSelector: 2,
		RPCs: []cldf_config_network.RPC{
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
		domain       cldf_domain.Domain
		filetoCreate []string
		want         *cldf_config_network.Config
		wantErr      bool
	}{
		{
			name:         "Loading single file",
			domain:       dummyDomain,
			filetoCreate: []string{yamlFile1},
			wantErr:      false,
			want:         cldf_config_network.NewConfig([]cldf_config_network.Network{network1}),
		},
		{
			name:         "Loading multiple files",
			domain:       dummyDomain,
			filetoCreate: []string{yamlFile1, yamlFile2},
			wantErr:      false,
			want:         cldf_config_network.NewConfig([]cldf_config_network.Network{network1, network2}),
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

			domain := cldf_domain.NewDomain(rootDir, "dummy")

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

func Test_LoadNetworks_WithDomainConfig(t *testing.T) {
	t.Parallel()

	var (
		networks = []cldf_config_network.Network{
			{
				Type:          cldf_config_network.NetworkTypeMainnet,
				ChainSelector: 1,
				RPCs: []cldf_config_network.RPC{
					{
						RPCName:            "test_rpc",
						PreferredURLScheme: "http",
						HTTPURL:            "https://test.rpc",
						WSURL:              "wss://test.rpc",
					},
				},
			},
			{
				Type:          cldf_config_network.NetworkTypeTestnet,
				ChainSelector: 2,
				RPCs: []cldf_config_network.RPC{
					{
						RPCName:            "test_rpc",
						PreferredURLScheme: "http",
						HTTPURL:            "https://test.rpc",
						WSURL:              "wss://test.rpc",
					},
				},
			},
		}

		cfg        = cldf_config_network.NewConfig(networks)
		mainnetCfg = cldf_config_network.NewConfig([]cldf_config_network.Network{networks[0]})
		testnetCfg = cldf_config_network.NewConfig([]cldf_config_network.Network{networks[1]})
	)

	fixture, err := yaml.Marshal(cfg)
	require.NoError(t, err)

	tests := []struct {
		name           string
		giveEnv        string
		domainConfig   string
		want           *cldf_config_network.Config
		wantErr        string
		expectFallback bool
	}{
		{
			name:    "Domain config with testnet only",
			giveEnv: "development",
			domainConfig: `environments:
  development:
    network_types:
      - testnet`,
			want: testnetCfg,
		},
		{
			name:    "Domain config with mainnet only",
			giveEnv: "production",
			domainConfig: `environments:
  production:
    network_types:
      - mainnet`,
			want: mainnetCfg,
		},
		{
			name:    "Domain config with both testnet and mainnet",
			giveEnv: "staging",
			domainConfig: `environments:
  staging:
    network_types:
      - testnet
      - mainnet`,
			want: cfg,
		},
		{
			name:    "Environment not found in domain config - falls back to legacy",
			giveEnv: "nonexistent",
			domainConfig: `environments:
  development:
    network_types:
      - testnet`,
			wantErr: "unknown env: nonexistent",
		},
		{
			name:           "Invalid domain config format - falls back to legacy",
			giveEnv:        environment.StagingTestnet,
			domainConfig:   `invalid yaml content: [}`,
			want:           testnetCfg,
			expectFallback: true,
		},
		{
			name:    "Empty network_types in domain config - falls back to legacy",
			giveEnv: environment.StagingTestnet,
			domainConfig: `environments:
  staging_testnet:
    network_types: []`,
			want:           testnetCfg,
			expectFallback: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Prepare a temp domain with network config files
			var (
				rootDir   = t.TempDir()
				domainDir = filepath.Join(rootDir, "dummy")
				configDir = filepath.Join(domainDir, ".config")
			)

			// Create the domain and config directory
			err = os.MkdirAll(filepath.Join(configDir, "networks"), 0755)
			require.NoError(t, err)

			// Create the network config file
			err = os.WriteFile(
				filepath.Join(configDir, "networks", "networks.yaml"), fixture, 0600,
			)
			require.NoError(t, err)

			// Create the domain config file
			err = os.WriteFile(
				filepath.Join(configDir, "domain.yaml"), []byte(tt.domainConfig), 0600,
			)
			require.NoError(t, err)

			domain := cldf_domain.NewDomain(rootDir, "dummy")

			got, err := LoadNetworks(tt.giveEnv, domain, logger.Test(t))
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

func Test_loadDomainConfigNetworkTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		giveEnv      string
		domainConfig string
		want         []cldf_config_network.NetworkType
		wantErr      string
	}{
		{
			name:    "Valid domain config with testnet",
			giveEnv: "development",
			domainConfig: `environments:
  development:
    network_types:
      - testnet`,
			want: []cldf_config_network.NetworkType{cldf_config_network.NetworkTypeTestnet},
		},
		{
			name:    "Valid domain config with mainnet",
			giveEnv: "production",
			domainConfig: `environments:
  production:
    network_types:
      - mainnet`,
			want: []cldf_config_network.NetworkType{cldf_config_network.NetworkTypeMainnet},
		},
		{
			name:    "Valid domain config with both types",
			giveEnv: "staging",
			domainConfig: `environments:
  staging:
    network_types:
      - testnet
      - mainnet`,
			want: []cldf_config_network.NetworkType{
				cldf_config_network.NetworkTypeTestnet,
				cldf_config_network.NetworkTypeMainnet,
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

			domain := cldf_domain.NewDomain(rootDir, "dummy")

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

func Test_getLegacyNetworkTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		giveEnv string
		domain  cldf_domain.Domain
		want    []cldf_config_network.NetworkType
		wantErr string
	}{
		{
			name:    "Local environment",
			giveEnv: environment.Local,
			domain:  dummyDomain,
			want:    []cldf_config_network.NetworkType{cldf_config_network.NetworkTypeTestnet},
		},
		{
			name:    "StagingTestnet environment",
			giveEnv: environment.StagingTestnet,
			domain:  dummyDomain,
			want:    []cldf_config_network.NetworkType{cldf_config_network.NetworkTypeTestnet},
		},
		{
			name:    "ProdTestnet environment",
			giveEnv: environment.ProdTestnet,
			domain:  dummyDomain,
			want:    []cldf_config_network.NetworkType{cldf_config_network.NetworkTypeTestnet},
		},
		{
			name:    "StagingMainnet environment",
			giveEnv: environment.StagingMainnet,
			domain:  dummyDomain,
			want:    []cldf_config_network.NetworkType{cldf_config_network.NetworkTypeMainnet},
		},
		{
			name:    "ProdMainnet environment",
			giveEnv: environment.ProdMainnet,
			domain:  dummyDomain,
			want:    []cldf_config_network.NetworkType{cldf_config_network.NetworkTypeMainnet},
		},
		{
			name:    "Prod environment",
			giveEnv: environment.Prod,
			domain:  dummyDomain,
			want: []cldf_config_network.NetworkType{
				cldf_config_network.NetworkTypeTestnet,
				cldf_config_network.NetworkTypeMainnet,
			},
		},
		{
			name:    "Testnet environment",
			giveEnv: environment.Testnet,
			domain:  dummyDomain,
			want:    []cldf_config_network.NetworkType{cldf_config_network.NetworkTypeTestnet},
		},
		{
			name:    "SolStaging environment",
			giveEnv: environment.SolStaging,
			domain:  dummyDomain,
			want:    []cldf_config_network.NetworkType{cldf_config_network.NetworkTypeTestnet},
		},
		{
			name:    "Staging environment (non-data-streams domain)",
			giveEnv: environment.Staging,
			domain:  dummyDomain,
			want:    []cldf_config_network.NetworkType{cldf_config_network.NetworkTypeTestnet},
		},
		{
			name:    "Staging environment (data-streams domain)",
			giveEnv: environment.Staging,
			domain:  cldf_domain.NewDomain(cldf_domain.DomainsRoot, "data-streams"),
			want: []cldf_config_network.NetworkType{
				cldf_config_network.NetworkTypeTestnet,
				cldf_config_network.NetworkTypeMainnet,
			},
		},
		{
			name:    "Mainnet environment",
			giveEnv: environment.Mainnet,
			domain:  dummyDomain,
			want:    []cldf_config_network.NetworkType{cldf_config_network.NetworkTypeMainnet},
		},
		{
			name:    "Unknown environment",
			giveEnv: "unknown",
			domain:  dummyDomain,
			wantErr: "unknown env: unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := getLegacyNetworkTypes(tt.giveEnv, tt.domain, logger.Test(t))
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

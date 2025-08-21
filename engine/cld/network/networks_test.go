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
						HTTPURL:            "https://test-mainnet.rpc",
						WSURL:              "wss://test-mainnet.rpc",
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
						HTTPURL:            "https://test-testnet.rpc",
						WSURL:              "wss://test-testnet.rpc",
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

	err = os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(configDir, "networks.yaml"), fixture, 0600)
	require.NoError(t, err)

	// Temp domain for data-streams exception
	var (
		streamsDomainDir = filepath.Join(rootDir, "data-streams")
		streamsConfigDir = filepath.Join(streamsDomainDir, ".config", "networks")
	)
	err = os.MkdirAll(streamsConfigDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(streamsConfigDir, "networks.yaml"), fixture, 0600)
	require.NoError(t, err)

	domain := cldf_domain.NewDomain(rootDir, "dummy")

	tests := []struct {
		name       string
		giveEnv    string
		giveDomain cldf_domain.Domain
		want       *cldf_config_network.Config
		wantErr    string
	}{
		{name: "Local", giveEnv: environment.Testnet, giveDomain: domain, want: testnetCfg},
		{name: "Staging Testnet", giveEnv: environment.StagingTestnet, giveDomain: domain, want: testnetCfg},
		{name: "Prod Testnet", giveEnv: environment.ProdTestnet, giveDomain: domain, want: testnetCfg},
		{name: "Staging Mainnet", giveEnv: environment.StagingMainnet, giveDomain: domain, want: mainnetCfg},
		{name: "Prod Mainnet", giveEnv: environment.ProdMainnet, giveDomain: domain, want: mainnetCfg},
		{name: "Prod", giveEnv: environment.Prod, giveDomain: domain, want: cfg},
		{name: "Testnet", giveEnv: environment.Testnet, giveDomain: domain, want: testnetCfg},
		{name: "Sol Staging", giveEnv: environment.SolStaging, giveDomain: domain, want: testnetCfg},
		{name: "Staging", giveEnv: environment.Staging, giveDomain: domain, want: testnetCfg},
		{name: "Staging (data-streams exception)", giveEnv: environment.Staging, giveDomain: cldf_domain.NewDomain(rootDir, "data-streams"), want: cfg},
		{name: "Mainnet", giveEnv: environment.Mainnet, giveDomain: domain, want: mainnetCfg},
		{name: "Unknown Environment", giveEnv: "unknown", giveDomain: domain, wantErr: "unknown env: unknown"},
		{name: "failed to load network config", giveEnv: environment.StagingTestnet, giveDomain: cldf_domain.NewDomain("nonexistent", "dummy"), wantErr: "failed to load network config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := LoadNetworks(tt.giveEnv, tt.giveDomain, logger.Test(t), nil)
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
        type: "TestExplorer"
        api_key: "test_key"
        url: "https://test-explorer.local"
      rpcs:
      - rpc_name: "test_rpc"
        preferred_url_scheme: "http"
        http_url: "https://test-rpcs.local/ethereum-mainnet"
        ws_url: "wss://test-rpcs.local/ethereum-mainnet"
      - rpc_name: "test_rpc2"
        preferred_url_scheme: "http"
        http_url: "https://test2.local"
        ws_url: "wss://test2.local"
    `

	yamlFile2 := `
  networks:
    - type: "testnet"
      chain_selector: 2
      rpcs:
      - rpc_name: "test_rpc"
        preferred_url_scheme: "http"
        http_url: "https://testnet.local"
        ws_url: "wss://testnet.local"
  `

	// malformed YAML
	yamlFile3 := `
    - type: "testnet"
      chain_selector: "2
  `

	network1 := cldf_config_network.Network{
		Type:          "mainnet",
		ChainSelector: 1,
		BlockExplorer: cldf_config_network.BlockExplorer{
			Type:   "TestExplorer",
			APIKey: "test_key",
			URL:    "https://test-explorer.local",
		},
		RPCs: []cldf_config_network.RPC{
			{
				RPCName:            "test_rpc",
				PreferredURLScheme: "http",
				HTTPURL:            "https://gap-proxy.local:4443/ethereum-mainnet",
				WSURL:              "wss://gap-proxy.local:9443/ethereum-mainnet",
			},
			{
				RPCName:            "test_rpc2",
				PreferredURLScheme: "http",
				HTTPURL:            "https://test2.local",
				WSURL:              "wss://test2.local",
			},
		},
	}
	network2 := cldf_config_network.Network{
		Type:          "testnet",
		ChainSelector: 2,
		RPCs: []cldf_config_network.RPC{
			{RPCName: "test_rpc", PreferredURLScheme: "http", HTTPURL: "https://testnet.local", WSURL: "wss://testnet.local"},
		},
	}

	injected := &Config{
		RPCsHost:     "test-rpcs.local",
		GapProxyHost: "gap-proxy.local",
		GapWSPort:    "9443",
		GapHTTPPort:  "4443",
		UseGAP:       true,
	}

	tests := []struct {
		name         string
		filetoCreate []string
		want         *cldf_config_network.Config
		wantErr      bool
	}{
		{name: "Loading single file", filetoCreate: []string{yamlFile1}, want: cldf_config_network.NewConfig([]cldf_config_network.Network{network1})},
		{name: "Loading multiple files", filetoCreate: []string{yamlFile1, yamlFile2}, want: cldf_config_network.NewConfig([]cldf_config_network.Network{network1, network2})},
		{name: "No config files found", filetoCreate: []string{}, wantErr: true},
		{name: "Error loading YAML file", filetoCreate: []string{yamlFile3}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rootDir := t.TempDir()
			configDir := filepath.Join(rootDir, "dummy", ".config", "networks")
			require.NoError(t, os.MkdirAll(configDir, 0755))

			for i, file := range tt.filetoCreate {
				ext := "yaml"
				if i%2 == 0 {
					ext = "yml"
				}
				require.NoError(t, os.WriteFile(filepath.Join(configDir, fmt.Sprintf("file-%d.%s", i, ext)), []byte(file), 0600))
			}

			domain := cldf_domain.NewDomain(rootDir, "dummy")
			got, err := loadNetworkConfig(domain, injected)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_gapURLTransform(t *testing.T) {
	t.Parallel()

	const (
		originalHost = "test-rpcs.local"
		proxyHost    = "gap-proxy.local"
	)

	tests := []struct {
		name string
		give string
		port string
		want string
	}{
		{
			name: "replace http uri host",
			give: "https://test-rpcs.local/chain",
			port: "4443",
			want: "https://gap-proxy.local:4443/chain",
		},
		{
			name: "replace ws uri host",
			give: "wss://test-rpcs.local/chain",
			port: "9443",
			want: "wss://gap-proxy.local:9443/chain",
		},
		{
			name: "leave unrelated url unchanged",
			give: "https://unrelated.local",
			port: "4443",
			want: "https://unrelated.local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			transformFunc := gapURLTransformer(originalHost, proxyHost, tt.port)
			assert.Equal(t, tt.want, transformFunc(tt.give))
		})
	}
}

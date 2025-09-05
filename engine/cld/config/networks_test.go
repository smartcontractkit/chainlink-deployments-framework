package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	cfgdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/domain"
	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

func Test_LoadNetworks(t *testing.T) {
	t.Parallel()

	// Default setup function for creating a domain with testnet and mainnet networks and a domain config file.
	defaultSetupFunc := func(t *testing.T) fdomain.Domain {
		t.Helper()

		dom, _ := setupConfigDirs(t)
		writeConfigNetworksFile(t, dom, "networks-testnet.yaml", "networks-testnet.yaml")
		writeConfigNetworksFile(t, dom, "networks-mainnet.yaml", "networks-mainnet.yaml")
		writeConfigDomainFile(t, dom, "fdomain.yaml")

		return dom
	}

	tests := []struct {
		name      string
		setupFunc func(t *testing.T) fdomain.Domain
		giveEnv   string
		want      *cfgnet.Config
		wantErr   string
	}{
		{
			name:      "Only Testnet",
			setupFunc: defaultSetupFunc,
			giveEnv:   "staging_testnet",
			want:      cfgnet.NewConfig([]cfgnet.Network{testnetNetwork1, testnetNetwork2}),
		},
		{
			name:      "Only Mainnet",
			setupFunc: defaultSetupFunc,
			giveEnv:   "prod_mainnet",
			want:      cfgnet.NewConfig([]cfgnet.Network{mainnetNetwork1}),
		},
		{
			name:      "Both Testnet and Mainnet",
			setupFunc: defaultSetupFunc,
			giveEnv:   "staging",
			want:      cfgnet.NewConfig([]cfgnet.Network{testnetNetwork1, testnetNetwork2, mainnetNetwork1}),
		},
		{
			name: "failed to load network config",
			setupFunc: func(t *testing.T) fdomain.Domain {
				t.Helper()

				return fdomain.NewDomain("nonexistent", "dummy")
			},
			giveEnv: "staging_testnet",
			wantErr: "failed to load network config",
		},
		{
			name: "domain config not found",
			setupFunc: func(t *testing.T) fdomain.Domain {
				t.Helper()

				dom, _ := setupConfigDirs(t)

				return dom
			},
			giveEnv: "staging_testnet",
			wantErr: "domain config not found",
		},
		{
			name: "failed to load domain config network types",
			setupFunc: func(t *testing.T) fdomain.Domain {
				t.Helper()

				dom, _ := setupConfigDirs(t)
				writeConfigDomainFile(t, dom, "fdomain.yaml")

				return dom
			},
			giveEnv: "nonexistent",
			wantErr: "failed to load domain config network types",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var dom fdomain.Domain
			if tt.setupFunc != nil {
				dom = tt.setupFunc(t)
			}

			got, err := LoadNetworks(tt.giveEnv, dom, logger.Test(t))
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

	tests := []struct {
		name         string
		networkFiles map[string]string // Maps the network config file name to a filename in the testdata directory.
		want         *cfgnet.Config
		wantErr      string
	}{
		{
			name:         "Loading single file with yaml extension",
			networkFiles: map[string]string{"networks.yaml": "networks-testnet.yaml"},
			want:         cfgnet.NewConfig([]cfgnet.Network{testnetNetwork1, testnetNetwork2}),
		},
		{
			name:         "Loading single file with yml extension",
			networkFiles: map[string]string{"networks.yml": "networks-testnet.yaml"},
			want:         cfgnet.NewConfig([]cfgnet.Network{testnetNetwork1, testnetNetwork2}),
		},
		{
			name: "Loading multiple files",
			networkFiles: map[string]string{
				"networks-1.yaml": "networks-testnet.yaml",
				"networks-2.yml":  "networks-mainnet.yaml",
			},
			want: cfgnet.NewConfig([]cfgnet.Network{testnetNetwork1, testnetNetwork2, mainnetNetwork1}),
		},
		{
			name:    "No config files found",
			wantErr: "no config files found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dom, _ := setupConfigDirs(t)

			for filename, testdataFileName := range tt.networkFiles {
				writeConfigNetworksFile(t, dom, filename, testdataFileName)
			}

			got, err := loadNetworkConfig(dom)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
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

	var (
		testEnv = "test"
	)

	tests := []struct {
		name       string
		configEnvs []string // The environments to include in the domain config file. Use this to set the network types for the environment.
		giveEnv    string
		want       []cfgnet.NetworkType
		wantErr    string
	}{
		{
			name:       "Valid domain config with testnet",
			configEnvs: []string{"testnet"},
			giveEnv:    testEnv,
			want:       []cfgnet.NetworkType{cfgnet.NetworkTypeTestnet},
		},
		{
			name:       "Valid domain config with mainnet",
			configEnvs: []string{"mainnet"},
			giveEnv:    testEnv,
			want:       []cfgnet.NetworkType{cfgnet.NetworkTypeMainnet},
		},
		{
			name:       "Valid domain config with both types",
			configEnvs: []string{"testnet", "mainnet"},
			giveEnv:    testEnv,
			want: []cfgnet.NetworkType{
				cfgnet.NetworkTypeTestnet,
				cfgnet.NetworkTypeMainnet,
			},
		},
		{
			name:       "Environment not found",
			configEnvs: []string{"testnet"},
			giveEnv:    "nonexistent",
			wantErr:    "environment nonexistent not found in domain config",
		},
		{
			name:       "domain config file not found",
			configEnvs: nil, // No domain config file will be created for this test.
			giveEnv:    "development",
			wantErr:    "failed to load domain config",
		},
		{
			name:       "Empty network_types - validation error",
			configEnvs: []string{},
			giveEnv:    "development",
			wantErr:    "network_types is required and cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dom, _ := setupConfigDirs(t)

			if tt.configEnvs != nil {
				domainConfig := cfgdomain.DomainConfig{
					Environments: map[string]cfgdomain.Environment{
						testEnv: {
							NetworkTypes: tt.configEnvs,
						},
					},
				}

				yamlConfig, err := yaml.Marshal(domainConfig)
				require.NoError(t, err)

				// Create the domain config file
				err = os.WriteFile(
					filepath.Join(dom.ConfigDomainFilePath()), yamlConfig, filePerms,
				)
				require.NoError(t, err)
			}

			got, err := loadDomainConfigNetworkTypes(tt.giveEnv, dom)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

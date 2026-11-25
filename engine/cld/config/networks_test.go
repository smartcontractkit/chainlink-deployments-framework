package config

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

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
		writeConfigDomainFile(t, dom, "domain.yaml")

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
				writeConfigDomainFile(t, dom, "domain.yaml")

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

func Test_networkTypesForEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		giveNetworkTypes []string
		giveEnv          string
		want             []cfgnet.NetworkType
		wantErr          string
	}{
		{
			name:             "testnet",
			giveNetworkTypes: []string{"testnet"},
			giveEnv:          "test",
			want:             []cfgnet.NetworkType{cfgnet.NetworkTypeTestnet},
		},
		{
			name:             "mainnet",
			giveNetworkTypes: []string{"mainnet"},
			giveEnv:          "test",
			want:             []cfgnet.NetworkType{cfgnet.NetworkTypeMainnet},
		},
		{
			name:             "both types",
			giveNetworkTypes: []string{"testnet", "mainnet"},
			giveEnv:          "test",
			want:             []cfgnet.NetworkType{cfgnet.NetworkTypeTestnet, cfgnet.NetworkTypeMainnet},
		},
		{
			name:             "environment not found",
			giveNetworkTypes: []string{"testnet"},
			giveEnv:          "nonexistent",
			wantErr:          "environment nonexistent not found in domain config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			domainConfig := &cfgdomain.DomainConfig{
				Environments: map[string]cfgdomain.Environment{
					"test": {NetworkTypes: tt.giveNetworkTypes},
				},
			}

			got, err := networkTypesForEnv(domainConfig, tt.giveEnv)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_LoadNetworks_RPCManifest(t *testing.T) {
	t.Parallel()

	const selector = uint64(16015286601757825753)

	manifestPath := filepath.Join("network", "rpcmanifest", "testdata", "rpcs.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, manifestPath)
	}))
	t.Cleanup(srv.Close)
	manifestURL := srv.URL + "/files/rpcs.json"

	const (
		emptyRPCs = `networks:
  - type: testnet
    chain_selector: 16015286601757825753
`
		appendRPCs = `networks:
  - type: testnet
    chain_selector: 16015286601757825753
    rpcs_mode: append
    rpcs:
      - rpc_name: Mine
        http_url: https://mine.example.com
`
	)

	tests := []struct {
		name            string
		giveNetworks    string
		giveManifestURL string
		wantRPCNames    []string
		wantErr         string
	}{
		{
			name:            "empty rpcs are filled from the manifest",
			giveNetworks:    emptyRPCs,
			giveManifestURL: manifestURL,
			wantRPCNames:    []string{"LinkPool (linkpool1)", "Alchemy (alchemy1)"},
		},
		{
			name:         "empty rpcs without a manifest fail validation",
			giveNetworks: emptyRPCs,
			wantErr:      "at least one RPC is required",
		},
		{
			name:            "append mode adds manifest rpcs after yaml rpcs",
			giveNetworks:    appendRPCs,
			giveManifestURL: manifestURL,
			wantRPCNames:    []string{"Mine", "LinkPool (linkpool1)", "Alchemy (alchemy1)"},
		},
		{
			name:         "append mode without a manifest keeps yaml rpcs",
			giveNetworks: appendRPCs,
			wantRPCNames: []string{"Mine"},
		},
		{
			name: "manifest is not loaded for networks outside the environment",
			giveNetworks: `networks:
  - type: testnet
    chain_selector: 16015286601757825753
    rpcs:
      - rpc_name: primary
        http_url: https://testnet.example.com
  - type: mainnet
    chain_selector: 5009297550715157269
`,
			giveManifestURL: "ftp://rpcs.example.com/rpcs.json",
			wantRPCNames:    []string{"primary"},
		},
		{
			name:            "non-http(s) manifest url fails to load",
			giveNetworks:    emptyRPCs,
			giveManifestURL: "ftp://rpcs.example.com/rpcs.json",
			wantErr:         "failed to load rpc manifest: rpc manifest url must be an http(s) url",
		},
		{
			name: "missing type",
			giveNetworks: `networks:
  - chain_selector: 16015286601757825753
    rpcs:
      - rpc_name: primary
        http_url: https://testnet.example.com
`,
			wantErr: "type is required",
		},
		{
			name: "unrecognized type",
			giveNetworks: `networks:
  - type: mainet
    chain_selector: 16015286601757825753
    rpcs:
      - rpc_name: primary
        http_url: https://testnet.example.com
`,
			wantErr: `type "mainet" is not a recognized network type`,
		},
		{
			name: "unrecognized rpcs_mode",
			giveNetworks: `networks:
  - type: testnet
    chain_selector: 16015286601757825753
    rpcs_mode: sideways
`,
			wantErr: `rpcs_mode "sideways" is not recognized`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dom, env := setupConfigDirs(t)
			require.NoError(t, os.WriteFile(
				dom.ConfigNetworksFilePath("networks.yaml"), []byte(tt.giveNetworks), filePerms,
			))

			domainConfig := "environments:\n  " + env + ":\n    network_types:\n      - testnet\n"
			if tt.giveManifestURL != "" {
				domainConfig += fmt.Sprintf("rpc_manifest_url: %q\n", tt.giveManifestURL)
			}
			require.NoError(t, os.WriteFile(dom.ConfigDomainFilePath(), []byte(domainConfig), filePerms))

			got, err := LoadNetworks(env, dom, logger.Test(t))
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}
			require.NoError(t, err)
			require.Equal(t, []uint64{selector}, got.ChainSelectors())

			network, err := got.NetworkBySelector(selector)
			require.NoError(t, err)

			names := make([]string, 0, len(network.RPCs))
			for _, rpc := range network.RPCs {
				names = append(names, rpc.RPCName)
			}
			assert.Equal(t, tt.wantRPCNames, names)
		})
	}
}

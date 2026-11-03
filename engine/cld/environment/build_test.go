package environment

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cfgenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	cfgnotifications "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/notifications"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

const (
	// Real selectors so that the chain-selectors lookups in resolveNetworkType resolve.
	selSepolia       uint64 = 16015286601757825753 // ethereum-testnet-sepolia
	selMainnet       uint64 = 5009297550715157269  // ethereum-mainnet
	selZkSyncSepolia uint64 = 6898391096552792247  // zksync-testnet-sepolia
	selNonExisting   uint64 = 1
)

func Test_resolveNetworkType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    cfgnet.Network
		want    cfgnet.NetworkType
		wantErr string
	}{
		{
			name: "derives testnet from the chain selector",
			give: cfgnet.Network{ChainSelector: selSepolia},
			want: cfgnet.NetworkTypeTestnet,
		},
		{
			name: "derives mainnet from the chain selector",
			give: cfgnet.Network{ChainSelector: selMainnet},
			want: cfgnet.NetworkTypeMainnet,
		},
		{
			name: "honours a matching caller supplied type",
			give: cfgnet.Network{ChainSelector: selSepolia, Type: cfgnet.NetworkTypeTestnet},
			want: cfgnet.NetworkTypeTestnet,
		},
		{
			// A mismatch is a mistake in the params, so it is reported rather
			// than silently corrected to the derived value.
			name:    "rejects a conflicting caller supplied type",
			give:    cfgnet.Network{ChainSelector: selSepolia, Type: cfgnet.NetworkTypeMainnet},
			wantErr: `type "mainnet" does not match chain selector, which is "testnet"`,
		},
		{
			name:    "unknown chain selector",
			give:    cfgnet.Network{ChainSelector: selNonExisting},
			wantErr: "unknown chain selector",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolveNetworkType(t.Context(), tt.give)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_BuildNetworksConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		giveNets      []cfgnet.Network
		giveBase      string
		giveSelectors []uint64
		want          *cfgnet.Config
		// wantRPCs      map[uint64][]cfgnet.RPC
		wantErr string
	}{
		{
			name:     "derives a default RPC per chain from the base URL",
			giveNets: []cfgnet.Network{{ChainSelector: selSepolia}, {ChainSelector: selMainnet}},
			giveBase: "http://proxy.example.com",
			want: cfgnet.NewConfig([]cfgnet.Network{
				{
					Type:          cfgnet.NetworkTypeTestnet,
					ChainSelector: 16015286601757825753,
					RPCs: []cfgnet.RPC{{
						RPCName:            "default",
						PreferredURLScheme: "http",
						HTTPURL:            "http://proxy.example.com/16015286601757825753",
					}},
				},
				{
					Type:          cfgnet.NetworkTypeMainnet,
					ChainSelector: 5009297550715157269,
					RPCs: []cfgnet.RPC{{
						RPCName:            "default",
						PreferredURLScheme: "http",
						HTTPURL:            "http://proxy.example.com/5009297550715157269",
					}},
				},
			}),
		},
		{
			name:          "filters by given chain selectors",
			giveNets:      []cfgnet.Network{{ChainSelector: selSepolia}, {ChainSelector: selMainnet}},
			giveBase:      "http://proxy.example.com",
			giveSelectors: []uint64{selSepolia},
			want: cfgnet.NewConfig([]cfgnet.Network{{
				Type:          cfgnet.NetworkTypeTestnet,
				ChainSelector: 16015286601757825753,
				RPCs: []cfgnet.RPC{{
					RPCName:            "default",
					PreferredURLScheme: "http",
					HTTPURL:            "http://proxy.example.com/16015286601757825753",
				}},
			}}),
		},
		{
			// Ordering is load bearing: chains.LoadChains reads RPCs[0] for every
			// non-EVM family, so the derived URL must win over caller supplied ones.
			name: "prepends the default RPC ahead of caller supplied RPCs",
			giveNets: []cfgnet.Network{{
				ChainSelector: selSepolia,
				RPCs:          []cfgnet.RPC{{RPCName: "caller", HTTPURL: "http://caller"}},
			}},
			giveBase: "http://proxy.example.com",
			want: cfgnet.NewConfig([]cfgnet.Network{{
				Type:          cfgnet.NetworkTypeTestnet,
				ChainSelector: 16015286601757825753,
				RPCs: []cfgnet.RPC{
					{
						RPCName:            "default",
						PreferredURLScheme: "http",
						HTTPURL:            "http://proxy.example.com/16015286601757825753",
					},
					{
						RPCName: "caller",
						HTTPURL: "http://caller",
					},
				},
			}}),
		},
		{
			name: "does not mutate params",
			giveNets: func() []cfgnet.Network {
				rpcs := make([]cfgnet.RPC, 1, 4) // spare capacity, so an in-place insert is possible
				rpcs[0] = cfgnet.RPC{RPCName: "caller", HTTPURL: "http://caller"}
				return []cfgnet.Network{{ChainSelector: selSepolia, RPCs: rpcs}} //nolint:nlreturn
			}(),
			giveBase: "http://proxy.example.com",
			want: cfgnet.NewConfig([]cfgnet.Network{{
				Type:          cfgnet.NetworkTypeTestnet,
				ChainSelector: 16015286601757825753,
				RPCs: []cfgnet.RPC{
					{
						RPCName:            "default",
						PreferredURLScheme: "http",
						HTTPURL:            "http://proxy.example.com/16015286601757825753",
					},
					{
						RPCName: "caller",
						HTTPURL: "http://caller",
					},
				},
			}}),
		},
		{
			name: "keeps only the caller supplied RPCs when no base URL is set",
			giveNets: []cfgnet.Network{{
				ChainSelector: selSepolia,
				RPCs:          []cfgnet.RPC{{RPCName: "caller", HTTPURL: "http://caller"}},
			}},
			giveBase: "",
			want: cfgnet.NewConfig([]cfgnet.Network{{
				Type:          cfgnet.NetworkTypeTestnet,
				ChainSelector: 16015286601757825753,
				RPCs: []cfgnet.RPC{{
					RPCName: "caller",
					HTTPURL: "http://caller",
				}},
			}}),
		},
		{
			name: "preserves metadata",
			giveNets: []cfgnet.Network{{
				ChainSelector: selSepolia,
				RPCs:          []cfgnet.RPC{{RPCName: "caller", HTTPURL: "http://caller"}},
				Metadata: map[string]any{
					"network_passphrase": "Test SDF Network ; September 2015",
					"friendbot_url":      "https://friendbot.stellar.org",
				},
			}},
			want: cfgnet.NewConfig([]cfgnet.Network{{
				Type:          cfgnet.NetworkTypeTestnet,
				ChainSelector: 16015286601757825753,
				RPCs: []cfgnet.RPC{{
					RPCName: "caller",
					HTTPURL: "http://caller",
				}},
				Metadata: map[string]any{
					"network_passphrase": "Test SDF Network ; September 2015",
					"friendbot_url":      "https://friendbot.stellar.org",
				},
			}}),
		},
		{
			name:     "trims trailing slashes from the base URL",
			giveNets: []cfgnet.Network{{ChainSelector: selSepolia}},
			giveBase: "http://proxy.example.com//",
			want: cfgnet.NewConfig([]cfgnet.Network{{
				Type:          cfgnet.NetworkTypeTestnet,
				ChainSelector: 16015286601757825753,
				RPCs: []cfgnet.RPC{{
					RPCName:            "default",
					PreferredURLScheme: "http",
					HTTPURL:            "http://proxy.example.com/16015286601757825753",
				}},
			}}),
		},
		{
			name:     "no networks",
			giveNets: nil,
			giveBase: "http://proxy.example.com",
			want:     cfgnet.NewConfig([]cfgnet.Network{}),
		},
		{
			name:     "unknown chain selector",
			giveNets: []cfgnet.Network{{ChainSelector: selNonExisting}},
			giveBase: "http://proxy.example.com",
			wantErr:  "unknown chain selector",
		},
		{
			// cfgnet.Network.Validate catches this now that a shared
			// cfgnet.Network is accepted, rather than it surfacing deep inside
			// chains.LoadChains as "no RPCs found for chain selector".
			name:     "no RPCs and no base URL",
			giveNets: []cfgnet.Network{{ChainSelector: selSepolia}},
			giveBase: "",
			wantErr:  "at least one RPC is required",
		},
		{
			name:     "conflicting network type",
			giveNets: []cfgnet.Network{{ChainSelector: selSepolia, Type: cfgnet.NetworkTypeMainnet}},
			giveBase: "http://proxy.example.com",
			wantErr:  "does not match chain selector",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := BuildNetworksConfig(t.Context(), tt.giveNets, tt.giveBase, tt.giveSelectors)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func Test_newBuildConfig_Defaults(t *testing.T) {
	t.Parallel()

	cfg, err := newBuildConfig()
	require.NoError(t, err)

	assert.NotNil(t, cfg.lggr)
	assert.NotNil(t, cfg.reporter)
	assert.NotNil(t, cfg.operationRegistry)
}

func Test_BuildOptions(t *testing.T) {
	t.Parallel()

	var (
		lggr     = logger.Test(t)
		reporter = operations.NewMemoryReporter()
		registry = operations.NewOperationRegistry()
	)

	cfg, err := newBuildConfig()
	require.NoError(t, err)

	cfg.Configure([]BuildOption{
		BuildWithLogger(lggr),
		BuildWithReporter(reporter),
		BuildWithOperationRegistry(registry),
		BuildWithChainSelectors([]uint64{1, 2}),
	})

	assert.Equal(t, lggr, cfg.lggr)
	assert.Equal(t, reporter, cfg.reporter)
	assert.Equal(t, registry, cfg.operationRegistry)
	assert.Equal(t, []uint64{1, 2}, cfg.chainSelectors)
}

func Test_Build_InvalidChainSelector(t *testing.T) {
	t.Parallel()

	_, err := Build(t.Context(), BuildParams{
		Domain:            "dummy",
		Environment:       "test",
		Networks:          []cfgnet.Network{{ChainSelector: selNonExisting}},
		DefaultRPCBaseURL: "http://proxy.example.com",
	}, BuildWithLogger(logger.Test(t)))

	require.ErrorContains(t, err, "unknown chain selector")
}

func Test_Build_ChainSelectors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		networks  []cfgnet.Network
		selectors []uint64
		wantErr   string
	}{
		{
			name:      "selector missing from networks",
			networks:  []cfgnet.Network{{ChainSelector: 1}},
			selectors: []uint64{1, 42},
			wantErr:   "chain selectors [42] weren't found in build params",
		},
		{
			name:      "no networks listed",
			selectors: []uint64{42},
			wantErr:   "chain selectors [42] weren't found in build params",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := Build(t.Context(), BuildParams{
				Domain:      "dummy",
				Environment: "test",
				Networks:    tt.networks,
			}, BuildWithLogger(logger.Test(t)), BuildWithChainSelectors(tt.selectors))

			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func Test_Build_CatalogUnreachable(t *testing.T) {
	t.Parallel()

	// Fail fast rather than sitting on the gRPC retry policy. This test
	// constructs params with no networks, so the catalog fetch is the first
	// external call.
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	t.Cleanup(cancel)

	_, err := Build(ctx, BuildParams{
		Domain:      "dummy",
		Environment: "local", // "local" selects insecure transport credentials
		Catalog:     cfgenv.CatalogConfig{GRPC: "127.0.0.1:1"},
	}, BuildWithLogger(logger.Test(t)))

	require.ErrorContains(t, err, "failed to load data from catalog")
}

// Test_parseBuildParams_Golden pins the yaml tags on BuildParams and
// cfgnet.Network, which are the public contract of the file format that
// BuildFromYAML accepts. Renaming a tag breaks callers silently otherwise.
//
// The JSON case is not incidental: YAML 1.2 is a superset of JSON, so the same
// decoder accepts both, and this fixes that as intended behaviour rather than a
// coincidence.
func Test_parseBuildParams_Golden(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		giveFile string
	}{
		{name: "yaml document", giveFile: "build_params.yaml"},
		{name: "json document", giveFile: "build_params.json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := os.ReadFile(filepath.Join("testdata", tt.giveFile))
			require.NoError(t, err)

			got, err := parseBuildParams(data)
			require.NoError(t, err)

			assert.Equal(t, wantGoldenParams(), got)
		})
	}
}

// Test_parseBuildParams_Golden_BuildableNetworks carries the golden fixture
// through to the cfgnet.Config that Build hands to chains.LoadChains, so the
// fixture is pinned as something that actually builds rather than merely
// decoding.
func Test_parseBuildParams_Golden_BuildableNetworks(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join("testdata", "build_params.yaml"))
	require.NoError(t, err)

	params, err := parseBuildParams(data)
	require.NoError(t, err)

	got, err := BuildNetworksConfig(t.Context(), params.Networks, params.DefaultRPCBaseURL, nil)
	require.NoError(t, err)

	// Every network gets the derived default RPC first, ahead of any explicit ones.
	sepolia, err := got.NetworkBySelector(selSepolia)
	require.NoError(t, err)
	assert.Equal(t, []cfgnet.RPC{{
		RPCName:            "default",
		PreferredURLScheme: "http",
		HTTPURL:            "http://rpc-proxy.example.com/16015286601757825753",
	}}, sepolia.RPCs)

	mainnet, err := got.NetworkBySelector(selMainnet)
	require.NoError(t, err)
	require.Len(t, mainnet.RPCs, 2)
	assert.Equal(t, "default", mainnet.RPCs[0].RPCName)
	assert.Equal(t, "mainnet-rpc", mainnet.RPCs[1].RPCName)

	// Metadata survives decoding in a shape the family decoders accept.
	zksync, err := got.NetworkBySelector(selZkSyncSepolia)
	require.NoError(t, err)
	md, err := cfgnet.DecodeMetadata[cfgnet.EVMMetadata](zksync.Metadata)
	require.NoError(t, err)
	require.NotNil(t, md.IsZkSync)
	assert.True(t, *md.IsZkSync)
}

func Test_parseBuildParams_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    string
		wantErr string
	}{
		{
			// Strict decoding: a mistyped key must not silently leave a zero value.
			name:    "unknown field",
			give:    "domain: exemplar\ndomian: typo\n",
			wantErr: "field domian not found in type environment.BuildParams",
		},
		{
			name:    "unknown nested field",
			give:    "networks:\n  - chain_selector: 1\n    rpc: nope\n",
			wantErr: "field rpc not found",
		},
		{
			name:    "unknown notifications field",
			give:    "notifications:\n  slack:\n    chanels: []\n",
			wantErr: "field chanels not found",
		},
		{
			name:    "empty document",
			give:    "",
			wantErr: "document is empty",
		},
		{
			name:    "malformed yaml",
			give:    "domain: [unterminated\n",
			wantErr: "failed to decode build params",
		},
		{
			name:    "wrong scalar type",
			give:    "networks:\n  - chain_selector: not-a-number\n",
			wantErr: "cannot unmarshal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseBuildParams([]byte(tt.give))
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func Test_missingSelectors(t *testing.T) {
	t.Parallel()

	networks := []cfgnet.Network{{ChainSelector: 1}, {ChainSelector: 2}}

	tests := []struct {
		name      string
		selectors []uint64
		want      []uint64
	}{
		{
			name:      "all listed",
			selectors: []uint64{1, 2},
			want:      []uint64{},
		},
		{
			name:      "some missing",
			selectors: []uint64{1, 3, 4},
			want:      []uint64{3, 4},
		},
		{
			name:      "all missing",
			selectors: []uint64{5},
			want:      []uint64{5},
		},
		{
			name:      "no selectors",
			selectors: []uint64{},
			want:      []uint64{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := missingSelectors(networks, tt.selectors)

			assert.Equal(t, tt.want, got)
		})
	}
}

// --- helpers -----------------------------------------------------------------

// wantGoldenParams is the decoded form of both testdata/build_params.yaml and
// testdata/build_params.json. The two fixtures describe the same environment in
// different encodings, so decoding either must produce this value.
func wantGoldenParams() BuildParams {
	return BuildParams{
		Domain:            "exemplar",
		Environment:       "staging",
		DefaultRPCBaseURL: "http://rpc-proxy.example.com",
		Networks: []cfgnet.Network{
			{ChainSelector: selSepolia},
			{
				ChainSelector: selMainnet,
				Type:          cfgnet.NetworkTypeMainnet,
				BlockExplorer: cfgnet.BlockExplorer{
					Type:   "etherscan",
					URL:    "https://api.etherscan.io/api",
					APIKey: "abc123",
				},
				RPCs: []cfgnet.RPC{{
					RPCName:            "mainnet-rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "https://mainnet.example.com",
					WSURL:              "wss://mainnet.example.com",
				}},
			},
			{
				ChainSelector: selZkSyncSepolia,
				Metadata:      map[string]any{"is_zksync": true},
			},
		},
		Onchain: cfgenv.OnchainConfig{
			KMS: cfgenv.KMSConfig{KeyID: "f1a2b3c4", KeyRegion: "us-west-1"},
			EVM: cfgenv.EVMConfig{
				DeployerKey: "0xabc",
				Seth: &cfgenv.SethConfig{
					ConfigFilePath:  "/tmp/config",
					GethWrapperDirs: []string{"./dir1"},
				},
			},
			Solana: cfgenv.SolanaConfig{
				WalletKey:       "0xbcd",
				ProgramsDirPath: "/tmp/program",
			},
		},
		Catalog: cfgenv.CatalogConfig{
			GRPC: "catalog.example.com:443",
			Auth: &cfgenv.CatalogAuthConfig{
				KMSKeyID:     "alias/catalog",
				KMSKeyRegion: "us-west-1",
			},
		},
		Notifications: &cfgnotifications.Config{
			Slack: cfgnotifications.SlackConfig{Channels: []cfgnotifications.SlackChannel{
				{
					Name:              "#exemplar-signers",
					SlackHandles:      []string{"@exemplar-signers"},
					TimelockContracts: []cfgnotifications.ContractRef{{ChainSelector: selSepolia, Address: "0xabc"}},
					MCMContracts:      []cfgnotifications.ContractRef{{ChainSelector: selSepolia, Address: "0xdef"}},
				},
				{Name: "#mcms-proposals"},
			}},
		},
	}
}

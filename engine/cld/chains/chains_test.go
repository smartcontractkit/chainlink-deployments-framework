package chains

import (
	"encoding/hex"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gagliardetto/solana-go"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	cfgenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
)

func Test_LoadChains(t *testing.T) {
	t.Parallel()

	var (
		fakeSrv = newFakeRPCServer(t)

		evmSelector    = chainsel.TEST_1000.Selector
		solanaSelector = chainsel.TEST_22222222222222222222222222222222222222222222.Selector
		aptosSelector  = chainsel.APTOS_LOCALNET.Selector
		tronSelector   = chainsel.TRON_TESTNET_NILE.Selector
		suiSelector    = chainsel.SUI_LOCALNET.Selector
		tonSelector    = chainsel.TON_TESTNET.Selector
	)

	networks := []cfgnet.Network{
		{
			Type:          cfgnet.NetworkTypeTestnet,
			ChainSelector: evmSelector,
			RPCs: []cfgnet.RPC{
				{
					RPCName:            "evm_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            fakeSrv.URL + "/evm",
					WSURL:              "ws://evm-ws",
				},
			},
		},
		{
			Type:          cfgnet.NetworkTypeTestnet,
			ChainSelector: solanaSelector,
			RPCs: []cfgnet.RPC{
				{
					RPCName:            "solana_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "http://solana-rpc",
					WSURL:              "ws://solana-ws",
				},
			},
		},
		{
			Type:          cfgnet.NetworkTypeTestnet,
			ChainSelector: aptosSelector,
			RPCs: []cfgnet.RPC{
				{
					RPCName:            "aptos_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "http://aptos-rpc",
					WSURL:              "",
				},
			},
		},
		{
			Type:          cfgnet.NetworkTypeTestnet,
			ChainSelector: tronSelector,
			RPCs: []cfgnet.RPC{
				{
					RPCName:            "tron_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "http://tron-rpc",
					WSURL:              "",
				},
			},
		},
		{
			Type:          cfgnet.NetworkTypeTestnet,
			ChainSelector: suiSelector,
			RPCs: []cfgnet.RPC{
				{
					RPCName:            "sui_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "http://sui-rpc",
					WSURL:              "",
				},
			},
		},
		{
			Type:          cfgnet.NetworkTypeTestnet,
			ChainSelector: tonSelector,
			RPCs: []cfgnet.RPC{
				{
					RPCName:            "ton_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "liteserver://publickey@host:port",
					WSURL:              "",
				},
			},
		},
	}
	networksConfig := cfgnet.NewConfig(networks)

	// Generate a random EVM private key for testing
	evmKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	evmKeyHex := hex.EncodeToString(crypto.FromECDSA(evmKey))

	// Generate a random Solana keypair for testing
	solKey, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)

	// Create a temporary directory for Solana programs
	solProgramsPath := t.TempDir()

	// Compose onchain config
	onchainConfig := cfgenv.OnchainConfig{
		EVM: cfgenv.EVMConfig{
			DeployerKey: evmKeyHex,
			Seth:        &cfgenv.SethConfig{},
		},
		Solana: cfgenv.SolanaConfig{
			WalletKey:       solKey.String(),
			ProgramsDirPath: solProgramsPath,
		},
		Aptos: cfgenv.AptosConfig{
			DeployerKey: "0xE4FD0E90D32CB98DC6AD64516A421E8C2731870217CDBA64203CEB158A866304",
		},
		Tron: cfgenv.TronConfig{
			DeployerKey: evmKeyHex, // Uses the same key as the EVM chain
		},
		Sui: cfgenv.SuiConfig{
			DeployerKey: "0xA1B2C3D4E5F60718293A4B5C6D7E8F90123456789ABCDEF0123456789ABCDEF0", // Mock private key
		},
		Ton: cfgenv.TonConfig{
			DeployerKey:   "0b1f7dbb19112fdac53344cf49731e41bfc420ac6a71d38c89fb38d04a6563d99aa3d1fa430550e8de5171ec55453b4e048c1701cadfa56726d489c56d67bab3", // Mock private key
			WalletVersion: "V4R2",
		},
	}

	tests := []struct {
		name              string
		giveNetworkConfig *cfgnet.Config
		giveOnchainConfig cfgenv.OnchainConfig
		giveSelectors     []uint64
		wantCount         int
		wantErr           string
	}{
		{
			name:              "loads all valid chains",
			giveNetworkConfig: networksConfig,
			giveOnchainConfig: onchainConfig,
			giveSelectors:     []uint64{evmSelector, solanaSelector, aptosSelector, tronSelector, suiSelector, tonSelector},
			wantCount:         6,
		},
		{
			name:              "fails with unknown selector",
			giveNetworkConfig: networksConfig,
			giveOnchainConfig: onchainConfig,
			giveSelectors:     []uint64{9999999},
			wantErr:           "unable to get chain family for selector 9999999",
		},
		{
			name:              "skips selector not found in networks",
			giveNetworkConfig: networksConfig,
			giveOnchainConfig: onchainConfig,
			giveSelectors:     []uint64{evmSelector, chainsel.TEST_90000001.Selector},
			wantErr:           "failed to load 1 out of 2 chains",
		},
		{
			name: "fails to load a chain",
			giveNetworkConfig: cfgnet.NewConfig([]cfgnet.Network{
				{
					Type:          cfgnet.NetworkTypeTestnet,
					ChainSelector: evmSelector,
					RPCs: []cfgnet.RPC{
						{
							RPCName:            "evm_rpc",
							PreferredURLScheme: "http",
							HTTPURL:            "invalid.url", // Use an invalid URL to force an error
							WSURL:              "ws://evm-ws",
						},
					},
				},
			}),
			giveOnchainConfig: onchainConfig,
			giveSelectors:     []uint64{evmSelector},
			wantCount:         0,
			wantErr:           "failed to load 1 out of 1 chains",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				ctx  = t.Context()
				lggr = logger.Test(t)
			)

			cfg := &config.Config{
				Networks: tt.giveNetworkConfig,
				Env: &cfgenv.Config{
					Onchain: tt.giveOnchainConfig,
				},
			}

			chains, err := LoadChains(ctx, lggr, cfg, tt.giveSelectors)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Len(t, maps.Collect(chains.All()), tt.wantCount)
			}
		})
	}
}

func Test_chainLoaderAptos_Load(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Test chain selector for Aptos mainnet
	var aptosSelector = chainsel.APTOS_LOCALNET.Selector

	// Create test network config
	networkCfg := cfgnet.NewConfig([]cfgnet.Network{
		{
			Type:          cfgnet.NetworkTypeMainnet,
			ChainSelector: aptosSelector,
			RPCs: []cfgnet.RPC{
				{
					RPCName:            "aptos_localnet_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "https://fullnode.localnet.aptoslabs.com/v1",
					WSURL:              "", // This is not used for Aptos
				},
			},
		},
		{
			Type:          cfgnet.NetworkTypeTestnet,
			ChainSelector: 999999, // Different chain for testing
			RPCs: []cfgnet.RPC{
				{
					RPCName:            "other_chain_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "https://other.rpc.com",
					WSURL:              "",
				},
			},
		},
	})

	// Create test secrets with Aptos deployer key
	onchainConfig := cfgenv.OnchainConfig{
		Aptos: cfgenv.AptosConfig{
			DeployerKey: "0xE4FD0E90D32CB98DC6AD64516A421E8C2731870217CDBA64203CEB158A866304", // Mock private key
		},
	}

	tests := []struct {
		name              string
		giveSelector      uint64
		giveNetworkConfig *cfgnet.Config
		giveOnchainConfig cfgenv.OnchainConfig
		wantErr           string
	}{
		{
			name:              "successful load",
			giveSelector:      aptosSelector,
			giveNetworkConfig: networkCfg,
			giveOnchainConfig: onchainConfig,
		},
		{
			name:              "network not found",
			giveSelector:      88888888, // Non-existent chain selector
			giveNetworkConfig: networkCfg,
			giveOnchainConfig: onchainConfig,
			wantErr:           "not found in configuration",
		},
		{
			name:         "no RPCs configured",
			giveSelector: 777777,
			giveNetworkConfig: cfgnet.NewConfig([]cfgnet.Network{
				{
					Type:          cfgnet.NetworkTypeTestnet,
					ChainSelector: 777777,
					RPCs:          []cfgnet.RPC{}, // Empty RPCs slice
				},
			}),
			giveOnchainConfig: onchainConfig,
			wantErr:           "no RPCs found for chain selector: 777777",
		},
		{
			name:              "empty private key",
			giveSelector:      aptosSelector,
			giveNetworkConfig: networkCfg,
			giveOnchainConfig: cfgenv.OnchainConfig{
				Aptos: cfgenv.AptosConfig{
					DeployerKey: "",
				},
			},
			wantErr: "failed to initialize Aptos chain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create the chain loader
			loader := newChainLoaderAptos(tt.giveNetworkConfig, tt.giveOnchainConfig)
			require.NotNil(t, loader)

			// Test the Load method
			chain, err := loader.Load(ctx, tt.giveSelector)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, chain)
			} else {
				require.NoError(t, err)
				require.NotNil(t, chain)

				// Verify that the chain has the correct selector
				assert.Equal(t, tt.giveSelector, chain.ChainSelector())

				// Verify that the chain family is Aptos
				assert.Equal(t, "aptos", chain.Family())
			}
		})
	}
}

func Test_chainLoaderSui_Load(t *testing.T) {
	t.Parallel()

	var suiSelector = chainsel.SUI_LOCALNET.Selector

	networkCfg := cfgnet.NewConfig([]cfgnet.Network{
		{
			Type:          cfgnet.NetworkTypeMainnet,
			ChainSelector: suiSelector,
			RPCs: []cfgnet.RPC{
				{
					RPCName:            "sui_localnet_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "https://fullnode.localnet.suilabs.com/v1",
					WSURL:              "", // This is not used for Sui
				},
			},
		},
		{
			Type:          cfgnet.NetworkTypeTestnet,
			ChainSelector: 999999, // Different chain for testing
			RPCs: []cfgnet.RPC{
				{
					RPCName:            "other_chain_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "https://other.rpc.com",
					WSURL:              "",
				},
			},
		},
	})

	onchainConfig := cfgenv.OnchainConfig{
		Sui: cfgenv.SuiConfig{
			DeployerKey: "0xA1B2C3D4E5F60718293A4B5C6D7E8F90123456789ABCDEF0123456789ABCDEF0", // Mock private key
		},
	}

	tests := []struct {
		name              string
		giveSelector      uint64
		giveNetworkConfig *cfgnet.Config
		giveOnchainConfig cfgenv.OnchainConfig
		wantErr           string
	}{
		{
			name:              "successful load",
			giveSelector:      suiSelector,
			giveNetworkConfig: networkCfg,
			giveOnchainConfig: onchainConfig,
		},
		{
			name:              "network not found",
			giveSelector:      88888888, // Non-existent chain selector
			giveNetworkConfig: networkCfg,
			giveOnchainConfig: onchainConfig,
			wantErr:           "not found in configuration",
		},
		{
			name:         "no RPCs configured",
			giveSelector: 777777,
			giveNetworkConfig: cfgnet.NewConfig([]cfgnet.Network{
				{
					Type:          cfgnet.NetworkTypeTestnet,
					ChainSelector: 777777,
					RPCs:          []cfgnet.RPC{},
				},
			}),
			giveOnchainConfig: onchainConfig,
			wantErr:           "no RPCs found for chain selector: 777777",
		},
		{
			name:              "empty private key",
			giveSelector:      suiSelector,
			giveNetworkConfig: networkCfg,
			giveOnchainConfig: cfgenv.OnchainConfig{
				Sui: cfgenv.SuiConfig{
					DeployerKey: "",
				},
			},
			wantErr: "failed to initialize Sui chain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			loader := newChainLoaderSui(tt.giveNetworkConfig, tt.giveOnchainConfig)
			require.NotNil(t, loader)

			ctx := t.Context()

			chain, err := loader.Load(ctx, tt.giveSelector)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, chain)
			} else {
				require.NoError(t, err)
				require.NotNil(t, chain)

				assert.Equal(t, tt.giveSelector, chain.ChainSelector())
				assert.Equal(t, "sui", chain.Family())
			}
		})
	}
}

func Test_ChainLoaderSolana_Load(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Test chain selector for Solana localnet
	var solanaSelector = chainsel.TEST_22222222222222222222222222222222222222222222.Selector

	// Create test network config
	networkCfg := cfgnet.NewConfig([]cfgnet.Network{
		{
			Type:          cfgnet.NetworkTypeTestnet,
			ChainSelector: solanaSelector,
			RPCs: []cfgnet.RPC{
				{
					RPCName:            "solana_localnet_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "http://localhost:8899",
					WSURL:              "ws://localhost:8900",
				},
			},
		},
		{
			Type:          cfgnet.NetworkTypeTestnet,
			ChainSelector: 123456, // Different chain for testing
			RPCs: []cfgnet.RPC{
				{
					RPCName:            "other_chain_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "http://other.rpc.com",
					WSURL:              "ws://other.ws.com",
				},
			},
		},
	})

	// Create test secrets with Solana wallet key and program path
	privKey, err := solana.NewRandomPrivateKey()
	require.NoError(t, err)
	programsPath := t.TempDir()

	onchainConfig := cfgenv.OnchainConfig{
		Solana: cfgenv.SolanaConfig{
			WalletKey:       privKey.String(), // Mock private key
			ProgramsDirPath: programsPath,     // Mock path
		},
	}

	tests := []struct {
		name              string
		giveSelector      uint64
		giveNetworkConfig *cfgnet.Config
		giveOnchainConfig cfgenv.OnchainConfig
		wantErr           string
	}{
		{
			name:              "successful load",
			giveSelector:      solanaSelector,
			giveNetworkConfig: networkCfg,
			giveOnchainConfig: onchainConfig,
		},
		{
			name:              "network not found",
			giveSelector:      99999999, // Non-existent chain selector
			giveNetworkConfig: networkCfg,
			giveOnchainConfig: onchainConfig,
			wantErr:           "not found in configuration",
		},
		{
			name:         "no RPCs configured",
			giveSelector: 888888,
			giveNetworkConfig: cfgnet.NewConfig([]cfgnet.Network{
				{
					Type:          cfgnet.NetworkTypeTestnet,
					ChainSelector: 888888,
					RPCs:          []cfgnet.RPC{}, // Empty RPCs slice
				},
			}),
			giveOnchainConfig: onchainConfig,
			wantErr:           "no RPCs found for chain selector: 888888",
		},
		{
			name:              "empty private key",
			giveSelector:      solanaSelector,
			giveNetworkConfig: networkCfg,
			giveOnchainConfig: cfgenv.OnchainConfig{
				Solana: cfgenv.SolanaConfig{
					WalletKey:       "",
					ProgramsDirPath: programsPath,
				},
			},
			wantErr: "failed to initialize Solana chain",
		},
		{
			name:              "invalid program path",
			giveSelector:      solanaSelector,
			giveNetworkConfig: networkCfg,
			giveOnchainConfig: cfgenv.OnchainConfig{
				Solana: cfgenv.SolanaConfig{
					WalletKey:       privKey.String(),
					ProgramsDirPath: "asaa",
				},
			},
			wantErr: "required file does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			loader := newChainLoaderSolana(tt.giveNetworkConfig, tt.giveOnchainConfig)
			require.NotNil(t, loader)

			chain, err := loader.Load(ctx, tt.giveSelector)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, chain)
			} else {
				require.NoError(t, err)
				require.NotNil(t, chain)

				assert.Equal(t, tt.giveSelector, chain.ChainSelector())
				assert.Equal(t, "solana", chain.Family())
			}
		})
	}
}

func Test_ChainLoaderEVM_Load(t *testing.T) {
	t.Parallel()

	// Test chain selector for Ethereum Sepolia testnet
	var (
		ctx                 = t.Context()
		fakeSrv             = newFakeRPCServer(t)
		evmSelector         = chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector
		zksyncChainSelector = chainsel.ETHEREUM_TESTNET_SEPOLIA_ZKSYNC_1.Selector
	)

	// Create test network config
	networkCfg := cfgnet.NewConfig([]cfgnet.Network{
		{
			Type:          cfgnet.NetworkTypeTestnet,
			ChainSelector: evmSelector,
			RPCs: []cfgnet.RPC{
				{
					RPCName:            "sepolia_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            fakeSrv.URL + "/evm",
				},
			},
		},
		{
			Type:          cfgnet.NetworkTypeTestnet,
			ChainSelector: zksyncChainSelector,
			RPCs: []cfgnet.RPC{
				{
					RPCName:            "zksync_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            fakeSrv.URL + "/evm",
				},
			},
		},
		{
			Type:          cfgnet.NetworkTypeTestnet,
			ChainSelector: 999999, // Different chain for testing
			RPCs: []cfgnet.RPC{
				{
					RPCName:            "other_chain_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "https://other.rpc.com",
					WSURL:              "wss://other.ws.com",
				},
			},
		},
	})

	// Setup the private key
	privKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	privKeyHex := hex.EncodeToString(crypto.FromECDSA(privKey))

	// Create test secrets with EVM wallet key
	onchainConfig := cfgenv.OnchainConfig{
		EVM: cfgenv.EVMConfig{
			DeployerKey: privKeyHex,
		},
	}

	tests := []struct {
		name              string
		giveSelector      uint64
		giveConfig        *cfgnet.Config
		giveOnchainConfig cfgenv.OnchainConfig
		wantErr           string
	}{
		{
			name:              "successful load: evm",
			giveSelector:      evmSelector,
			giveConfig:        networkCfg,
			giveOnchainConfig: onchainConfig,
		},
		{
			name:              "successful load: zksync",
			giveSelector:      zksyncChainSelector,
			giveConfig:        networkCfg,
			giveOnchainConfig: onchainConfig,
		},
		{
			name:              "network not found",
			giveSelector:      88888888, // Non-existent chain selector
			giveConfig:        networkCfg,
			giveOnchainConfig: onchainConfig,
			wantErr:           "not found in configuration",
		},
		{
			name:         "no RPCs configured",
			giveSelector: 777777,
			giveConfig: cfgnet.NewConfig([]cfgnet.Network{
				{
					Type:          cfgnet.NetworkTypeTestnet,
					ChainSelector: 777777,
					RPCs:          []cfgnet.RPC{}, // Empty RPCs slice
				},
			}),
			giveOnchainConfig: onchainConfig,
			wantErr:           "no RPCs found for chain selector: 777777",
		},
		{
			name:         "fail to initialize evm chain",
			giveSelector: evmSelector,
			giveConfig:   networkCfg,
			giveOnchainConfig: cfgenv.OnchainConfig{
				EVM: cfgenv.EVMConfig{
					DeployerKey: privKeyHex,
					Seth: &cfgenv.SethConfig{
						GethWrapperDirs: []string{"some/valid/gethwrapper"},
						ConfigFilePath:  "/invalid", // Must be empty or a valid path
					},
				},
			},
			wantErr: fmt.Sprintf("failed to initialize chain %d", evmSelector),
		},
		{
			name:         "fail to initialize zksync chain",
			giveSelector: zksyncChainSelector,
			giveConfig:   networkCfg,
			giveOnchainConfig: cfgenv.OnchainConfig{
				EVM: cfgenv.EVMConfig{
					DeployerKey: privKeyHex,
					Seth: &cfgenv.SethConfig{
						GethWrapperDirs: []string{"some/valid/gethwrapper"},
						ConfigFilePath:  "/invalid", // Must be empty or a valid path
					},
				},
			},
			wantErr: fmt.Sprintf("failed to initialize chain %d", zksyncChainSelector),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create the chain loader
			loader := newChainLoaderEVM(tt.giveConfig, tt.giveOnchainConfig, logger.Test(t))
			require.NotNil(t, loader)

			// Test the Load method
			chain, err := loader.Load(ctx, tt.giveSelector)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, chain)
			} else {
				require.NoError(t, err)
				require.NotNil(t, chain)

				// Verify that the chain has the correct selector
				assert.Equal(t, tt.giveSelector, chain.ChainSelector())

				// Verify that the chain family is EVM
				assert.Equal(t, "evm", chain.Family())
			}
		})
	}
}

func Test_ChainLoaderTron_Load(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	var tronSelector = chainsel.TRON_TESTNET_NILE.Selector

	networks := cfgnet.NewConfig([]cfgnet.Network{
		{
			Type:          cfgnet.NetworkTypeTestnet,
			ChainSelector: tronSelector,
			RPCs: []cfgnet.RPC{
				{
					RPCName:            "tron_testnet_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "https://nile.trongrid.io",
				},
			},
		},
		{
			Type:          cfgnet.NetworkTypeTestnet,
			ChainSelector: 999999, // Different chain for testing
			RPCs: []cfgnet.RPC{
				{
					RPCName:            "other_chain_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "https://other.rpc.com",
					WSURL:              "",
				},
			},
		},
	})

	onchainConfig := cfgenv.OnchainConfig{
		Tron: cfgenv.TronConfig{
			DeployerKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", // Mock private key
		},
	}

	tests := []struct {
		name              string
		giveSelector      uint64
		giveNetworkConfig *cfgnet.Config
		giveOnchainConfig cfgenv.OnchainConfig
		wantErr           string
	}{
		{
			name:              "successful load",
			giveSelector:      tronSelector,
			giveNetworkConfig: networks,
			giveOnchainConfig: onchainConfig,
		},
		{
			name:              "network not found",
			giveSelector:      88888888, // Non-existent chain selector
			giveNetworkConfig: networks,
			giveOnchainConfig: onchainConfig,
			wantErr:           "not found in configuration",
		},
		{
			name:         "no RPCs configured",
			giveSelector: 777777,
			giveNetworkConfig: cfgnet.NewConfig([]cfgnet.Network{
				{
					Type:          cfgnet.NetworkTypeTestnet,
					ChainSelector: 777777,
					RPCs:          []cfgnet.RPC{}, // Empty RPCs slice
				},
			}),
			giveOnchainConfig: onchainConfig,
			wantErr:           "no RPCs found for chain selector: 777777",
		},
		{
			name:              "empty private key",
			giveSelector:      tronSelector,
			giveNetworkConfig: networks,
			giveOnchainConfig: cfgenv.OnchainConfig{
				Tron: cfgenv.TronConfig{
					DeployerKey: "", // Empty key
				},
			},
			wantErr: "failed to create TRON account generator: failed to parse private key: invalid length, need 256 bits",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			loader := newChainLoaderTron(tt.giveNetworkConfig, tt.giveOnchainConfig)
			require.NotNil(t, loader)

			chain, err := loader.Load(ctx, tt.giveSelector)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, chain)
			} else {
				require.NoError(t, err)
				require.NotNil(t, chain)

				assert.Equal(t, tt.giveSelector, chain.ChainSelector())
				assert.Equal(t, "tron", chain.Family())
			}
		})
	}
}

// newFakeRPCServer returns a fake RPC server which always answers with a valid `eth_blockNumber“
// response for evm and a valid config for ton.
//
// When the test is done, the server is closed automatically.
func newFakeRPCServer(t *testing.T) *httptest.Server {
	t.Helper()

	evmResponse := `{"jsonrpc":"2.0","id":1,"result":"0x1"}`
	tonResponse := `{"liteservers":[{"ip":822907680,"port":27842,"provided":"Swiss","id":{"@type":"pub.ed25519","key":"sU7QavX2F964iI9oToP9gffQpCQIoOLppeqL/pdPvpM="}},{"ip":-1468571697,"port":27787,"provided":"Swiss","id":{"@type":"pub.ed25519","key":"Y/QVf6G5VDiKTZOKitbFVm067WsuocTN8Vg036A4zGk="}},{"ip":-1468575011,"port":51088,"provided":"Swiss","id":{"@type":"pub.ed25519","key":"Sy5ghr3EahQd/1rDayzZXt5+inlfF+7kLfkZDJcU/ek="}},{"ip":1844203589,"port":49913,"provided":"Neo","id":{"@type":"pub.ed25519","key":"AxFZRHVD1qIO9Fyva52P4vC3tRvk8ac1KKOG0c6IVio="}},{"ip":1844203537,"port":4330,"provided":"Neo","id":{"@type":"pub.ed25519","key":"IraRAKcsECFUwFRKXF23YeXAxrDMVWobbw8Hpb2m7Aw="}},{"ip":1047529523,"port":36005,"provided":"Neo","id":{"@type":"pub.ed25519","key":"39EIFy/IaCjgJSu1WvHJjD/mNbGI0c+bM/hrkCjzfrw="}},{"ip":1097633201,"port":17439,"id":{"@type":"pub.ed25519","key":"0MIADpLH4VQn+INHfm0FxGiuZZAA8JfTujRqQugkkA8="}},{"ip":1091956407,"port":16351,"id":{"@type":"pub.ed25519","key":"Mf/JGvcWAvcrN3oheze8RF/ps6p7oL6ifrIzFmGQFQ8="}}],"dht":{"a":3,"k":3,"static_nodes":{"nodes":[{"@type":"dht.node","id":{"@type":"pub.ed25519","key":"MKaJGWvZ8AOX/VWV2rJglcEJh07MKjgZiy5Mel6L2Xk="},"addr_list":{"@type":"adnl.addressList","addrs":[{"@type":"adnl.address.udp","ip":1053990413,"port":7694}],"version":0,"reinit_date":0,"priority":0,"expire_at":0},"version":-1,"signature":"GJiGFKV5sfOJuKNj13Bo7TfEk8A7NMyAruzj1nwWvfSlGSmnYUhUa9LmyHU7XyrlKcLmYC+MU0h5SctUkctsCA=="},{"@type":"dht.node","id":{"@type":"pub.ed25519","key":"+htkM588jJXXidOs64fuGj/jiZiCY/AG3EljcugfOs0="},"addr_list":{"@type":"adnl.addressList","addrs":[{"@type":"adnl.address.udp","ip":-2087062611,"port":17750}],"version":0,"reinit_date":0,"priority":0,"expire_at":0},"version":-1,"signature":"K9zywmobUDVsYlXdJwTg1b9xtbSNueX6cizpI26xD71ntQbAURyd6TLXUzEezdYZYSzTvK0NJoL2VojqBgFBAw=="},{"@type":"dht.node","id":{"@type":"pub.ed25519","key":"LFnKVKTO+GYsOBrTH2xaVAGsOGEgSNGo0TRdDZmBeL4="},"addr_list":{"@type":"adnl.addressList","addrs":[{"@type":"adnl.address.udp","ip":-1874058059,"port":21533}],"version":0,"reinit_date":0,"priority":0,"expire_at":0},"version":-1,"signature":"UqP2Mvgtu4f/70NdsFekQ3jarIpcBQDamGn2jYhocw4yWTfHaxnP6m4FMh+qSe07q7e0DbkBjwKnGxu+YfIBCQ=="},{"@type":"dht.node","id":{"@type":"pub.ed25519","key":"pdeuI0a/RhBqgkQxn+6+J2EMdcpTi0WhhaR8Q3/+u+4="},"addr_list":{"@type":"adnl.addressList","addrs":[{"@type":"adnl.address.udp","ip":-2087065405,"port":13654}],"version":0,"reinit_date":0,"priority":0,"expire_at":0},"version":-1,"signature":"Ph5LdoX9NyBFDqF5YLS2jNdb/omco3pgmCt0iI99tS95Mcic+WFUsH9nt0zyzy1dd8D75vR952go2HMHpKYaBA=="},{"@type":"dht.node","id":{"@type":"pub.ed25519","key":"GIpxz5qHuKu/kNZl7Zc/w8LbxYamzCBKpbT9+FAnIJU="},"addr_list":{"@type":"adnl.addressList","addrs":[{"@type":"adnl.address.udp","ip":1495755553,"port":2810}],"version":0,"reinit_date":0,"priority":0,"expire_at":0},"version":-1,"signature":"RvSTi5eaZ7wH1ap0EfffnT66CccrJA2JZtwb8tYOlxhaxtlRqwPOr0Om1pWBmbItwwAGScCxoYZFtdGbEImbDw=="}],"@type":"dht.nodes"},"@type":"dht.config.global"},"@type":"config.global","validator":{"zero_state":{"file_hash":"Z+IKwYS54DmmJmesw/nAD5DzWadnOCMzee+kdgSYDOg=","seqno":0,"root_hash":"gj+B8wb/AmlPk1z1AhVI484rhrUpgSr2oSFIh56VoSg=","workchain":-1,"shard":-9223372036854775808},"@type":"validator.config.global","init_block":{"workchain":-1,"shard":-9223372036854775808,"seqno":17908219,"root_hash":"y6qWqhCnLgzWHjUFmXysaiOljuK5xVoCRMLzUwGInVM=","file_hash":"Y/GziXxwuYte0AM4WT7tTWsCx+6rcfLpGmRaEQwhUKI="},"hardforks":[{"file_hash":"jF3RTD+OyOoP+OI9oIjdV6M8EaOh9E+8+c3m5JkPYdg=","seqno":5141579,"root_hash":"6JSqIYIkW7y8IorxfbQBoXiuY3kXjcoYgQOxTJpjXXA=","workchain":-1,"shard":-9223372036854775808},{"file_hash":"WrNoMrn5UIVPDV/ug/VPjYatvde8TPvz5v1VYHCLPh8=","seqno":5172980,"root_hash":"054VCNNtUEwYGoRe1zjH+9b1q21/MeM+3fOo76Vcjes=","workchain":-1,"shard":-9223372036854775808},{"file_hash":"xRaxgUwgTXYFb16YnR+Q+VVsczLl6jmYwvzhQ/ncrh4=","seqno":5176527,"root_hash":"SoPLqMe9Dz26YJPOGDOHApTSe5i0kXFtRmRh/zPMGuI=","workchain":-1,"shard":-9223372036854775808}]}}}`

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/ton" {
			_, _ = w.Write([]byte(tonResponse))
		} else {
			_, _ = w.Write([]byte(evmResponse))
		}
	})

	srv := httptest.NewServer(handler)

	t.Cleanup(func() {
		srv.Close()
	})

	return srv
}

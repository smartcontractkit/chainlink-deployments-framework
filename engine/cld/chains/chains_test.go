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
	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cldf_config_env "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
	cldf_config_network "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	cldf_environment "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
)

func Test_LoadChains(t *testing.T) {
	t.Parallel()

	var (
		fakeSrv = newFakeRPCServer(t)

		evmSelector    = chain_selectors.TEST_1000.Selector
		solanaSelector = chain_selectors.TEST_22222222222222222222222222222222222222222222.Selector
		aptosSelector  = chain_selectors.APTOS_LOCALNET.Selector
		tronSelector   = chain_selectors.TRON_TESTNET_NILE.Selector
		suiSelector    = chain_selectors.SUI_LOCALNET.Selector
	)

	networks := []cldf_config_network.Network{
		{
			Type:          cldf_config_network.NetworkTypeTestnet,
			ChainSelector: evmSelector,
			RPCs: []cldf_config_network.RPC{
				{
					RPCName:            "evm_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            fakeSrv.URL,
					WSURL:              "ws://evm-ws",
				},
			},
		},
		{
			Type:          cldf_config_network.NetworkTypeTestnet,
			ChainSelector: solanaSelector,
			RPCs: []cldf_config_network.RPC{
				{
					RPCName:            "solana_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "http://solana-rpc",
					WSURL:              "ws://solana-ws",
				},
			},
		},
		{
			Type:          cldf_config_network.NetworkTypeTestnet,
			ChainSelector: aptosSelector,
			RPCs: []cldf_config_network.RPC{
				{
					RPCName:            "aptos_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "http://aptos-rpc",
					WSURL:              "",
				},
			},
		},
		{
			Type:          cldf_config_network.NetworkTypeTestnet,
			ChainSelector: tronSelector,
			RPCs: []cldf_config_network.RPC{
				{
					RPCName:            "tron_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "http://tron-rpc",
					WSURL:              "",
				},
			},
		},
		{
			Type:          cldf_config_network.NetworkTypeTestnet,
			ChainSelector: suiSelector,
			RPCs: []cldf_config_network.RPC{
				{
					RPCName:            "sui_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "http://sui-rpc",
					WSURL:              "",
				},
			},
		},
	}
	networksConfig := cldf_config_network.NewConfig(networks)

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
	onchainConfig := cldf_config_env.OnchainConfig{
		EVM: cldf_config_env.EVMConfig{
			DeployerKey: evmKeyHex,
			Seth:        &cldf_config_env.SethConfig{},
		},
		Solana: cldf_config_env.SolanaConfig{
			WalletKey:       solKey.String(),
			ProgramsDirPath: solProgramsPath,
		},
		Aptos: cldf_config_env.AptosConfig{
			DeployerKey: "0xE4FD0E90D32CB98DC6AD64516A421E8C2731870217CDBA64203CEB158A866304",
		},
		Tron: cldf_config_env.TronConfig{
			DeployerKey: evmKeyHex, // Uses the same key as the EVM chain
		},
		Sui: cldf_config_env.SuiConfig{
			DeployerKey: "0xA1B2C3D4E5F60718293A4B5C6D7E8F90123456789ABCDEF0123456789ABCDEF0", // Mock private key
		},
	}

	tests := []struct {
		name              string
		giveNetworkConfig *cldf_config_network.Config
		giveOnchainConfig cldf_config_env.OnchainConfig
		giveSelectors     []uint64
		wantCount         int
		wantErr           string
	}{
		{
			name:              "loads all valid chains",
			giveNetworkConfig: networksConfig,
			giveOnchainConfig: onchainConfig,
			giveSelectors:     []uint64{evmSelector, solanaSelector, aptosSelector, tronSelector, suiSelector},
			wantCount:         5,
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
			giveSelectors:     []uint64{evmSelector, chain_selectors.TEST_90000001.Selector},
			wantErr:           "failed to load 1 out of 2 chains",
		},
		{
			name: "fails to load a chain",
			giveNetworkConfig: cldf_config_network.NewConfig([]cldf_config_network.Network{
				{
					Type:          cldf_config_network.NetworkTypeTestnet,
					ChainSelector: evmSelector,
					RPCs: []cldf_config_network.RPC{
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

			config := &cldf_environment.Config{
				Networks: tt.giveNetworkConfig,
				Env: &cldf_config_env.Config{
					Onchain: tt.giveOnchainConfig,
				},
			}

			chains, err := LoadChains(ctx, lggr, config, tt.giveSelectors)

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
	var aptosSelector = chain_selectors.APTOS_LOCALNET.Selector

	// Create test network config
	networkCfg := cldf_config_network.NewConfig([]cldf_config_network.Network{
		{
			Type:          cldf_config_network.NetworkTypeMainnet,
			ChainSelector: aptosSelector,
			RPCs: []cldf_config_network.RPC{
				{
					RPCName:            "aptos_localnet_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "https://fullnode.localnet.aptoslabs.com/v1",
					WSURL:              "", // This is not used for Aptos
				},
			},
		},
		{
			Type:          cldf_config_network.NetworkTypeTestnet,
			ChainSelector: 999999, // Different chain for testing
			RPCs: []cldf_config_network.RPC{
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
	onchainConfig := cldf_config_env.OnchainConfig{
		Aptos: cldf_config_env.AptosConfig{
			DeployerKey: "0xE4FD0E90D32CB98DC6AD64516A421E8C2731870217CDBA64203CEB158A866304", // Mock private key
		},
	}

	tests := []struct {
		name              string
		giveSelector      uint64
		giveNetworkConfig *cldf_config_network.Config
		giveOnchainConfig cldf_config_env.OnchainConfig
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
			giveNetworkConfig: cldf_config_network.NewConfig([]cldf_config_network.Network{
				{
					Type:          cldf_config_network.NetworkTypeTestnet,
					ChainSelector: 777777,
					RPCs:          []cldf_config_network.RPC{}, // Empty RPCs slice
				},
			}),
			giveOnchainConfig: onchainConfig,
			wantErr:           "no RPCs found for chain selector: 777777",
		},
		{
			name:              "empty private key",
			giveSelector:      aptosSelector,
			giveNetworkConfig: networkCfg,
			giveOnchainConfig: cldf_config_env.OnchainConfig{
				Aptos: cldf_config_env.AptosConfig{
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

	var suiSelector = chain_selectors.SUI_LOCALNET.Selector

	networkCfg := cldf_config_network.NewConfig([]cldf_config_network.Network{
		{
			Type:          cldf_config_network.NetworkTypeMainnet,
			ChainSelector: suiSelector,
			RPCs: []cldf_config_network.RPC{
				{
					RPCName:            "sui_localnet_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "https://fullnode.localnet.suilabs.com/v1",
					WSURL:              "", // This is not used for Sui
				},
			},
		},
		{
			Type:          cldf_config_network.NetworkTypeTestnet,
			ChainSelector: 999999, // Different chain for testing
			RPCs: []cldf_config_network.RPC{
				{
					RPCName:            "other_chain_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "https://other.rpc.com",
					WSURL:              "",
				},
			},
		},
	})

	onchainConfig := cldf_config_env.OnchainConfig{
		Sui: cldf_config_env.SuiConfig{
			DeployerKey: "0xA1B2C3D4E5F60718293A4B5C6D7E8F90123456789ABCDEF0123456789ABCDEF0", // Mock private key
		},
	}

	tests := []struct {
		name              string
		giveSelector      uint64
		giveNetworkConfig *cldf_config_network.Config
		giveOnchainConfig cldf_config_env.OnchainConfig
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
			giveNetworkConfig: cldf_config_network.NewConfig([]cldf_config_network.Network{
				{
					Type:          cldf_config_network.NetworkTypeTestnet,
					ChainSelector: 777777,
					RPCs:          []cldf_config_network.RPC{},
				},
			}),
			giveOnchainConfig: onchainConfig,
			wantErr:           "no RPCs found for chain selector: 777777",
		},
		{
			name:              "empty private key",
			giveSelector:      suiSelector,
			giveNetworkConfig: networkCfg,
			giveOnchainConfig: cldf_config_env.OnchainConfig{
				Sui: cldf_config_env.SuiConfig{
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
	var solanaSelector = chain_selectors.TEST_22222222222222222222222222222222222222222222.Selector

	// Create test network config
	networkCfg := cldf_config_network.NewConfig([]cldf_config_network.Network{
		{
			Type:          cldf_config_network.NetworkTypeTestnet,
			ChainSelector: solanaSelector,
			RPCs: []cldf_config_network.RPC{
				{
					RPCName:            "solana_localnet_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "http://localhost:8899",
					WSURL:              "ws://localhost:8900",
				},
			},
		},
		{
			Type:          cldf_config_network.NetworkTypeTestnet,
			ChainSelector: 123456, // Different chain for testing
			RPCs: []cldf_config_network.RPC{
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

	onchainConfig := cldf_config_env.OnchainConfig{
		Solana: cldf_config_env.SolanaConfig{
			WalletKey:       privKey.String(), // Mock private key
			ProgramsDirPath: programsPath,     // Mock path
		},
	}

	tests := []struct {
		name              string
		giveSelector      uint64
		giveNetworkConfig *cldf_config_network.Config
		giveOnchainConfig cldf_config_env.OnchainConfig
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
			giveNetworkConfig: cldf_config_network.NewConfig([]cldf_config_network.Network{
				{
					Type:          cldf_config_network.NetworkTypeTestnet,
					ChainSelector: 888888,
					RPCs:          []cldf_config_network.RPC{}, // Empty RPCs slice
				},
			}),
			giveOnchainConfig: onchainConfig,
			wantErr:           "no RPCs found for chain selector: 888888",
		},
		{
			name:              "empty private key",
			giveSelector:      solanaSelector,
			giveNetworkConfig: networkCfg,
			giveOnchainConfig: cldf_config_env.OnchainConfig{
				Solana: cldf_config_env.SolanaConfig{
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
			giveOnchainConfig: cldf_config_env.OnchainConfig{
				Solana: cldf_config_env.SolanaConfig{
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
		ctx            = t.Context()
		fakeSrv        = newFakeRPCServer(t)
		evmSelector    = chain_selectors.ETHEREUM_TESTNET_SEPOLIA.Selector
		zksyncSelector = chain_selectors.ETHEREUM_TESTNET_SEPOLIA_ZKSYNC_1.Selector
	)

	// Create test network config
	networkCfg := cldf_config_network.NewConfig([]cldf_config_network.Network{
		{
			Type:          cldf_config_network.NetworkTypeTestnet,
			ChainSelector: evmSelector,
			RPCs: []cldf_config_network.RPC{
				{
					RPCName:            "sepolia_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            fakeSrv.URL,
				},
			},
		},
		{
			Type:          cldf_config_network.NetworkTypeTestnet,
			ChainSelector: zksyncSelector,
			RPCs: []cldf_config_network.RPC{
				{
					RPCName:            "zksync_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            fakeSrv.URL,
				},
			},
		},
		{
			Type:          cldf_config_network.NetworkTypeTestnet,
			ChainSelector: 999999, // Different chain for testing
			RPCs: []cldf_config_network.RPC{
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
	onchainConfig := cldf_config_env.OnchainConfig{
		EVM: cldf_config_env.EVMConfig{
			DeployerKey: privKeyHex,
		},
	}

	tests := []struct {
		name              string
		giveSelector      uint64
		giveConfig        *cldf_config_network.Config
		giveOnchainConfig cldf_config_env.OnchainConfig
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
			giveSelector:      zksyncSelector,
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
			giveConfig: cldf_config_network.NewConfig([]cldf_config_network.Network{
				{
					Type:          cldf_config_network.NetworkTypeTestnet,
					ChainSelector: 777777,
					RPCs:          []cldf_config_network.RPC{}, // Empty RPCs slice
				},
			}),
			giveOnchainConfig: onchainConfig,
			wantErr:           "no RPCs found for chain selector: 777777",
		},
		{
			name:         "fail to initialize evm chain",
			giveSelector: evmSelector,
			giveConfig:   networkCfg,
			giveOnchainConfig: cldf_config_env.OnchainConfig{
				EVM: cldf_config_env.EVMConfig{
					DeployerKey: privKeyHex,
					Seth: &cldf_config_env.SethConfig{
						GethWrapperDirs: []string{"some/valid/gethwrapper"},
						ConfigFilePath:  "/invalid", // Must be empty or a valid path
					},
				},
			},
			wantErr: fmt.Sprintf("failed to initialize chain %d", evmSelector),
		},
		{
			name:         "fail to initialize zksync chain",
			giveSelector: zksyncSelector,
			giveConfig:   networkCfg,
			giveOnchainConfig: cldf_config_env.OnchainConfig{
				EVM: cldf_config_env.EVMConfig{
					DeployerKey: privKeyHex,
					Seth: &cldf_config_env.SethConfig{
						GethWrapperDirs: []string{"some/valid/gethwrapper"},
						ConfigFilePath:  "/invalid", // Must be empty or a valid path
					},
				},
			},
			wantErr: fmt.Sprintf("failed to initialize chain %d", zksyncSelector),
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

	var tronSelector = chain_selectors.TRON_TESTNET_NILE.Selector

	networks := cldf_config_network.NewConfig([]cldf_config_network.Network{
		{
			Type:          cldf_config_network.NetworkTypeTestnet,
			ChainSelector: tronSelector,
			RPCs: []cldf_config_network.RPC{
				{
					RPCName:            "tron_testnet_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "https://nile.trongrid.io",
				},
			},
		},
		{
			Type:          cldf_config_network.NetworkTypeTestnet,
			ChainSelector: 999999, // Different chain for testing
			RPCs: []cldf_config_network.RPC{
				{
					RPCName:            "other_chain_rpc",
					PreferredURLScheme: "http",
					HTTPURL:            "https://other.rpc.com",
					WSURL:              "",
				},
			},
		},
	})

	onchainConfig := cldf_config_env.OnchainConfig{
		Tron: cldf_config_env.TronConfig{
			DeployerKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", // Mock private key
		},
	}

	tests := []struct {
		name              string
		giveSelector      uint64
		giveNetworkConfig *cldf_config_network.Config
		giveOnchainConfig cldf_config_env.OnchainConfig
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
			giveNetworkConfig: cldf_config_network.NewConfig([]cldf_config_network.Network{
				{
					Type:          cldf_config_network.NetworkTypeTestnet,
					ChainSelector: 777777,
					RPCs:          []cldf_config_network.RPC{}, // Empty RPCs slice
				},
			}),
			giveOnchainConfig: onchainConfig,
			wantErr:           "no RPCs found for chain selector: 777777",
		},
		{
			name:              "empty private key",
			giveSelector:      tronSelector,
			giveNetworkConfig: networks,
			giveOnchainConfig: cldf_config_env.OnchainConfig{
				Tron: cldf_config_env.TronConfig{
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
// response.
//
// When the test is done, the server is closed automatically.
func newFakeRPCServer(t *testing.T) *httptest.Server {
	t.Helper()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return a valid eth_blockNumber response
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`))
	})

	srv := httptest.NewServer(handler)

	t.Cleanup(func() {
		srv.Close()
	})

	return srv
}

package provider

import (
	"context"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/smartcontractkit/freeport"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider/rpcclient"
)

func TestCTFGethChainProvider_Initialize(t *testing.T) {
	t.Parallel()

	var once sync.Once
	config := CTFGethChainProviderConfig{
		Once:           &once,
		ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
		T:              t,
	}

	// Use a test chain selector (e.g., for local development)
	// Chain ID 31337 (0x7A69) is commonly used for local Geth instances
	selector := uint64(13264668187771770619) // This corresponds to chain ID 31337

	provider := NewCTFGethChainProvider(selector, config)

	// Test initial state
	require.Equal(t, selector, provider.ChainSelector())
	require.Equal(t, "Geth EVM CTF Chain Provider", provider.Name())

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()

	// Initialize the provider
	blockchain, err := provider.Initialize(ctx)
	require.NoError(t, err)
	require.NotNil(t, blockchain)

	// Test that subsequent calls return the same chain
	blockchain2, err := provider.Initialize(ctx)
	require.NoError(t, err)
	require.Equal(t, blockchain.ChainSelector(), blockchain2.ChainSelector())
	require.Equal(t, blockchain.Name(), blockchain2.Name())
	require.Equal(t, blockchain.Family(), blockchain2.Family())
	require.Equal(t, blockchain.String(), blockchain2.String())

	// Test that BlockChain() returns the initialized chain
	chainFromProvider := provider.BlockChain()
	require.Equal(t, blockchain.ChainSelector(), chainFromProvider.ChainSelector())
	require.Equal(t, blockchain.Name(), chainFromProvider.Name())
	require.Equal(t, blockchain.Family(), chainFromProvider.Family())
	require.Equal(t, blockchain.String(), chainFromProvider.String())

	// Test that GetNodeHTTPURL returns a valid URL after initialization
	httpURL := provider.GetNodeHTTPURL()
	require.NotEmpty(t, httpURL)
	require.Contains(t, httpURL, "http://")
}

func TestCTFGethChainProvider_ValidateConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		config        CTFGethChainProviderConfig
		expectedError string
	}{
		{
			name: "valid config",
			config: CTFGethChainProviderConfig{
				Once:           &sync.Once{},
				ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
				T:              t, // Required when Port is not provided
			},
		},
		{
			name: "valid config with additional accounts",
			config: CTFGethChainProviderConfig{
				Once:               &sync.Once{},
				ConfirmFunctor:     ConfirmFuncGeth(2 * time.Minute),
				AdditionalAccounts: true,
				T:                  t, // Required when Port is not provided
			},
		},
		{
			name: "valid config with explicit port",
			config: CTFGethChainProviderConfig{
				Once:           &sync.Once{},
				ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
				Port:           "8545", // When port is provided, T is not required
			},
		},
		{
			name: "missing Once",
			config: CTFGethChainProviderConfig{
				Once:           nil,
				ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
				T:              t,
			},
			expectedError: "sync.Once instance is required",
		},
		{
			name: "missing ConfirmFunctor",
			config: CTFGethChainProviderConfig{
				Once:           &sync.Once{},
				ConfirmFunctor: nil,
				T:              t,
			},
			expectedError: "confirm functor is required",
		},
		{
			name: "missing T field when port not provided",
			config: CTFGethChainProviderConfig{
				Once:           &sync.Once{},
				ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
				T:              nil, // Missing T when Port is not provided
			},
			expectedError: "field T is required when port is not provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.validate()
			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCTFGethChainProvider_NewProvider(t *testing.T) {
	t.Parallel()

	var once sync.Once
	config := CTFGethChainProviderConfig{
		Once:           &once,
		ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
		T:              t,
	}
	selector := uint64(13264668187771770619)

	provider := NewCTFGethChainProvider(selector, config)

	require.NotNil(t, provider)
	require.Equal(t, selector, provider.selector)
	require.Equal(t, config, provider.config)
	require.Nil(t, provider.chain) // Should be nil until initialized

	// Test that GetNodeHTTPURL returns empty string before initialization
	httpURL := provider.GetNodeHTTPURL()
	require.Empty(t, httpURL)
}

func TestCTFGethChainProvider_CustomConfigurations(t *testing.T) {
	t.Parallel()

	// Custom private key for testing (this is just an example key, not used in real scenarios)
	customKey := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

	var once sync.Once
	config := CTFGethChainProviderConfig{
		Once:                  &once,
		ConfirmFunctor:        ConfirmFuncGeth(2 * time.Minute),
		DeployerTransactorGen: TransactorFromRaw(customKey),
		DockerCmdParamsOverrides: []string{
			"--miner.gasprice", "1000000000", // Set min gas price to 1 gwei
			"--miner.threads", "1", // Use 1 mining thread
		},
		Port:  "8545",                      // Custom port
		Image: "ethereum/client-go:stable", // Custom image
		T:     t,                           // Required for validation
	}

	// Test that config validation passes with all custom configurations
	err := config.validate()
	require.NoError(t, err)

	// Verify all custom configurations are stored correctly
	require.Equal(t, TransactorFromRaw(customKey), config.DeployerTransactorGen)
	require.Equal(t, []string{
		"--miner.gasprice", "1000000000",
		"--miner.threads", "1",
	}, config.DockerCmdParamsOverrides)
	require.Equal(t, "8545", config.Port)
	require.Equal(t, "ethereum/client-go:stable", config.Image)
}

func TestCTFGethChainProviderConfig_InvalidPort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		port          string
		expectedError string
	}{
		{
			name:          "invalid port - not a number",
			port:          "invalid",
			expectedError: "invalid port invalid: must be a valid integer",
		},
		{
			name:          "invalid port - zero",
			port:          "0",
			expectedError: "invalid port 0: must be between 1 and 65535",
		},
		{
			name:          "invalid port - negative",
			port:          "-1",
			expectedError: "invalid port -1: must be between 1 and 65535",
		},
		{
			name:          "invalid port - too large",
			port:          "65536",
			expectedError: "invalid port 65536: must be between 1 and 65535",
		},
		{
			name: "valid port",
			port: "8545",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var once sync.Once
			config := CTFGethChainProviderConfig{
				Once:           &once,
				ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
				Port:           tt.port,
				T:              t, // Required when port is not provided
			}

			err := config.validate()
			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCTFGethChainProvider_InitializeErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		selector      uint64
		config        CTFGethChainProviderConfig
		expectedError string
	}{
		{
			name:     "invalid chain selector",
			selector: uint64(999999999999999999), // Invalid chain selector that doesn't map to any chain ID
			config: CTFGethChainProviderConfig{
				Once:           &sync.Once{},
				ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
				T:              t,
			},
			expectedError: "chain selector", // Error should contain chain selector related message
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := NewCTFGethChainProvider(tt.selector, tt.config)

			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()

			_, err := provider.Initialize(ctx)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestCTFGethChainProvider_SignerIntegration(t *testing.T) {
	t.Parallel()

	t.Run("custom raw key integration", func(t *testing.T) {
		t.Parallel()

		// Custom private key for testing (this is a valid test key, not for production)
		customKey := "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

		var once sync.Once
		config := CTFGethChainProviderConfig{
			Once:                  &once,
			ConfirmFunctor:        ConfirmFuncGeth(2 * time.Minute),
			DeployerTransactorGen: TransactorFromRaw(customKey),
			T:                     t,
		}

		selector := uint64(13264668187771770619) // Chain ID 31337
		provider := NewCTFGethChainProvider(selector, config)

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
		defer cancel()

		blockchain, err := provider.Initialize(ctx)
		require.NoError(t, err)
		require.NotNil(t, blockchain)

		// Verify the blockchain has the expected properties
		require.Equal(t, selector, blockchain.ChainSelector())
		require.Equal(t, "evm", blockchain.Family())

		// Test that the deployer key is properly configured
		evmChain := blockchain.(evm.Chain)
		require.NotNil(t, evmChain.DeployerKey)
		require.NotNil(t, evmChain.SignHash)

		// Test signing functionality
		testHash := make([]byte, 32) // Hash must be exactly 32 bytes
		copy(testHash, []byte("test message to sign"))
		signature, err := evmChain.SignHash(testHash)
		require.NoError(t, err)
		require.NotEmpty(t, signature)
		require.Len(t, signature, 65) // Standard Ethereum signature length
	})

	t.Run("default geth key integration", func(t *testing.T) {
		t.Parallel()

		var once sync.Once
		config := CTFGethChainProviderConfig{
			Once:           &once,
			ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
			// No DeployerTransactorGen - should use default Geth account
			T: t,
		}

		selector := uint64(13264668187771770619) // Chain ID 31337
		provider := NewCTFGethChainProvider(selector, config)

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
		defer cancel()

		blockchain, err := provider.Initialize(ctx)
		require.NoError(t, err)
		require.NotNil(t, blockchain)

		evmChain := blockchain.(evm.Chain)
		require.NotNil(t, evmChain.DeployerKey)
		require.NotNil(t, evmChain.SignHash)

		// Verify the default deployer address matches Geth's first test account
		expectedAddress := "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
		require.Equal(t, expectedAddress, evmChain.DeployerKey.From.Hex())

		// Test signing functionality with default key
		testHash := make([]byte, 32) // Hash must be exactly 32 bytes
		copy(testHash, []byte("test message for default key"))
		signature, err := evmChain.SignHash(testHash)
		require.NoError(t, err)
		require.NotEmpty(t, signature)
		require.Len(t, signature, 65) // Standard Ethereum signature length
	})

	t.Run("verify additional users are autofunded when created", func(t *testing.T) {
		t.Parallel()
		var once sync.Once
		cfg := CTFGethChainProviderConfig{
			Once:               &once,
			ConfirmFunctor:     ConfirmFuncGeth(2 * time.Minute),
			AdditionalAccounts: true,
			T:                  t,
		}
		sel := uint64(13264668187771770619)
		p := NewCTFGethChainProvider(sel, cfg)

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
		defer cancel()

		bc, err := p.Initialize(ctx)
		require.NoError(t, err)

		evmChain := bc.(evm.Chain)
		require.Len(t, evmChain.Users, 4) // accounts 1..4

		expected := new(big.Int).Mul(big.NewInt(100), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)) // 100 ETH
		for _, u := range evmChain.Users {
			bal, err := evmChain.Client.BalanceAt(ctx, u.From, nil)
			require.NoError(t, err)
			cmp := bal.Cmp(expected)
			require.Equal(t, 0, cmp, "balance should be = 100 ETH")
		}
	})

	t.Run("user transactors generation", func(t *testing.T) {
		t.Parallel()

		var once sync.Once
		config := CTFGethChainProviderConfig{
			Once:               &once,
			ConfirmFunctor:     ConfirmFuncGeth(2 * time.Minute),
			AdditionalAccounts: true,
			T:                  t,
		}

		selector := uint64(13264668187771770619) // Chain ID 31337
		provider := NewCTFGethChainProvider(selector, config)

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
		defer cancel()

		blockchain, err := provider.Initialize(ctx)
		require.NoError(t, err)
		require.NotNil(t, blockchain)

		evmChain := blockchain.(evm.Chain)
		require.NotNil(t, evmChain.Users)
		require.Len(t, evmChain.Users, 4) // accounts 1..4

		// Verify each user transactor is properly configured
		expectedAddresses := []string{
			"0x70997970C51812dc3A010C7d01b50e0d17dc79C8", // Account 1
			"0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC", // Account 2
			"0x90F79bf6EB2c4f870365E785982E1f101E93b906", // Account 3
			"0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65", // Account 4
		}

		for i, user := range evmChain.Users {
			require.NotNil(t, user)
			require.Equal(t, expectedAddresses[i], user.From.Hex())
			require.NotNil(t, user.Signer)
		}
	})

	t.Run("maximum user accounts", func(t *testing.T) {
		t.Parallel()

		var once sync.Once
		config := CTFGethChainProviderConfig{
			Once:               &once,
			ConfirmFunctor:     ConfirmFuncGeth(2 * time.Minute),
			AdditionalAccounts: true,
			T:                  t,
		}

		selector := uint64(13264668187771770619) // Chain ID 31337
		provider := NewCTFGethChainProvider(selector, config)

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
		defer cancel()

		blockchain, err := provider.Initialize(ctx)
		require.NoError(t, err)
		require.NotNil(t, blockchain)

		evmChain := blockchain.(evm.Chain)
		require.NotNil(t, evmChain.Users)
		// Should have 4 user accounts (total 5 geth keys - 1 deployer = 4 users)
		require.Len(t, evmChain.Users, 4)
	})

	t.Run("client options integration", func(t *testing.T) {
		t.Parallel()

		var once sync.Once
		var clientOptsCalled bool

		config := CTFGethChainProviderConfig{
			Once:           &once,
			ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
			ClientOpts: []func(client *rpcclient.MultiClient){
				func(client *rpcclient.MultiClient) {
					clientOptsCalled = true
					// This is a test to verify the option is called
					// In real scenarios, you might configure timeouts, etc.
				},
			},
			T: t,
		}

		selector := uint64(13264668187771770619) // Chain ID 31337
		provider := NewCTFGethChainProvider(selector, config)

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
		defer cancel()

		blockchain, err := provider.Initialize(ctx)
		require.NoError(t, err)
		require.NotNil(t, blockchain)

		// Verify that client options were applied
		require.True(t, clientOptsCalled, "ClientOpts should have been called during initialization")
	})
}

func TestCTFGethChainProvider_Cleanup(t *testing.T) {
	t.Parallel()

	t.Run("cleanup after initialization with T provided", func(t *testing.T) {
		t.Parallel()

		var once sync.Once
		config := CTFGethChainProviderConfig{
			Once:           &once,
			ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
			T:              t,
		}

		selector := uint64(13264668187771770619) // Chain ID 31337
		provider := NewCTFGethChainProvider(selector, config)

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
		defer cancel()

		blockchain, err := provider.Initialize(ctx)
		require.NoError(t, err)
		require.NotNil(t, blockchain)

		require.NotNil(t, provider.container, "Container reference should be stored after initialization")

		err = provider.Cleanup(ctx)
		require.NoError(t, err, "Cleanup should succeed")

		require.Nil(t, provider.container, "Container reference should be cleared after cleanup")
	})

	t.Run("cleanup after initialization with fixed port (T=nil)", func(t *testing.T) {
		t.Parallel()

		// Allocate a free port for this test
		port := freeport.GetOne(t)
		t.Cleanup(func() {
			freeport.Return([]int{port})
		})

		var once sync.Once
		config := CTFGethChainProviderConfig{
			Once:           &once,
			ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
			Port:           strconv.Itoa(port), // Use allocated port to avoid conflicts
			T:              nil,                // No T provided - simulates production usage
		}

		selector := uint64(13264668187771770619) // Chain ID 31337
		provider := NewCTFGethChainProvider(selector, config)

		t.Cleanup(func() {
			// Ensure cleanup is called at the end of the test to avoid leaking containers
			// even if the test fails
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			_ = provider.Cleanup(ctx)
		})

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
		defer cancel()

		blockchain, err := provider.Initialize(ctx)
		require.NoError(t, err)
		require.NotNil(t, blockchain)

		require.NotNil(t, provider.container, "Container reference should be stored after initialization")

		err = provider.Cleanup(ctx)
		require.NoError(t, err, "Cleanup should succeed even when T is nil")

		require.Nil(t, provider.container, "Container reference should be cleared after cleanup")
	})

	t.Run("cleanup before initialization", func(t *testing.T) {
		t.Parallel()

		var once sync.Once
		config := CTFGethChainProviderConfig{
			Once:           &once,
			ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
			T:              t,
		}

		selector := uint64(13264668187771770619) // Chain ID 31337
		provider := NewCTFGethChainProvider(selector, config)

		ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
		defer cancel()

		// Test cleanup before initialization - should be a no-op
		err := provider.Cleanup(ctx)
		require.NoError(t, err, "Cleanup should succeed even when no container exists")

		// Verify container is still nil
		require.Nil(t, provider.container, "Container reference should remain nil")
	})

	t.Run("multiple cleanup calls", func(t *testing.T) {
		t.Parallel()

		var once sync.Once
		config := CTFGethChainProviderConfig{
			Once:           &once,
			ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
			T:              t,
		}

		selector := uint64(13264668187771770619) // Chain ID 31337
		provider := NewCTFGethChainProvider(selector, config)

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
		defer cancel()

		blockchain, err := provider.Initialize(ctx)
		require.NoError(t, err)
		require.NotNil(t, blockchain)

		// First cleanup
		err = provider.Cleanup(ctx)
		require.NoError(t, err, "First cleanup should succeed")
		require.Nil(t, provider.container, "Container reference should be cleared after first cleanup")

		// Second cleanup - should be a no-op
		err = provider.Cleanup(ctx)
		require.NoError(t, err, "Second cleanup should succeed (no-op)")
		require.Nil(t, provider.container, "Container reference should remain nil after second cleanup")

		// Third cleanup - should still be a no-op
		err = provider.Cleanup(ctx)
		require.NoError(t, err, "Third cleanup should succeed (no-op)")
		require.Nil(t, provider.container, "Container reference should remain nil after third cleanup")
	})

	t.Run("cleanup with cancelled context", func(t *testing.T) {
		t.Parallel()

		var once sync.Once
		config := CTFGethChainProviderConfig{
			Once:           &once,
			ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
			T:              t,
		}

		selector := uint64(13264668187771770619) // Chain ID 31337
		provider := NewCTFGethChainProvider(selector, config)

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
		defer cancel()

		blockchain, err := provider.Initialize(ctx)
		require.NoError(t, err)
		require.NotNil(t, blockchain)

		cancelledCtx, cancelFunc := context.WithCancel(t.Context())
		cancelFunc() // Cancel immediately

		// Test cleanup with cancelled context - should return an error
		err = provider.Cleanup(cancelledCtx)
		require.Error(t, err, "Cleanup should fail with cancelled context")
		require.Contains(t, err.Error(), "failed to terminate Geth container", "Error should mention container termination failure")

		// Container reference should still exist since cleanup failed
		require.NotNil(t, provider.container, "Container reference should remain when cleanup fails")

		// Cleanup with proper context should still work
		err = provider.Cleanup(ctx)
		require.NoError(t, err, "Cleanup with valid context should succeed after previous failure")
		require.Nil(t, provider.container, "Container reference should be cleared after successful cleanup")
	})
}

func TestCTFGethChainProvider_WebSocket(t *testing.T) {
	t.Parallel()

	var once sync.Once
	config := CTFGethChainProviderConfig{
		Once:           &once,
		ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
		T:              t,
	}

	selector := uint64(13264668187771770619) // maps to chain ID 31337
	provider := NewCTFGethChainProvider(selector, config)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()

	// initialize the node
	_, err := provider.Initialize(ctx)
	require.NoError(t, err)

	httpURL := provider.GetNodeHTTPURL()
	require.NotEmpty(t, httpURL)

	// CTF exposes WS on the same host:port as HTTP; just swap the scheme
	wsURL := strings.Replace(httpURL, "http://", "ws://", 1)
	require.NotEqual(t, httpURL, wsURL)

	// Dial WS and make a simple call
	wsCtx, wsCancel := context.WithTimeout(ctx, 30*time.Second)
	defer wsCancel()

	// go-ethereum's low-level RPC client
	rpcClient, err := rpc.DialWebsocket(wsCtx, wsURL, "")
	require.NoError(t, err, "should connect to WS endpoint")

	var chainIDHex string
	err = rpcClient.CallContext(wsCtx, &chainIDHex, "eth_chainId")
	require.NoError(t, err, "eth_chainId over WS should succeed")
	require.NotEmpty(t, chainIDHex)

	// Optional: subscribe to new heads to verify real WS subscriptions
	headers := make(chan *types.Header, 1)
	sub, err := rpcClient.EthSubscribe(wsCtx, headers, "newHeads")
	require.NoError(t, err, "newHeads subscription should succeed")

	// Wait briefly for at least one header (Geth is mining)
	select {
	case <-headers:
		// ok
	case <-time.After(20 * time.Second):
		t.Fatal("timed out waiting for newHeads over WS")
	}

	sub.Unsubscribe()
	rpcClient.Close()
}

func TestCTFGeth_NoAdditionalAccounts_NoFunding(t *testing.T) {
	t.Parallel()
	var once sync.Once
	cfg := CTFGethChainProviderConfig{
		Once:           &once,
		ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
		// AdditionalAccounts defaults to false
		T: t,
	}
	sel := uint64(13264668187771770619)
	p := NewCTFGethChainProvider(sel, cfg)

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	bc, err := p.Initialize(ctx)
	require.NoError(t, err)
	evmChain := bc.(evm.Chain)

	// No users should be created
	require.Empty(t, evmChain.Users)

	// Deployer should not have sent any txs (funding), so nonce == 0
	nonce, err := evmChain.Client.PendingNonceAt(ctx, evmChain.DeployerKey.From)
	require.NoError(t, err)
	require.Equal(t, uint64(0), nonce)
}

func TestCTFGeth_Initialize_IdempotentFunding(t *testing.T) {
	t.Parallel()
	var once sync.Once
	cfg := CTFGethChainProviderConfig{
		Once:               &once,
		ConfirmFunctor:     ConfirmFuncGeth(2 * time.Minute),
		AdditionalAccounts: true,
		T:                  t,
	}
	sel := uint64(13264668187771770619)
	p := NewCTFGethChainProvider(sel, cfg)

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()

	bc, err := p.Initialize(ctx)
	require.NoError(t, err)
	evmChain := bc.(evm.Chain)
	require.Len(t, evmChain.Users, 4)

	// Snapshot balances after first init
	initBals := make([]*big.Int, len(evmChain.Users))
	for i, u := range evmChain.Users {
		b, errB := evmChain.Client.BalanceAt(ctx, u.From, nil)
		require.NoError(t, errB)
		initBals[i] = new(big.Int).Set(b)
	}

	// Call Initialize again; should be a no-op
	bc2, err := p.Initialize(ctx)
	require.NoError(t, err)
	_ = bc2

	// Balances should be unchanged (no re-funding)
	for i, u := range evmChain.Users {
		b, err := evmChain.Client.BalanceAt(ctx, u.From, nil)
		require.NoError(t, err)
		require.Equal(t, 0, b.Cmp(initBals[i]), "user %d balance changed on re-init", i)
	}
}

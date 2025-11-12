package provider

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/smartcontractkit/freeport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider/rpcclient"
)

func TestCTFAnvilChainProvider_Initialize(t *testing.T) {
	t.Parallel()

	var once sync.Once
	config := CTFAnvilChainProviderConfig{
		Once:           &once,
		ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
		T:              t,
	}

	// Use a test chain selector (e.g., for local development)
	// Chain ID 31337 (0x7A69) is commonly used for local Anvil instances
	selector := uint64(13264668187771770619) // This corresponds to chain ID 31337

	provider := NewCTFAnvilChainProvider(selector, config)

	// Test initial state
	assert.Equal(t, selector, provider.ChainSelector())
	assert.Equal(t, "Anvil EVM CTF Chain Provider", provider.Name())

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()

	// Initialize the provider
	blockchain, err := provider.Initialize(ctx)
	require.NoError(t, err)
	require.NotNil(t, blockchain)

	// Test that subsequent calls return the same chain
	blockchain2, err := provider.Initialize(ctx)
	require.NoError(t, err)
	assert.Equal(t, blockchain.ChainSelector(), blockchain2.ChainSelector())
	assert.Equal(t, blockchain.Name(), blockchain2.Name())
	assert.Equal(t, blockchain.Family(), blockchain2.Family())
	assert.Equal(t, blockchain.String(), blockchain2.String())

	// Test that BlockChain() returns the initialized chain
	chainFromProvider := provider.BlockChain()
	assert.Equal(t, blockchain.ChainSelector(), chainFromProvider.ChainSelector())
	assert.Equal(t, blockchain.Name(), chainFromProvider.Name())
	assert.Equal(t, blockchain.Family(), chainFromProvider.Family())
	assert.Equal(t, blockchain.String(), chainFromProvider.String())

	// Test that GetNodeHTTPURL returns a valid URL after initialization
	httpURL := provider.GetNodeHTTPURL()
	assert.NotEmpty(t, httpURL)
	assert.Contains(t, httpURL, "http://")
}

func TestCTFAnvilChainProvider_ValidateConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		config        CTFAnvilChainProviderConfig
		expectedError string
	}{
		{
			name: "valid config",
			config: CTFAnvilChainProviderConfig{
				Once:           &sync.Once{},
				ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
				T:              t, // Required when Port is not provided
			},
		},
		{
			name: "valid config with additional accounts",
			config: CTFAnvilChainProviderConfig{
				Once:                  &sync.Once{},
				ConfirmFunctor:        ConfirmFuncGeth(2 * time.Minute),
				NumAdditionalAccounts: 5,
				T:                     t, // Required when Port is not provided
			},
		},
		{
			name: "valid config with explicit port",
			config: CTFAnvilChainProviderConfig{
				Once:           &sync.Once{},
				ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
				Port:           "8545", // When port is provided, T is not required
			},
		},
		{
			name: "missing Once",
			config: CTFAnvilChainProviderConfig{
				Once:           nil,
				ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
				T:              t,
			},
			expectedError: "sync.Once instance is required",
		},
		{
			name: "missing ConfirmFunctor",
			config: CTFAnvilChainProviderConfig{
				Once:           &sync.Once{},
				ConfirmFunctor: nil,
				T:              t,
			},
			expectedError: "confirm functor is required",
		},
		{
			name: "missing T field when port not provided",
			config: CTFAnvilChainProviderConfig{
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
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCTFAnvilChainProvider_NewProvider(t *testing.T) {
	t.Parallel()

	var once sync.Once
	config := CTFAnvilChainProviderConfig{
		Once:           &once,
		ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
		T:              t,
	}
	selector := uint64(13264668187771770619)

	provider := NewCTFAnvilChainProvider(selector, config)

	assert.NotNil(t, provider)
	assert.Equal(t, selector, provider.selector)
	assert.Equal(t, config, provider.config)
	assert.Nil(t, provider.chain) // Should be nil until initialized

	// Test that GetNodeHTTPURL returns empty string before initialization
	httpURL := provider.GetNodeHTTPURL()
	assert.Empty(t, httpURL)
}

func TestCTFAnvilChainProvider_CustomConfigurations(t *testing.T) {
	t.Parallel()

	// Custom private key for testing (this is just an example key, not used in real scenarios)
	customKey := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

	var once sync.Once
	config := CTFAnvilChainProviderConfig{
		Once:                  &once,
		ConfirmFunctor:        ConfirmFuncGeth(2 * time.Minute),
		DeployerTransactorGen: TransactorFromRaw(customKey),
		DockerCmdParamsOverrides: []string{
			"--block-time", "2", // Mine blocks every 2 seconds
			"--gas-limit", "30000000", // Set gas limit to 30M
			"--gas-price", "1000000000", // Set gas price to 1 gwei
		},
		Port:  "8545",                              // Custom port
		Image: "ghcr.io/foundry-rs/foundry:latest", // Custom image
		T:     t,                                   // Required for validation
	}

	// Test that config validation passes with all custom configurations
	err := config.validate()
	require.NoError(t, err)

	// Verify all custom configurations are stored correctly
	assert.Equal(t, TransactorFromRaw(customKey), config.DeployerTransactorGen)
	assert.Equal(t, []string{
		"--block-time", "2",
		"--gas-limit", "30000000",
		"--gas-price", "1000000000",
	}, config.DockerCmdParamsOverrides)
	assert.Equal(t, "8545", config.Port)
	assert.Equal(t, "ghcr.io/foundry-rs/foundry:latest", config.Image)
}

func TestCTFAnvilChainProviderConfig_InvalidPort(t *testing.T) {
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
			config := CTFAnvilChainProviderConfig{
				Once:           &once,
				ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
				Port:           tt.port,
				T:              t, // Required when port is not provided
			}

			err := config.validate()
			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCTFAnvilChainProvider_InitializeErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		selector      uint64
		config        CTFAnvilChainProviderConfig
		expectedError string
	}{
		{
			name:     "invalid chain selector",
			selector: uint64(999999999999999999), // Invalid chain selector that doesn't map to any chain ID
			config: CTFAnvilChainProviderConfig{
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

			provider := NewCTFAnvilChainProvider(tt.selector, tt.config)

			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()

			_, err := provider.Initialize(ctx)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestCTFAnvilChainProvider_SignerIntegration(t *testing.T) {
	t.Parallel()

	t.Run("custom tick interval", func(t *testing.T) {
		t.Parallel()
		customKey := "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

		var once sync.Once
		config := CTFAnvilChainProviderConfig{
			Once:                  &once,
			ConfirmFunctor:        ConfirmFuncGeth(2*time.Minute, WithTickInterval(100*time.Millisecond)),
			DeployerTransactorGen: TransactorFromRaw(customKey),
			T:                     t,
		}

		selector := uint64(13264668187771770619) // Chain ID 31337
		provider := NewCTFAnvilChainProvider(selector, config)

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
		defer cancel()

		blockchain, err := provider.Initialize(ctx)
		require.NoError(t, err)
		require.NotNil(t, blockchain)

		// Verify the blockchain has the expected properties
		assert.Equal(t, selector, blockchain.ChainSelector())
		assert.Equal(t, "evm", blockchain.Family())

		// Test that the deployer key is properly configured
		evmChain := blockchain.(evm.Chain)
		assert.NotNil(t, evmChain.DeployerKey)
		assert.NotNil(t, evmChain.SignHash)

		// Test signing functionality
		testHash := make([]byte, 32) // Hash must be exactly 32 bytes
		copy(testHash, []byte("test message to sign"))
		signature, err := evmChain.SignHash(testHash)
		require.NoError(t, err)
		assert.NotEmpty(t, signature)
		assert.Len(t, signature, 65) // Standard Ethereum signature length
	})

	t.Run("custom raw key integration", func(t *testing.T) {
		t.Parallel()

		// Custom private key for testing (this is a valid test key, not for production)
		customKey := "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

		var once sync.Once
		config := CTFAnvilChainProviderConfig{
			Once:                  &once,
			ConfirmFunctor:        ConfirmFuncGeth(2 * time.Minute),
			DeployerTransactorGen: TransactorFromRaw(customKey),
			T:                     t,
		}

		selector := uint64(13264668187771770619) // Chain ID 31337
		provider := NewCTFAnvilChainProvider(selector, config)

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
		defer cancel()

		blockchain, err := provider.Initialize(ctx)
		require.NoError(t, err)
		require.NotNil(t, blockchain)

		// Verify the blockchain has the expected properties
		assert.Equal(t, selector, blockchain.ChainSelector())
		assert.Equal(t, "evm", blockchain.Family())

		// Test that the deployer key is properly configured
		evmChain := blockchain.(evm.Chain)
		assert.NotNil(t, evmChain.DeployerKey)
		assert.NotNil(t, evmChain.SignHash)

		// Test signing functionality
		testHash := make([]byte, 32) // Hash must be exactly 32 bytes
		copy(testHash, []byte("test message to sign"))
		signature, err := evmChain.SignHash(testHash)
		require.NoError(t, err)
		assert.NotEmpty(t, signature)
		assert.Len(t, signature, 65) // Standard Ethereum signature length
	})

	t.Run("default anvil key integration with 3 additional accounts", func(t *testing.T) {
		t.Parallel()

		var once sync.Once
		config := CTFAnvilChainProviderConfig{
			Once:           &once,
			ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
			// No DeployerTransactorGen - should use default Anvil account
			NumAdditionalAccounts: 3, // Limit to 3 user accounts
			T:                     t,
		}

		selector := uint64(13264668187771770619) // Chain ID 31337
		provider := NewCTFAnvilChainProvider(selector, config)

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
		defer cancel()

		blockchain, err := provider.Initialize(ctx)
		require.NoError(t, err)
		require.NotNil(t, blockchain)

		evmChain := blockchain.(evm.Chain)
		assert.NotNil(t, evmChain.DeployerKey)
		assert.NotNil(t, evmChain.SignHash)
		assert.NotNil(t, evmChain.Users)
		assert.Len(t, evmChain.Users, 3) // Should have exactly 3 user accounts

		// Verify the default deployer address matches Anvil's first test account
		expectedAddress := "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
		assert.Equal(t, expectedAddress, evmChain.DeployerKey.From.Hex())

		// Test signing functionality with default key
		testHash := make([]byte, 32) // Hash must be exactly 32 bytes
		copy(testHash, []byte("test message for default key"))
		signature, err := evmChain.SignHash(testHash)
		require.NoError(t, err)
		assert.NotEmpty(t, signature)
		assert.Len(t, signature, 65) // Standard Ethereum signature length

		// Verify each user transactor is properly configured
		expectedAddresses := []string{
			"0x70997970C51812dc3A010C7d01b50e0d17dc79C8", // Account 1
			"0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC", // Account 2
			"0x90F79bf6EB2c4f870365E785982E1f101E93b906", // Account 3
		}

		for i, user := range evmChain.Users {
			assert.NotNil(t, user)
			assert.Equal(t, expectedAddresses[i], user.From.Hex())
			assert.NotNil(t, user.Signer)
		}
	})

	t.Run("maximum user accounts with client options", func(t *testing.T) {
		t.Parallel()

		var clientOptsCalled bool
		var once sync.Once
		config := CTFAnvilChainProviderConfig{
			Once:           &once,
			ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
			ClientOpts: []func(client *rpcclient.MultiClient){
				func(client *rpcclient.MultiClient) {
					clientOptsCalled = true
					// This is a test to verify the option is called
					// In real scenarios, you might configure timeouts, etc.
				},
			},
			NumAdditionalAccounts: 0, // Should use all available accounts
			T:                     t,
		}

		selector := uint64(13264668187771770619) // Chain ID 31337
		provider := NewCTFAnvilChainProvider(selector, config)

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
		defer cancel()

		blockchain, err := provider.Initialize(ctx)
		require.NoError(t, err)
		require.NotNil(t, blockchain)

		evmChain := blockchain.(evm.Chain)
		assert.NotNil(t, evmChain.Users)
		// Should have 4 user accounts (total 5 anvil keys - 1 deployer = 4 users)
		assert.Len(t, evmChain.Users, 4)

		// Verify that client options were applied
		assert.True(t, clientOptsCalled, "ClientOpts should have been called during initialization")
	})
}

func TestCTFAnvilChainProvider_Cleanup(t *testing.T) {
	t.Parallel()

	t.Run("cleanup after initialization with T provided", func(t *testing.T) {
		t.Parallel()

		var once sync.Once
		config := CTFAnvilChainProviderConfig{
			Once:           &once,
			ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
			T:              t,
		}

		selector := uint64(13264668187771770619) // Chain ID 31337
		provider := NewCTFAnvilChainProvider(selector, config)

		// Test cleanup before initialization - should be a no-op
		err := provider.Cleanup(t.Context())
		require.NoError(t, err, "Cleanup should succeed even when no container exists")

		// Verify container is still nil
		assert.Nil(t, provider.container, "Container reference should remain nil")

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
		defer cancel()

		blockchain, err := provider.Initialize(ctx)
		require.NoError(t, err)
		require.NotNil(t, blockchain)

		assert.NotNil(t, provider.container, "Container reference should be stored after initialization")

		err = provider.Cleanup(ctx)
		require.NoError(t, err, "Cleanup should succeed")

		assert.Nil(t, provider.container, "Container reference should be cleared after cleanup")
	})

	t.Run("cleanup after initialization with fixed port (T=nil)", func(t *testing.T) {
		t.Parallel()

		// Allocate a free port for this test
		port := freeport.GetOne(t)
		t.Cleanup(func() {
			freeport.Return([]int{port})
		})

		var once sync.Once
		config := CTFAnvilChainProviderConfig{
			Once:           &once,
			ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
			Port:           strconv.Itoa(port), // Use allocated port to avoid conflicts
			T:              nil,                // No T provided - simulates production usage
		}

		selector := uint64(13264668187771770619) // Chain ID 31337
		provider := NewCTFAnvilChainProvider(selector, config)

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

		assert.NotNil(t, provider.container, "Container reference should be stored after initialization")

		err = provider.Cleanup(ctx)
		require.NoError(t, err, "Cleanup should succeed even when T is nil")

		assert.Nil(t, provider.container, "Container reference should be cleared after cleanup")

		// Second cleanup - should be a no-op
		err = provider.Cleanup(ctx)
		require.NoError(t, err, "Second cleanup should succeed (no-op)")
		assert.Nil(t, provider.container, "Container reference should remain nil after second cleanup")
	})
}

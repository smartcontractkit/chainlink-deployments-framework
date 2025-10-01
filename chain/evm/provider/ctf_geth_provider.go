// Package provider contains EVM chain providers for the Chainlink Deployments Framework.
//
// This file implements CTFGethChainProvider, which provides Geth EVM chain instances
// running inside Chainlink Testing Framework (CTF) Docker container.
//
// # Geth Integration
//
// Geth is a local Ethereum node.
//
// # Usage Patterns
//
// Basic usage for simple testing:
//
//	func TestBasicContract(t *testing.T) {
//		var once sync.Once
//		config := CTFGethChainProviderConfig{
//			Once:           &once,
//			ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
//			T:              t, // Required when Port is not provided
//		}
//
//		provider := NewCTFGethChainProvider(chainSelector, config)
//		blockchain, err := provider.Initialize(context.Background())
//		require.NoError(t, err)
//
//		// Deploy and test contracts...
//	}
//
// Advanced usage with custom configuration:
//
//		func TestAdvancedScenario(t *testing.T) {
//			var once sync.Once
//	 	config := CTFGethChainProviderConfig{
//	     Once:               &once,
//	     ConfirmFunctor:     ConfirmFuncGeth(30 * time.Second),
//	     AdditionalAccounts: true, // Create & auto-fund all extra users from the built-in pool
//	     T:                  t,    // Required when Port is not provided
//	 }
//
//			provider := NewCTFGethChainProvider(chainSelector, config)
//			blockchain, err := provider.Initialize(context.Background())
//			require.NoError(t, err)
//
//			// Run complex multi-account tests...
//		}
//
// Usage with custom deployer key (using TransactorFromRaw):
//
//	func TestWithCustomDeployer(t *testing.T) {
//		var once sync.Once
//		config := CTFGethChainProviderConfig{
//			Once:                  &once,
//			ConfirmFunctor:        ConfirmFuncGeth(2 * time.Minute),
//			DeployerTransactorGen: TransactorFromRaw("your-custom-private-key-here"), // 64 chars hex, no 0x prefix
//			T:                     t, // Required when Port is not provided
//		}
//
//		provider := NewCTFGethChainProvider(chainSelector, config)
//		blockchain, err := provider.Initialize(context.Background())
//		require.NoError(t, err)
//
//		// The deployer account will use your custom key instead of Geth's default
//		// User accounts will still use Geth's standard test accounts
//	}
//
// Usage with KMS deployer key:
//
//	func TestWithKMSDeployer(t *testing.T) {
//		var once sync.Once
//		config := CTFGethChainProviderConfig{
//			Once:                  &once,
//			ConfirmFunctor:        ConfirmFuncGeth(2 * time.Minute),
//			DeployerTransactorGen: TransactorFromKMS("your-kms-key-id"),
//			T:                     t, // Required when Port is not provided
//		}
//
//		provider := NewCTFGethChainProvider(chainSelector, config)
//		blockchain, err := provider.Initialize(context.Background())
//		require.NoError(t, err)
//
//		// The deployer account will use KMS for signing
//		// User accounts will still use Geth's standard test accounts
//	}
//
// Usage with custom client options:
//
//	func TestWithCustomClientOpts(t *testing.T) {
//		var once sync.Once
//		config := CTFGethChainProviderConfig{
//			Once:           &once,
//			ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
//			ClientOpts: []func(client *rpcclient.MultiClient){
//				func(client *rpcclient.MultiClient) {
//					// Custom client configuration
//					client.SetTimeout(30 * time.Second)
//				},
//			},
//			T: t, // Required when Port is not provided
//		}
//
//		provider := NewCTFGethChainProvider(chainSelector, config)
//		blockchain, err := provider.Initialize(context.Background())
//		require.NoError(t, err)
//
//		// The MultiClient will use the custom configuration options
//	}
//
// Usage with custom Docker command parameters:
//
//		func TestWithDockerCmdOverrides(t *testing.T) {
//			var once sync.Once
//			config := CTFGethChainProviderConfig{
//				Once:           &once,
//				ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
//				DockerCmdParamsOverrides: []string{
//	         "--miner.gasprice", "1000000000",  // Set min gas price to 1 gwei
//	         "--miner.threads", "1",            // Use 1 mining thread
//	         "--cache", "1024",                 // Allocate 1GB cache
//				},
//				T: t, // Required when Port is not provided
//			}
//
//			provider := NewCTFGethChainProvider(chainSelector, config)
//			blockchain, err := provider.Initialize(context.Background())
//			require.NoError(t, err)
//
//			// Geth will run with the custom parameters
//		}
//
// Usage with custom Port and Image:
//
//	func TestWithCustomContainer(t *testing.T) {
//		var once sync.Once
//		config := CTFGethChainProviderConfig{
//			Once:           &once,
//			ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
//			Port:           "8546",                    // Use specific port (T not required when Port is provided)
//			Image:          "ethereum/client-go:stable", // Custom Geth image/version if desired
//		}
//
//		provider := NewCTFGethChainProvider(chainSelector, config)
//		blockchain, err := provider.Initialize(context.Background())
//		require.NoError(t, err)
//
//		// Geth will run on port 8546, using the specified image
//		// Chain ID is automatically derived from the chainSelector
//	}
//
// Usage in production/non-test contexts with manual cleanup:
//
//	func main() {
//		var once sync.Once
//		config := CTFGethChainProviderConfig{
//			Once:           &once,
//			ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
//			Port:           "8546", // T not required when Port is provided
//		}
//
//		provider := NewCTFGethChainProvider(chainSelector, config)
//		blockchain, err := provider.Initialize(context.Background())
//		if err != nil {
//			log.Fatal(err)
//		}
//
//		// Use the blockchain...
//
//		// Important: Clean up the container when done
//		defer func() {
//			if err := provider.Cleanup(context.Background()); err != nil {
//				log.Printf("Failed to cleanup container: %v", err)
//			}
//		}()
//	}
//
// # Chain Selectors
//
// Common chain selectors for Geth testing:
//   - 13264668187771770619: Chain ID 31337 (default Geth chain ID)
//
// # Standard Test Accounts
//
// This provider ships with a built-in pool of Geth-style test keys:
//   - Account 0 (Deployer): 0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266
//   - Account 1:            0x70997970c51812dc3a010c7d01b50e0d17dc79c8
//   - Account 2:            0x3c44cdddb6a900fa2b585dd299e03d12fa4293bc
//   - Account 3:            0x90f79bf6eb2c4f870365e785982e1f101e93b906
//   - Account 4:            0x15d34aaf54267db7d7c367839aaf71a00a2c6a65
//
// Behavior:
//   - By default, only the deployer (Account 0) is created.
//   - If config.AdditionalAccounts == true, *all* extra users (Accounts 1..N) from the pool
//     are created and auto-funded during Initialize().
//
// # Port Management
//
// The provider supports two modes for port allocation:
//   - Explicit Port: When Port is specified in the config, that exact port will be used
//   - Automatic Port: When Port is empty, a free port is automatically allocated using freeport
//     (requires T field to be set in the config for cleanup management)
//
// # Requirements
//
//   - Docker must be installed and running
//   - CTF framework must be properly configured
//   - Sufficient system resources for container execution
//   - When using automatic port allocation (Port not specified), T field must be set in config
package provider

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-testing-framework/framework"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/smartcontractkit/freeport"
	"github.com/testcontainers/testcontainers-go"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider/rpcclient"
)

// gethTestPrivateKeys contains the standard Geth viable accounts.
// By default, Geth creates only the first account (index 0) as the deployer account.
var gethTestPrivateKeys = []string{
	"ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", // Account 0: 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266
	"59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d", // Account 1: 0x70997970C51812dc3A010C7d01b50e0d17dc79C8
	"5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a", // Account 2: 0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC
	"7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6", // Account 3: 0x90F79bf6EB2c4f870365E785982E1f101E93b906
	"47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a", // Account 4: 0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65
}

// CTFGethChainProviderConfig holds the configuration to initialize the CTFGethChainProvider.
//
// This configuration struct provides all necessary parameters to set up an Geth EVM chain
// instance running inside a Chainlink Testing Framework (CTF) Docker container.
type CTFGethChainProviderConfig struct {
	// Required: A sync.Once instance to ensure that the CTF framework only sets up the new
	// DefaultNetwork once
	Once *sync.Once

	// Required: ConfirmFunctor is a type that generates a confirmation function for transactions.
	// Use ConfirmFuncGeth to use the Geth client for transaction confirmation, or
	// ConfirmFuncSeth to use the Seth client for transaction confirmation with richer debugging.
	//
	// If in doubt, use ConfirmFuncGeth.
	ConfirmFunctor ConfirmFunctor

	// Optional: DeployerTransactorGen is a generator for the deployer key. Use TransactorFromRaw
	// to create a deployer key from a private key, or TransactorFromKMS to create a deployer
	// key from a KMS key. If not provided, the default Geth deployer account will be used.
	DeployerTransactorGen SignerGenerator

	// Optional: ClientOpts are additional options to configure the MultiClient used by the
	// CTFGethChainProvider. These options are applied to the MultiClient instance created by the
	// provider. You can use this to set up custom HTTP clients, timeouts, or other
	// configurations for the RPC connections.
	ClientOpts []func(client *rpcclient.MultiClient)

	// Optional: DockerCmdParamsOverrides allows customization of Docker command parameters
	// for the Geth container. These parameters are passed directly to the Docker container
	// startup command, enabling advanced Geth configurations such as custom block time,
	// gas limits, or other Geth-specific options.
	DockerCmdParamsOverrides []string

	// Optional: Port specifies the port for the Geth container. If not provided,
	// a free port will be automatically allocated. Use this when you need the Geth
	// instance to run on a specific port.
	Port string

	// Optional: Image specifies the Docker image to use for the Geth container.
	// If not provided, the default Geth image from the CTF framework will be used.
	// This allows using custom Geth builds or specific versions.
	Image string

	// Optional: Whether to create and auto-fund additional user accounts from the built-in pool.
	// When false (default), only the deployer is available.
	// When true, ALL users from gethTestPrivateKeys (excluding the deployer) are created and auto-funded.
	AdditionalAccounts bool

	// Optional: This is only required when Port is not provided so we can use freeport to get a free port.
	// This will be ignored when Port is provided.
	T testing.TB
}

// validate checks if the config fields are valid.
func (c CTFGethChainProviderConfig) validate() error {
	if c.Once == nil {
		return errors.New("sync.Once instance is required")
	}
	if c.ConfirmFunctor == nil {
		return errors.New("confirm functor is required")
	}

	// Validate Port if provided
	if c.Port != "" {
		port, err := strconv.Atoi(c.Port)
		if err != nil {
			return fmt.Errorf("invalid port %s: must be a valid integer", c.Port)
		}
		if port <= 0 || port > 65535 {
			return fmt.Errorf("invalid port %d: must be between 1 and 65535", port)
		}
	} else {
		// T is required when port is not provided (for freeport allocation)
		if c.T == nil {
			return errors.New("field T is required when port is not provided")
		}
	}

	// DeployerTransactorGen is optional - if not provided, default Geth account will be used
	// No additional validation needed since SignerGenerator interface handles validation internally

	// Image is optional and doesn't need validation here

	return nil
}

var _ chain.Provider = (*CTFGethChainProvider)(nil)

// CTFGethChainProvider manages an Geth EVM chain instance running inside a Chainlink Testing
// Framework (CTF) Docker container.
//
// This provider requires Docker to be installed and operational. Spinning up a new container
// can be slow, so it is recommended to initialize the provider only once per test suite or parent
// test to optimize performance.
type CTFGethChainProvider struct {
	selector uint64
	config   CTFGethChainProviderConfig

	chain     *evm.Chain
	httpURL   string
	wsURL     string
	container testcontainers.Container
}

// NewCTFGethChainProvider creates a new CTFGethChainProvider with the given selector and
// configuration.
//
// Parameters:
//   - selector: Chain selector that maps to a specific chain ID
//   - config: Configuration struct containing all necessary setup parameters.
//     Note: config.T is required when config.Port is not provided (for automatic port allocation)
//
// Returns a new CTFGethChainProvider instance ready for initialization.
func NewCTFGethChainProvider(
	selector uint64, config CTFGethChainProviderConfig,
) *CTFGethChainProvider {
	return &CTFGethChainProvider{
		selector: selector,
		config:   config,
	}
}

// Initialize sets up the Geth EVM chain instance managed by this provider. It starts a CTF
// container, initializes the Ethereum client, and sets up the chain instance with the necessary
// transactors and deployer key gathered from the standard Geth test accounts.
func (p *CTFGethChainProvider) Initialize(ctx context.Context) (chain.BlockChain, error) {
	if p.chain != nil {
		return *p.chain, nil // Already initialized
	}

	err := p.config.validate()
	if err != nil {
		return nil, err
	}

	chainID, err := chainsel.GetChainIDFromSelector(p.selector)
	if err != nil {
		return nil, err
	}

	httpURL, err := p.startContainer(ctx, chainID)
	if err != nil {
		return nil, err
	}
	p.httpURL = httpURL

	lggr, err := logger.New()
	if err != nil {
		return nil, err
	}

	client, err := rpcclient.NewMultiClient(lggr, rpcclient.RPCConfig{
		ChainSelector: p.selector,
		RPCs: []rpcclient.RPC{
			{
				Name:               "geth-local",
				HTTPURL:            p.httpURL,
				WSURL:              p.wsURL, // WS is exposed by the CTF container on the same port
				PreferredURLScheme: rpcclient.URLSchemePreferenceHTTP,
			},
		},
	}, p.config.ClientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create multiclient for Geth at %s: %w", httpURL, err)
	}

	// Get the Chain ID as big.Int for transactor generation
	chainIDBigInt, ok := new(big.Int).SetString(chainID, 10)
	if !ok {
		return nil, fmt.Errorf("failed to parse chain ID into big.Int: %s", chainID)
	}

	// Generate deployer key using the provided transactor generator or default Geth account
	var deployerKey *bind.TransactOpts
	var signHashFunc func([]byte) ([]byte, error)

	if p.config.DeployerTransactorGen != nil {
		// Use custom deployer transactor generator
		deployerKey, err = p.config.DeployerTransactorGen.Generate(chainIDBigInt)
		if err != nil {
			return nil, err
		}

		signHashFunc = func(hash []byte) ([]byte, error) {
			return p.config.DeployerTransactorGen.SignHash(hash)
		}
	} else {
		// Use default Geth deployer account
		deployerPrivateKey, parseErr := crypto.HexToECDSA(gethTestPrivateKeys[0])
		if parseErr != nil {
			return nil, parseErr
		}

		deployerKey, err = bind.NewKeyedTransactorWithChainID(deployerPrivateKey, chainIDBigInt)
		if err != nil {
			return nil, err
		}

		signHashFunc = func(hash []byte) ([]byte, error) {
			sig, signErr := crypto.Sign(hash, deployerPrivateKey)
			if signErr != nil {
				return nil, fmt.Errorf("failed to sign hash: %w", signErr)
			}

			return sig, nil
		}
	}

	// Build additional user transactors from the default Geth accounts
	userTransactors, err := p.getUserTransactors(chainID)
	if err != nil {
		return nil, err
	}

	// Auto-fund created users so tests don't need to
	if len(userTransactors) > 0 {
		// 100 ETH in wei
		amountWei := new(big.Int).Mul(big.NewInt(100), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
		if err = p.fundUsers(ctx, client, deployerKey, userTransactors, amountWei); err != nil {
			return nil, fmt.Errorf("fund users: %w", err)
		}
	}

	confirmFunc, err := p.config.ConfirmFunctor.Generate(
		ctx, p.selector, client, deployerKey.From,
	)
	if err != nil {
		return nil, err
	}

	p.chain = &evm.Chain{
		Selector:    p.selector,
		Client:      client,
		DeployerKey: deployerKey,
		Users:       userTransactors,
		Confirm:     confirmFunc,
		SignHash:    signHashFunc,
	}

	return *p.chain, nil
}

// Name returns the human-readable name of the CTFGethChainProvider.
// This name is used for logging and identification purposes.
func (*CTFGethChainProvider) Name() string {
	return "Geth EVM CTF Chain Provider"
}

// ChainSelector returns the chain selector of the Geth EVM chain managed by this provider.
// The chain selector is a unique identifier that maps to a specific blockchain network.
func (p *CTFGethChainProvider) ChainSelector() uint64 {
	return p.selector
}

// BlockChain returns the Geth EVM chain instance managed by this provider.
//
// You must call Initialize before using this method to ensure the chain is properly set up.
// Calling this method before initialization will return an uninitialized chain instance.
//
// Returns the chain.BlockChain interface that can be used for blockchain operations
// such as deploying contracts, sending transactions, and querying blockchain state.
func (p *CTFGethChainProvider) BlockChain() chain.BlockChain {
	return *p.chain
}

// GetNodeHTTPURL returns the external HTTP URL of the first Geth node.
//
// This URL can be used to connect to the Geth node directly for RPC calls or other operations.
// You must call Initialize before using this method to ensure the container is started and the URL is available.
//
// Returns an empty string if the provider has not been initialized yet.
func (p *CTFGethChainProvider) GetNodeHTTPURL() string {
	return p.httpURL
}

// Cleanup terminates the Geth container and cleans up associated resources.
//
// This method provides explicit control over container lifecycle, which is especially
// important when the provider is used outside of test contexts where automatic cleanup
// via testcontainers.CleanupContainer is not available.
//
// It's safe to call this method multiple times - subsequent calls will be no-ops if
// the container has already been terminated.
//
// Returns an error if the container termination fails.
func (p *CTFGethChainProvider) Cleanup(ctx context.Context) error {
	if p.container != nil {
		err := p.container.Terminate(ctx)
		if err != nil {
			return fmt.Errorf("failed to terminate Geth container: %w", err)
		}
		p.container = nil // Clear the reference after successful termination
	}

	return nil
}

// startContainer starts a CTF container for the Geth EVM returning the HTTP URL of the node.
//
// This method handles the Docker container lifecycle including:
//   - Setting up the CTF default network
//   - Port allocation: uses config.Port if provided, otherwise allocates a free port via freeport
//   - Creating and starting the Geth container
//   - Implementing retry logic for robust container startup
//   - Registering container cleanup with the test framework
//
// Returns the external HTTP URL that can be used to connect to the Geth node.
func (p *CTFGethChainProvider) startContainer(ctx context.Context, chainID string) (string, error) {
	var (
		attempts = uint(10)
	)

	err := framework.DefaultNetwork(p.config.Once)
	if err != nil {
		return "", fmt.Errorf("failed to set up CTF default network: %w", err)
	}

	httpURL, err := retry.DoWithData(func() (string, error) {
		var port int
		var portStr string
		if p.config.Port != "" {
			// Use provided port directly - no need to call freeport
			portStr = p.config.Port
			var parseErr error
			port, parseErr = strconv.Atoi(portStr)
			if parseErr != nil {
				return "", fmt.Errorf("invalid port %s: %w", portStr, parseErr)
			}
		} else {
			// Allocate a free port automatically when port is not provided
			if p.config.T == nil {
				return "", errors.New("t is required when port is not provided")
			}
			port = freeport.GetOne(p.config.T)
			portStr = strconv.Itoa(port)
		}

		// Create the input for the Geth blockchain network
		input := &blockchain.Input{
			Type:                     blockchain.TypeGeth,
			ChainID:                  chainID,
			Port:                     portStr,
			Image:                    p.config.Image, // Use custom image if provided, empty string uses default
			DockerCmdParamsOverrides: p.config.DockerCmdParamsOverrides,
		}

		// Create the CTF container for Geth
		output, rerr := blockchain.NewBlockchainNetwork(input)
		if rerr != nil {
			// Return the port to freeport only if it was auto-allocated
			if p.config.Port == "" {
				freeport.Return([]int{port})
			}

			return "", fmt.Errorf("failed to create Geth container: %w", rerr)
		}

		// Store container reference for manual cleanup
		p.container = output.Container

		// Only register cleanup if T is available (for test cleanup)
		if p.config.T != nil {
			testcontainers.CleanupContainer(p.config.T, output.Container)
		}

		// Validate that the ExternalHTTPUrl is not empty
		externalURL := output.Nodes[0].ExternalHTTPUrl
		if externalURL == "" {
			return "", errors.New("container started but ExternalHTTPUrl is empty")
		}

		externalWS := output.Nodes[0].ExternalWSUrl
		if externalWS == "" {
			return "", errors.New("container started but ExternalWSUrl is empty")
		}

		// Perform health check to ensure Geth is ready
		if healthErr := p.waitForGethReady(ctx, externalURL); healthErr != nil {
			return "", fmt.Errorf("geth container started but health check failed: %w", healthErr)
		}

		p.wsURL = externalWS

		return externalURL, nil
	},
		retry.Context(ctx),
		retry.Attempts(attempts),
		retry.Delay(1*time.Second),
		retry.DelayType(retry.FixedDelay),
	)
	if err != nil {
		return "", fmt.Errorf("failed to start CTF Geth container after %d attempts: %w", attempts, err)
	}

	return httpURL, nil
}

// getUserTransactors generates user transactors for additional Geth accounts.
//
// Behavior:
//   - If config.AdditionalAccounts == false, it returns nil (no extra users).
//   - If config.AdditionalAccounts == true, it creates transactors for all accounts
//     in gethTestPrivateKeys except index 0 (the deployer).
//
// These users are auto-funded later in Initialize() via fundUsers().
//
// Parameters:
//   - chainID: The chain ID as a decimal string used to create chain-specific transactors.
//
// Returns a slice of *bind.TransactOpts for user accounts (or nil when disabled).
func (p *CTFGethChainProvider) getUserTransactors(chainID string) ([]*bind.TransactOpts, error) {
	if len(gethTestPrivateKeys) <= 1 {
		return nil, errors.New("at least 2 geth test private keys are required")
	}

	// Parse chainID into big.Int for transactor generation
	cid, ok := new(big.Int).SetString(chainID, 10)
	if !ok {
		return nil, fmt.Errorf("failed to parse chain ID into big.Int: %s", chainID)
	}

	// If flag is false, don't create any additional users
	if !p.config.AdditionalAccounts {
		return nil, nil
	}

	// Create user transactors for all available keys except deployer (index 0)
	totalUsers := len(gethTestPrivateKeys) - 1
	transactors := make([]*bind.TransactOpts, 0, totalUsers)

	for i := 1; i < len(gethTestPrivateKeys); i++ {
		pk := gethTestPrivateKeys[i]
		privateKey, err := crypto.HexToECDSA(pk)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key %d: %w", i, err)
		}

		transactor, err := bind.NewKeyedTransactorWithChainID(privateKey, cid)
		if err != nil {
			return nil, fmt.Errorf("failed to create transactor %d: %w", i, err)
		}

		transactors = append(transactors, transactor)
	}

	return transactors, nil
}

// waitForGethReady performs a health check on the Geth node to ensure it's ready to accept requests.
// It sends a simple JSON-RPC request to check if the node is responding correctly.
func (p *CTFGethChainProvider) waitForGethReady(ctx context.Context, httpURL string) error {
	const (
		maxAttempts = 30
		retryDelay  = 1 * time.Second
	)

	// Simple JSON-RPC request to check if Geth is ready
	jsonRPCRequest := `{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}`

	return retry.Do(func() error {
		reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, httpURL, strings.NewReader(jsonRPCRequest))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to make request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
		}

		return nil
	},
		retry.Context(ctx),
		retry.Attempts(maxAttempts),
		retry.Delay(retryDelay),
		retry.DelayType(retry.FixedDelay),
	)
}

// fundUsers transfers 'amountWei' from the deployer to each user. Only called when additional accounts are created.
func (p *CTFGethChainProvider) fundUsers(
	ctx context.Context,
	cli *rpcclient.MultiClient,
	deployer *bind.TransactOpts,
	users []*bind.TransactOpts,
	amountWei *big.Int,
) error {
	if len(users) == 0 {
		return nil
	}

	from := deployer.From

	// Get deployer nonce
	nonce, err := cli.PendingNonceAt(ctx, from)
	if err != nil {
		return fmt.Errorf("get deployer nonce: %w", err)
	}

	// Try EIP-1559 first
	var (
		gasLimit uint64 = 21_000
	)
	tip, _ := cli.SuggestGasTipCap(ctx)
	head, _ := cli.HeaderByNumber(ctx, nil)

	for _, u := range users {
		if u == nil {
			continue
		}
		to := u.From

		var tx *types.Transaction
		if head != nil && head.BaseFee != nil && head.BaseFee.Sign() > 0 && tip != nil {
			feeCap := new(big.Int).Add(head.BaseFee, new(big.Int).Mul(tip, big.NewInt(2)))
			tx = types.NewTx(&types.DynamicFeeTx{
				Nonce:     nonce,
				To:        &to,
				Value:     new(big.Int).Set(amountWei),
				Gas:       gasLimit,
				GasTipCap: tip,
				GasFeeCap: feeCap,
			})
		} else {
			gasPrice, err := cli.SuggestGasPrice(ctx)
			if err != nil {
				return fmt.Errorf("suggest gas price: %w", err)
			}
			tx = types.NewTransaction(nonce, to, new(big.Int).Set(amountWei), gasLimit, gasPrice, nil)
		}

		signed, err := deployer.Signer(from, tx)
		if err != nil {
			return fmt.Errorf("sign tx: %w", err)
		}
		if err := cli.SendTransaction(ctx, signed); err != nil {
			return fmt.Errorf("send tx to %s: %w", to.Hex(), err)
		}

		txHash := signed.Hash()
		// best-effort context for receipt polling
		pollCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		for {
			receipt, rerr := cli.TransactionReceipt(pollCtx, txHash)
			if rerr == nil && receipt != nil {
				break
			}
			select {
			case <-pollCtx.Done():
				cancel()
				return fmt.Errorf("timeout waiting receipt for %s", txHash.Hex())
			case <-time.After(500 * time.Millisecond):
			}
		}
		cancel() // ensure we cancel before next iteration
		nonce++
	}

	return nil
}

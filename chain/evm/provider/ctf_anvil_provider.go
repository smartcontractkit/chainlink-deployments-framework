// Package provider contains EVM chain providers for the Chainlink Deployments Framework.
//
// This file implements CTFAnvilChainProvider, which provides Anvil EVM chain instances
// running inside Chainlink Testing Framework (CTF) Docker containers.
//
// # Anvil Integration
//
// Anvil is a local Ethereum node designed for development and testing, part of the Foundry
// toolkit. It provides fast, deterministic blockchain simulation with pre-funded accounts
// and configurable mining behavior.
//
// # Usage Patterns
//
// Basic usage for simple testing:
//
//	func TestBasicContract(t *testing.T) {
//		var once sync.Once
//		config := CTFAnvilChainProviderConfig{
//			Once:           &once,
//			ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
//			T:              t, // Required when Port is not provided
//		}
//
//		provider := NewCTFAnvilChainProvider(chainSelector, config)
//		blockchain, err := provider.Initialize(context.Background())
//		require.NoError(t, err)
//
//		// Deploy and test contracts...
//	}
//
// Advanced usage with custom configuration:
//
//	func TestAdvancedScenario(t *testing.T) {
//		var once sync.Once
//		config := CTFAnvilChainProviderConfig{
//			Once:                  &once,
//			ConfirmFunctor:        ConfirmFuncGeth(30 * time.Second),
//			NumAdditionalAccounts: 5, // Limit to 5 additional accounts + deployer
//			T:                     t, // Required when Port is not provided
//		}
//
//		provider := NewCTFAnvilChainProvider(chainSelector, config)
//		blockchain, err := provider.Initialize(context.Background())
//		require.NoError(t, err)
//
//		// Run complex multi-account tests...
//	}
//
// Usage with custom deployer key (using TransactorFromRaw):
//
//	func TestWithCustomDeployer(t *testing.T) {
//		var once sync.Once
//		config := CTFAnvilChainProviderConfig{
//			Once:                  &once,
//			ConfirmFunctor:        ConfirmFuncGeth(2 * time.Minute),
//			DeployerTransactorGen: TransactorFromRaw("your-custom-private-key-here"), // 64 chars hex, no 0x prefix
//			T:                     t, // Required when Port is not provided
//		}
//
//		provider := NewCTFAnvilChainProvider(chainSelector, config)
//		blockchain, err := provider.Initialize(context.Background())
//		require.NoError(t, err)
//
//		// The deployer account will use your custom key instead of Anvil's default
//		// User accounts will still use Anvil's standard test accounts
//	}
//
// Usage with KMS deployer key:
//
//	func TestWithKMSDeployer(t *testing.T) {
//		var once sync.Once
//		config := CTFAnvilChainProviderConfig{
//			Once:                  &once,
//			ConfirmFunctor:        ConfirmFuncGeth(2 * time.Minute),
//			DeployerTransactorGen: TransactorFromKMS("your-kms-key-id"),
//			T:                     t, // Required when Port is not provided
//		}
//
//		provider := NewCTFAnvilChainProvider(chainSelector, config)
//		blockchain, err := provider.Initialize(context.Background())
//		require.NoError(t, err)
//
//		// The deployer account will use KMS for signing
//		// User accounts will still use Anvil's standard test accounts
//	}
//
// Usage with custom client options:
//
//	func TestWithCustomClientOpts(t *testing.T) {
//		var once sync.Once
//		config := CTFAnvilChainProviderConfig{
//			Once:           &once,
//			ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
//			ClientOpts: []func(client *deployment.MultiClient){
//				func(client *deployment.MultiClient) {
//					// Custom client configuration
//					client.SetTimeout(30 * time.Second)
//				},
//			},
//			T: t, // Required when Port is not provided
//		}
//
//		provider := NewCTFAnvilChainProvider(chainSelector, config)
//		blockchain, err := provider.Initialize(context.Background())
//		require.NoError(t, err)
//
//		// The MultiClient will use the custom configuration options
//	}
//
// Usage with custom Docker command parameters:
//
//	func TestWithDockerCmdOverrides(t *testing.T) {
//		var once sync.Once
//		config := CTFAnvilChainProviderConfig{
//			Once:           &once,
//			ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
//			DockerCmdParamsOverrides: []string{
//				"--block-time", "2",          // Mine blocks every 2 seconds
//				"--gas-limit", "30000000",    // Set gas limit to 30M
//				"--gas-price", "1000000000",  // Set gas price to 1 gwei
//			},
//			T: t, // Required when Port is not provided
//		}
//
//		provider := NewCTFAnvilChainProvider(chainSelector, config)
//		blockchain, err := provider.Initialize(context.Background())
//		require.NoError(t, err)
//
//		// Anvil will run with the custom parameters
//	}
//
// Usage with custom Port and Image:
//
//	func TestWithCustomContainer(t *testing.T) {
//		var once sync.Once
//		config := CTFAnvilChainProviderConfig{
//			Once:           &once,
//			ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
//			Port:           "8545",                    // Use specific port (T not required when Port is provided)
//			Image:          "ghcr.io/foundry-rs/foundry:latest", // Custom Anvil image
//		}
//
//		provider := NewCTFAnvilChainProvider(chainSelector, config)
//		blockchain, err := provider.Initialize(context.Background())
//		require.NoError(t, err)
//
//		// Anvil will run on port 8545, using the specified image
//		// Chain ID is automatically derived from the chainSelector
//	}
//
// Usage in production/non-test contexts with manual cleanup:
//
//	func main() {
//		var once sync.Once
//		config := CTFAnvilChainProviderConfig{
//			Once:           &once,
//			ConfirmFunctor: ConfirmFuncGeth(2 * time.Minute),
//			Port:           "8545", // T not required when Port is provided
//		}
//
//		provider := NewCTFAnvilChainProvider(chainSelector, config)
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
// Common chain selectors for Anvil testing:
//   - 13264668187771770619: Chain ID 31337 (default Anvil chain ID)
//
// # Standard Test Accounts
//
// The provider uses Anvil's standard test accounts:
//   - Account 0 (Deployer): 0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266
//   - Account 1: 0x70997970c51812dc3a010c7d01b50e0d17dc79c8
//   - Account 2: 0x3c44cdddb6a900fa2b585dd299e03d12fa4293bc
//   - ... up to 10 total pre-funded accounts
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
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-testing-framework/framework"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/smartcontractkit/freeport"
	"github.com/testcontainers/testcontainers-go"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

// anvilTestPrivateKeys contains the standard Anvil test accounts.
// These are the well-known private keys that Anvil uses for its default accounts.
var anvilTestPrivateKeys = []string{
	"ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", // Account 0: 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266
	"59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d", // Account 1: 0x70997970C51812dc3A010C7d01b50e0d17dc79C8
	"5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a", // Account 2: 0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC
	"7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6", // Account 3: 0x90F79bf6EB2c4f870365E785982E1f101E93b906
	"47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a", // Account 4: 0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65
}

// CTFAnvilChainProviderConfig holds the configuration to initialize the CTFAnvilChainProvider.
//
// This configuration struct provides all necessary parameters to set up an Anvil EVM chain
// instance running inside a Chainlink Testing Framework (CTF) Docker container.
type CTFAnvilChainProviderConfig struct {
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
	// key from a KMS key. If not provided, the default Anvil deployer account will be used.
	DeployerTransactorGen SignerGenerator

	// Optional: ClientOpts are additional options to configure the MultiClient used by the
	// CTFAnvilChainProvider. These options are applied to the MultiClient instance created by the
	// provider. You can use this to set up custom HTTP clients, timeouts, or other
	// configurations for the RPC connections.
	ClientOpts []func(client *deployment.MultiClient)

	// Optional: DockerCmdParamsOverrides allows customization of Docker command parameters
	// for the Anvil container. These parameters are passed directly to the Docker container
	// startup command, enabling advanced Anvil configurations such as custom block time,
	// gas limits, or other Anvil-specific options.
	DockerCmdParamsOverrides []string

	// Optional: Port specifies the port for the Anvil container. If not provided,
	// a free port will be automatically allocated. Use this when you need the Anvil
	// instance to run on a specific port.
	Port string

	// Optional: Image specifies the Docker image to use for the Anvil container.
	// If not provided, the default Anvil image from the CTF framework will be used.
	// This allows using custom Anvil builds or specific versions.
	Image string

	// Optional: Number of additional accounts to generate beyond the default Anvil accounts.
	// If not specified, defaults to using all available default Anvil accounts.
	NumAdditionalAccounts uint

	// Optional: This is only required when Port is not provided so we can use freeport to get a free port.
	// This will be ignored when Port is provided.
	T testing.TB
}

// validate checks if the config fields are valid.
func (c CTFAnvilChainProviderConfig) validate() error {
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

	// DeployerTransactorGen is optional - if not provided, default Anvil account will be used
	// No additional validation needed since SignerGenerator interface handles validation internally

	// Image is optional and doesn't need validation here

	return nil
}

var _ chain.Provider = (*CTFAnvilChainProvider)(nil)

// CTFAnvilChainProvider manages an Anvil EVM chain instance running inside a Chainlink Testing
// Framework (CTF) Docker container.
//
// This provider requires Docker to be installed and operational. Spinning up a new container
// can be slow, so it is recommended to initialize the provider only once per test suite or parent
// test to optimize performance.
type CTFAnvilChainProvider struct {
	selector uint64
	config   CTFAnvilChainProviderConfig

	chain     *evm.Chain
	httpURL   string
	container testcontainers.Container
}

// NewCTFAnvilChainProvider creates a new CTFAnvilChainProvider with the given selector and
// configuration.
//
// Parameters:
//   - selector: Chain selector that maps to a specific chain ID
//   - config: Configuration struct containing all necessary setup parameters.
//     Note: config.T is required when config.Port is not provided (for automatic port allocation)
//
// Returns a new CTFAnvilChainProvider instance ready for initialization.
func NewCTFAnvilChainProvider(
	selector uint64, config CTFAnvilChainProviderConfig,
) *CTFAnvilChainProvider {
	return &CTFAnvilChainProvider{
		selector: selector,
		config:   config,
	}
}

// Initialize sets up the Anvil EVM chain instance managed by this provider. It starts a CTF
// container, initializes the Ethereum client, and sets up the chain instance with the necessary
// transactors and deployer key gathered from the standard Anvil test accounts.
func (p *CTFAnvilChainProvider) Initialize(ctx context.Context) (chain.BlockChain, error) {
	if p.chain != nil {
		return *p.chain, nil // Already initialized
	}

	err := p.config.validate()
	if err != nil {
		return nil, err
	}

	chainID, err := chain_selectors.GetChainIDFromSelector(p.selector)
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

	client, err := deployment.NewMultiClient(lggr, deployment.RPCConfig{
		ChainSelector: p.selector,
		RPCs: []deployment.RPC{
			{
				Name:               "anvil-local",
				HTTPURL:            httpURL,
				WSURL:              "", // Anvil typically doesn't provide WebSocket, only HTTP
				PreferredURLScheme: deployment.URLSchemePreferenceHTTP,
			},
		},
	}, p.config.ClientOpts...)
	if err != nil {
		return nil, err
	}

	// Get the Chain ID as big.Int for transactor generation
	chainIDBigInt, ok := new(big.Int).SetString(chainID, 10)
	if !ok {
		return nil, fmt.Errorf("failed to parse chain ID into big.Int: %s", chainID)
	}

	// Generate deployer key using the provided transactor generator or default Anvil account
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
		// Use default Anvil deployer account
		deployerPrivateKey, parseErr := crypto.HexToECDSA(anvilTestPrivateKeys[0])
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

	// Build additional user transactors from the default Anvil accounts
	userTransactors, err := p.getUserTransactors(chainID)
	if err != nil {
		return nil, err
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

// Name returns the human-readable name of the CTFAnvilChainProvider.
// This name is used for logging and identification purposes.
func (*CTFAnvilChainProvider) Name() string {
	return "Anvil EVM CTF Chain Provider"
}

// ChainSelector returns the chain selector of the Anvil EVM chain managed by this provider.
// The chain selector is a unique identifier that maps to a specific blockchain network.
func (p *CTFAnvilChainProvider) ChainSelector() uint64 {
	return p.selector
}

// BlockChain returns the Anvil EVM chain instance managed by this provider.
//
// You must call Initialize before using this method to ensure the chain is properly set up.
// Calling this method before initialization will return an uninitialized chain instance.
//
// Returns the chain.BlockChain interface that can be used for blockchain operations
// such as deploying contracts, sending transactions, and querying blockchain state.
func (p *CTFAnvilChainProvider) BlockChain() chain.BlockChain {
	return *p.chain
}

// GetNodeHTTPURL returns the external HTTP URL of the first Anvil node.
//
// This URL can be used to connect to the Anvil node directly for RPC calls or other operations.
// You must call Initialize before using this method to ensure the container is started and the URL is available.
//
// Returns an empty string if the provider has not been initialized yet.
func (p *CTFAnvilChainProvider) GetNodeHTTPURL() string {
	return p.httpURL
}

// Cleanup terminates the Anvil container and cleans up associated resources.
//
// This method provides explicit control over container lifecycle, which is especially
// important when the provider is used outside of test contexts where automatic cleanup
// via testcontainers.CleanupContainer is not available.
//
// It's safe to call this method multiple times - subsequent calls will be no-ops if
// the container has already been terminated.
//
// Returns an error if the container termination fails.
func (p *CTFAnvilChainProvider) Cleanup(ctx context.Context) error {
	if p.container != nil {
		err := p.container.Terminate(ctx)
		if err != nil {
			return fmt.Errorf("failed to terminate Anvil container: %w", err)
		}
		p.container = nil // Clear the reference after successful termination
	}

	return nil
}

// startContainer starts a CTF container for the Anvil EVM returning the HTTP URL of the node.
//
// This method handles the Docker container lifecycle including:
//   - Setting up the CTF default network
//   - Port allocation: uses config.Port if provided, otherwise allocates a free port via freeport
//   - Creating and starting the Anvil container
//   - Implementing retry logic for robust container startup
//   - Registering container cleanup with the test framework
//
// Returns the external HTTP URL that can be used to connect to the Anvil node.
func (p *CTFAnvilChainProvider) startContainer(ctx context.Context, chainID string) (string, error) {
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

		// Create the input for the Anvil blockchain network
		input := &blockchain.Input{
			Type:                     blockchain.TypeAnvil,
			ChainID:                  chainID,
			Port:                     portStr,
			Image:                    p.config.Image, // Use custom image if provided, empty string uses default
			DockerCmdParamsOverrides: p.config.DockerCmdParamsOverrides,
		}

		// Create the CTF container for Anvil
		output, rerr := blockchain.NewBlockchainNetwork(input)
		if rerr != nil {
			// Return the port to freeport only if it was auto-allocated
			if p.config.Port == "" {
				freeport.Return([]int{port})
			}

			return "", fmt.Errorf("failed to create Anvil container: %w", rerr)
		}

		// Store container reference for manual cleanup
		p.container = output.Container

		// Only register cleanup if T is available (for test cleanup)
		if p.config.T != nil {
			testcontainers.CleanupContainer(p.config.T, output.Container)
		}

		return output.Nodes[0].ExternalHTTPUrl, nil
	},
		retry.Context(ctx),
		retry.Attempts(attempts),
		retry.Delay(1*time.Second),
		retry.DelayType(retry.FixedDelay),
	)
	if err != nil {
		return "", fmt.Errorf("failed to start CTF Anvil container after %d attempts: %w", attempts, err)
	}

	return httpURL, nil
}

// getUserTransactors generates user transactors from the standard Anvil test accounts.
//
// This method creates bind.TransactOpts instances from the well-known Anvil test private keys
// for user accounts (excluding the deployer). These accounts are pre-funded in Anvil and
// provide deterministic addresses for testing.
//
// The standard Anvil user accounts used are:
//   - Account 1: 0x70997970C51812dc3A010C7d01b50e0d17dc79C8
//   - Account 2: 0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC
//   - Account 3: 0x90F79bf6EB2c4f870365E785982E1f101E93b906
//   - Account 4: 0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65
//
// The number of user accounts generated can be limited by the NumAdditionalAccounts configuration.
//
// Parameters:
//   - chainID: The chain ID as a string, used to create chain-specific transactors
//
// Returns a slice of bind.TransactOpts ready for use as user accounts.
func (p *CTFAnvilChainProvider) getUserTransactors(chainID string) ([]*bind.TransactOpts, error) {
	if len(anvilTestPrivateKeys) <= 1 {
		return nil, errors.New("at least 2 anvil test private keys are required")
	}

	cid, ok := new(big.Int).SetString(chainID, 10)
	if !ok {
		return nil, fmt.Errorf("failed to parse chain ID into big.Int: %s", chainID)
	}

	// Determine how many user accounts to create (excluding deployer)
	maxUserAccounts := uint(len(anvilTestPrivateKeys)) - 1 // -1 to exclude deployer account
	if p.config.NumAdditionalAccounts > 0 && p.config.NumAdditionalAccounts < maxUserAccounts {
		maxUserAccounts = p.config.NumAdditionalAccounts
	}

	transactors := make([]*bind.TransactOpts, 0, maxUserAccounts)

	// Create user account transactors from standard Anvil accounts (starting from index 1)
	for i := uint(1); i <= maxUserAccounts && i < uint(len(anvilTestPrivateKeys)); i++ {
		pk := anvilTestPrivateKeys[i]
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

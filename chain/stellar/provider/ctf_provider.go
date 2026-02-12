package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/avast/retry-go/v4"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-testing-framework/framework"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/smartcontractkit/freeport"
	"github.com/stellar/go-stellar-sdk/clients/rpcclient"
	"github.com/testcontainers/testcontainers-go"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/stellar"
)

// CTFChainProviderConfig holds the configuration to initialize the CTFChainProvider.
type CTFChainProviderConfig struct {
	// Required: A generator for the deployer keypair. Use KeypairFromHex to create a deployer
	// keypair from a hex-encoded private key, or KeypairRandom to generate a random keypair.
	DeployerKeypairGen KeypairGenerator

	// Required: A sync.Once instance to ensure that the CTF framework only sets up the new
	// DefaultNetwork once
	Once *sync.Once

	// Optional: Docker image to use for the Stellar localnet. If empty, defaults to defaultStellarImage.
	Image string

	// Optional: Network passphrase for the Stellar network. If empty, defaults to defaultNetworkPassphrase.
	NetworkPassphrase string

	// Optional: Custom environment variables to pass to the Stellar container.
	CustomEnv map[string]string

	// Optional: Port to expose the Stellar container on. If 0 or not specified, a free port will be
	// automatically selected using freeport.
	Port int
}

// validate checks if the CTFChainProviderConfig is valid.
func (c CTFChainProviderConfig) validate() error {
	if c.DeployerKeypairGen == nil {
		return errors.New("deployer keypair generator is required")
	}

	if c.Once == nil {
		return errors.New("sync.Once instance is required")
	}

	return nil
}

var _ chain.Provider = (*CTFChainProvider)(nil)

// CTFChainProvider manages a Stellar chain instance running inside a Chainlink Testing Framework (CTF) Docker container.
//
// This provider requires Docker to be installed and operational. Spinning up a new container can be slow,
// so it is recommended to initialize the provider only once per test suite or parent test to optimize performance.
type CTFChainProvider struct {
	t        *testing.T
	selector uint64
	config   CTFChainProviderConfig

	chain *stellar.Chain
}

// NewCTFChainProvider creates a new CTFChainProvider with the given selector and configuration.
func NewCTFChainProvider(
	t *testing.T, selector uint64, config CTFChainProviderConfig,
) *CTFChainProvider {
	t.Helper()

	p := &CTFChainProvider{
		t:        t,
		selector: selector,
		config:   config,
	}

	return p
}

// Initialize sets up the Stellar chain by validating the configuration, starting a CTF container,
// generating a deployer keypair, and constructing the chain instance.
func (p *CTFChainProvider) Initialize(ctx context.Context) (chain.BlockChain, error) {
	if p.chain != nil {
		return *p.chain, nil // Already initialized
	}

	if err := p.config.validate(); err != nil {
		return nil, fmt.Errorf("failed to validate provider config: %w", err)
	}

	// Get the Stellar Chain ID
	chainID, err := chainsel.GetChainIDFromSelector(p.selector)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID from selector %d: %w", p.selector, err)
	}

	// Generate the deployer keypair
	deployerSigner, err := p.config.DeployerKeypairGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate deployer keypair: %w", err)
	}

	// Start the CTF Container
	url, friendbotURL, client, err := p.startContainer(ctx, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Get network passphrase
	networkPassphrase := p.config.NetworkPassphrase
	if networkPassphrase == "" {
		networkPassphrase = blockchain.DefaultStellarNetworkPassphrase
	}

	// Construct the chain
	p.chain = &stellar.Chain{
		ChainMetadata:     stellar.ChainMetadata{Selector: p.selector},
		Client:            client,
		Signer:            deployerSigner,
		URL:               url,
		FriendbotURL:      friendbotURL,
		NetworkPassphrase: networkPassphrase,
	}

	return *p.chain, nil
}

// Name returns the name of the CTFChainProvider.
func (*CTFChainProvider) Name() string {
	return "Stellar CTF Chain Provider"
}

// ChainSelector returns the chain selector of the Stellar chain managed by this provider.
func (p *CTFChainProvider) ChainSelector() uint64 {
	return p.selector
}

// BlockChain returns the Stellar chain instance managed by this provider. You must call Initialize
// before using this method to ensure the chain is properly set up.
func (p *CTFChainProvider) BlockChain() chain.BlockChain {
	return *p.chain
}

// startContainer starts a CTF container for the Stellar chain with the given chain ID.
// It returns the Soroban RPC URL, Friendbot URL, and the RPC client to interact with it.
func (p *CTFChainProvider) startContainer(
	ctx context.Context, chainID string,
) (string, string, *rpcclient.Client, error) {
	var (
		attempts = uint(10)
	)

	// initialize the docker network used by CTF
	err := framework.DefaultNetwork(p.config.Once)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to initialize default network: %w", err)
	}

	type stellarResult struct {
		rpcURL       string
		friendbotURL string
	}

	result, err := retry.DoWithData(func() (stellarResult, error) {
		port, usedFreeport := p.getPort()

		// Build environment variables
		env := p.buildEnvVars()

		// spin up Stellar with CTFv2
		output, rerr := blockchain.NewBlockchainNetwork(&blockchain.Input{
			Type:      blockchain.TypeStellar,
			ChainID:   chainID,
			Port:      strconv.Itoa(port),
			Image:     p.getImage(),
			CustomEnv: env,
		})
		if rerr != nil {
			// Return the ports to freeport to avoid leaking them during retries
			// Only return if we obtained the port from freeport
			if usedFreeport {
				freeport.Return([]int{port})
			}

			return stellarResult{}, rerr
		}

		testcontainers.CleanupContainer(p.t, output.Container)

		// Extract URLs from CTF framework output
		rpcURL := output.Nodes[0].ExternalHTTPUrl
		friendbotURL := ""
		if output.NetworkSpecificData != nil && output.NetworkSpecificData.StellarNetwork != nil {
			friendbotURL = output.NetworkSpecificData.StellarNetwork.FriendbotURL
		}

		return stellarResult{
			rpcURL:       rpcURL,
			friendbotURL: friendbotURL,
		}, nil
	},
		retry.Context(ctx),
		retry.Attempts(attempts),
		retry.Delay(1*time.Second),
		retry.DelayType(retry.FixedDelay),
		retry.OnRetry(func(attempt uint, err error) {
			p.t.Logf("Attempt %d/%d: Failed to start CTF Stellar container: %v", attempt+1, attempts, err)
		}),
	)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to start CTF Stellar container after %d attempts: %w", attempts, err)
	}

	// Use the URLs provided by the CTF framework
	sorobanRPCURL := result.rpcURL
	friendbotURL := result.friendbotURL

	// Create the Soroban RPC client with timeout
	client := rpcclient.NewClient(sorobanRPCURL, &http.Client{
		Timeout: 60 * time.Second,
	})

	// Check if the Soroban RPC endpoint is healthy
	if err := checkStellarNodeHealth(ctx, p.t, sorobanRPCURL); err != nil {
		return "", "", nil, fmt.Errorf("stellar node health check failed: %w", err)
	}

	return sorobanRPCURL, friendbotURL, client, nil
}

// getPort returns the port to use for the Stellar container.
// It returns the port and a boolean indicating whether freeport was used.
func (p *CTFChainProvider) getPort() (int, bool) {
	if p.config.Port != 0 {
		return p.config.Port, false
	}

	return freeport.GetOne(p.t), true
}

// getImage returns the Docker image to use for the Stellar container.
func (p *CTFChainProvider) getImage() string {
	if p.config.Image != "" {
		return p.config.Image
	}

	return blockchain.DefaultStellarImage
}

// buildEnvVars constructs the environment variables for the Stellar container.
func (p *CTFChainProvider) buildEnvVars() map[string]string {
	env := make(map[string]string)

	// Add custom environment variables
	for k, v := range p.config.CustomEnv {
		env[k] = v
	}

	return env
}

// checkStellarNodeHealth checks if the Stellar node is ready by checking if the RPC endpoint is accessible.
func checkStellarNodeHealth(ctx context.Context, t *testing.T, sorobanRPCURL string) error {
	t.Helper()
	var lastErr error

	err := retry.Do(func() error {
		client := &http.Client{Timeout: 5 * time.Second}

		// Try to reach the Soroban RPC endpoint
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, sorobanRPCURL, nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			return lastErr
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to reach Soroban RPC endpoint: %w", err)
			return lastErr
		}
		defer resp.Body.Close()

		// For Stellar Quickstart, the endpoint is ready when it's accessible
		// Even if it returns 4xx, it means the service is up
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("stellar node not ready, status code: %d", resp.StatusCode)
			return lastErr
		}

		t.Logf("Stellar Soroban RPC node is healthy (status: %d)", resp.StatusCode)

		return nil
	},
		retry.Context(ctx),
		retry.Attempts(30),
		retry.Delay(1*time.Second),
		retry.DelayType(retry.FixedDelay),
	)

	if err != nil {
		return fmt.Errorf("stellar node did not become healthy: %w", lastErr)
	}

	return nil
}

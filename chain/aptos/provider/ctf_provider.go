package provider

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	aptoslib "github.com/aptos-labs/aptos-go-sdk"
	"github.com/avast/retry-go/v4"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-testing-framework/framework"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/smartcontractkit/freeport"
	"github.com/testcontainers/testcontainers-go"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos"
)

// CTFChainProviderConfig holds the configuration to initialize the CTFChainProvider.
type CTFChainProviderConfig struct {
	// Required: A generator for the deployer signer account. Use AccountGenCTFDefault to
	// create a deployer signer from the default CTF account. Alternatively, you can use
	// AccountGenNewSingleSender to create a new single sender account.
	DeployerSignerGen AccountGenerator

	// Required: A sync.Once instance to ensure that the CTF framework only sets up the new
	// DefaultNetwork once
	Once *sync.Once
}

// validate checks if the CTFChainProviderConfig is valid.
func (c CTFChainProviderConfig) validate() error {
	if c.DeployerSignerGen == nil {
		return errors.New("deployer signer generator is required")
	}

	if c.Once == nil {
		return errors.New("sync.Once instance is required")
	}

	return nil
}

var _ chain.Provider = (*CTFChainProvider)(nil)

// CTFChainProvider manages an Aptos chain instance running inside a Chainlink Testing Framework (CTF) Docker container.
//
// This provider requires Docker to be installed and operational. Spinning up a new container can be slow,
// so it is recommended to initialize the provider only once per test suite or parent test to optimize performance.
type CTFChainProvider struct {
	t        *testing.T
	selector uint64
	config   CTFChainProviderConfig

	chain *aptos.Chain
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

// Initialize sets up the Aptos chain by validating the configuration, starting a CTF container,
// generating a deployer signer account, and constructing the chain instance.
func (p *CTFChainProvider) Initialize(ctx context.Context) (chain.BlockChain, error) {
	if p.chain != nil {
		return *p.chain, nil // Already initialized
	}

	if err := p.config.validate(); err != nil {
		return nil, fmt.Errorf("failed to validate provider config: %w", err)
	}

	// Generate the deployer account
	deployerSigner, err := p.config.DeployerSignerGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate deployer account: %w", err)
	}

	// Get the Aptos Chain ID
	chainID, err := chainsel.GetChainIDFromSelector(p.selector)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID from selector %d: %w", p.selector, err)
	}

	// Start the CTF Container
	url, client, err := p.startContainer(ctx, chainID, deployerSigner)
	if err != nil {
		return nil, err
	}

	// Construct the chain
	p.chain = &aptos.Chain{
		Selector:       p.selector,
		Client:         client,
		DeployerSigner: deployerSigner,
		URL:            url,
		Confirm: func(txHash string, opts ...any) error {
			userTx, err := client.WaitForTransaction(txHash, opts...)
			if err != nil {
				return err
			}
			if !userTx.Success {
				return fmt.Errorf("transaction failed: %s", userTx.VmStatus)
			}

			return nil
		},
	}

	return *p.chain, nil
}

// Name returns the name of the CTFChainProvider.
func (*CTFChainProvider) Name() string {
	return "Aptos CTF Chain Provider"
}

// ChainSelector returns the chain selector of the Aptos chain managed by this provider.
func (p *CTFChainProvider) ChainSelector() uint64 {
	return p.selector
}

// BlockChain returns the Aptos chain instance managed by this provider. You must call Initialize
// before using this method to ensure the chain is properly set up.
func (p *CTFChainProvider) BlockChain() chain.BlockChain {
	return *p.chain
}

type aptosContainerResult struct {
	url           string
	containerName string
}

// startContainer starts a CTF container for the Aptos chain with the given chain ID and deployer account.
// It returns the URL of the Aptos node and the client to interact with it.
func (p *CTFChainProvider) startContainer(
	ctx context.Context, chainID string, account *aptoslib.Account,
) (string, *aptoslib.NodeClient, error) {
	const attempts = uint(10)

	// initialize the docker network used by CTF
	err := framework.DefaultNetwork(p.config.Once)
	if err != nil {
		return "", nil, fmt.Errorf("failed to initialize default network: %w", err)
	}

	result, err := retry.DoWithData(func() (aptosContainerResult, error) {
		// reserve all the ports we need explicitly to avoid port conflicts in other tests
		ports := freeport.GetN(p.t, 2)

		input := &blockchain.Input{
			Image:     "", // filled out by defaultAptos function
			Type:      blockchain.TypeAptos,
			ChainID:   chainID,
			PublicKey: account.Address.String(),
			CustomPorts: []string{
				fmt.Sprintf("%d:8080", ports[0]),
				fmt.Sprintf("%d:8081", ports[1]),
			},
		}

		output, rerr := blockchain.NewBlockchainNetwork(input)
		if rerr != nil {
			freeport.Return(ports)

			return aptosContainerResult{}, rerr
		}

		testcontainers.CleanupContainer(p.t, output.Container)

		// Use 127.0.0.1 instead of testcontainers' "localhost" host to avoid IPv6
		// resolution (dial tcp [::1]:port: connection refused) on Linux CI runners.
		url := fmt.Sprintf("http://127.0.0.1:%d/v1", ports[0])

		return aptosContainerResult{
			url:           url,
			containerName: output.ContainerName,
		}, nil
	},
		retry.Context(ctx),
		retry.Attempts(attempts),
		retry.Delay(1*time.Second),
		retry.DelayType(retry.FixedDelay),
		retry.OnRetry(func(attempt uint, err error) {
			p.t.Logf("Attempt %d/%d: Failed to start CTF Aptos container: %v", attempt+1, attempts, err)
		}),
	)
	if err != nil {
		return "", nil, fmt.Errorf("failed to start CTF Aptos container after %d attempts: %w", attempts, err)
	}

	client, err := aptoslib.NewNodeClient(result.url, 0)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create Aptos node client: %w", err)
	}

	if err := waitForAptosReady(ctx, p.t, client); err != nil {
		return "", nil, fmt.Errorf("aptos node health check failed: %w", err)
	}

	dc, err := framework.NewDockerClient()
	if err != nil {
		return "", nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	// incase we didn't use the default account above
	_, err = dc.ExecContainer(result.containerName, []string{
		"aptos", "account", "fund-with-faucet",
		"--account", account.Address.String(),
		"--amount", "100000000000",
	})
	if err != nil {
		return "", nil, fmt.Errorf("failed to fund deployer account: %w", err)
	}

	return result.url, client, nil
}

// waitForAptosReady blocks until the Aptos REST API accepts requests.
func waitForAptosReady(ctx context.Context, t *testing.T, client *aptoslib.NodeClient) error {
	t.Helper()

	const (
		maxAttempts = 30
		retryDelay  = 1 * time.Second
	)

	return retry.Do(func() error {
		_, err := client.GetChainId()
		return err
	},
		retry.Context(ctx),
		retry.Attempts(maxAttempts),
		retry.Delay(retryDelay),
		retry.DelayType(retry.FixedDelay),
		retry.OnRetry(func(attempt uint, err error) {
			t.Logf("Aptos API server not ready yet (attempt %d): %v", attempt+1, err)
		}),
	)
}

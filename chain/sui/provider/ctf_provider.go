package provider

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/block-vision/sui-go-sdk/models"
	sui_sdk "github.com/block-vision/sui-go-sdk/sui"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-testing-framework/framework"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/sui"
)

// CTFChainProviderConfig holds the configuration to initialize the CTFChainProvider.
type CTFChainProviderConfig struct {
	// Required: A generator for the deployer signer account. Use AccountGenPrivateKey to
	// create a deployer signer from a hex private key.
	DeployerSignerGen AccountGenerator

	// Required: A sync.Once instance to ensure that the CTF framework only sets up the new
	// DefaultNetwork once
	Once *sync.Once

	// Optional: A specification of the image to use for the CTF container.
	// Default: mysten/sui-tools:devnet
	Image *string

	// Optional: A specification of the platform to use for the CTF container.
	// Default: linux/amd64
	Platform *string
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

// CTFChainProvider manages a Sui chain instance running inside a Chainlink Testing Framework (CTF) Docker container.
//
// This provider requires Docker to be installed and operational. Spinning up a new container can be slow,
// so it is recommended to initialize the provider only once per test suite or parent test to optimize performance.
type CTFChainProvider struct {
	t        *testing.T
	selector uint64
	config   CTFChainProviderConfig

	chain *sui.Chain
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

// Initialize sets up the Sui chain by validating the configuration, starting a CTF container,
// generating a deployer signer account, and constructing the chain instance.
func (p *CTFChainProvider) Initialize(_ context.Context) (chain.BlockChain, error) {
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

	// Get the Sui Chain ID
	chainID, err := chainsel.GetChainIDFromSelector(p.selector)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID from selector %d: %w", p.selector, err)
	}

	// Start the CTF Container
	url, client := p.startContainer(chainID, deployerSigner)

	// Construct the chain
	p.chain = &sui.Chain{
		ChainMetadata: sui.ChainMetadata{
			Selector: p.selector,
		},
		Client: client,
		Signer: deployerSigner,
		URL:    url,
		// TODO: Implement ConfirmTransaction when available
	}

	return *p.chain, nil
}

// Name returns the name of the CTFChainProvider.
func (*CTFChainProvider) Name() string {
	return "Sui CTF Chain Provider"
}

// ChainSelector returns the chain selector of the Sui chain managed by this provider.
func (p *CTFChainProvider) ChainSelector() uint64 {
	return p.selector
}

// BlockChain returns the Sui chain instance managed by this provider. You must call Initialize
// before using this method to ensure the chain is properly set up.
func (p *CTFChainProvider) BlockChain() chain.BlockChain {
	return *p.chain
}

// startContainer starts a CTF container for the Sui chain with the given chain ID and deployer account.
// It returns the URL of the Sui node and the client to interact with it.
func (p *CTFChainProvider) startContainer(
	chainID string, account sui.SuiSigner,
) (string, sui_sdk.ISuiAPI) {
	var (
		attempts = uint(10)
		url      string
	)

	// initialize the docker network used by CTF
	err := framework.DefaultNetwork(p.config.Once)
	require.NoError(p.t, err)

	// Get address from signer
	address, err := account.GetAddress()
	require.NoError(p.t, err)

	type containerResult struct {
		url           string
		containerName string
	}

	result, err := retry.DoWithData(func() (containerResult, error) {
		image := ""
		platform := ""

		// by default, if image and platform are empty, they are set to amd64 by CTF
		// to support running locally on macos arm64, we set the image and platform to ci-arm64 and linux/arm64 respectively
		if p.config.Image != nil {
			image = *p.config.Image
		} else {
			if runtime.GOARCH == "arm64" {
				image = "mysten/sui-tools:ci-arm64"
			}
		}

		if p.config.Platform != nil {
			platform = *p.config.Platform
		} else {
			if runtime.GOARCH == "arm64" {
				platform = "linux/arm64"
			}
		}

		input := &blockchain.Input{
			Image:         image,
			ImagePlatform: &platform,
			Type:          blockchain.TypeSui,
			ChainID:       chainID,
			PublicKey:     address,
		}

		output, rerr := blockchain.NewBlockchainNetwork(input)
		if rerr != nil {
			// Return the ports to freeport to avoid leaking them during retries

			return containerResult{}, rerr
		}

		testcontainers.CleanupContainer(p.t, output.Container)

		return containerResult{
			url:           output.Nodes[0].ExternalHTTPUrl,
			containerName: output.ContainerName,
		}, nil
	},
		retry.Context(p.t.Context()),
		retry.Attempts(attempts),
		retry.Delay(1*time.Second),
		retry.DelayType(retry.FixedDelay),
		retry.OnRetry(func(attempt uint, err error) {
			p.t.Logf("Attempt %d/%d: Failed to start CTF Sui container: %v", attempt+1, attempts, err)
		}),
	)
	require.NoError(p.t, err, "Failed to start CTF Sui container after %d attempts", attempts)

	url = result.url

	client := sui_sdk.NewSuiClient(url)

	var ready bool
	for i := range 30 {
		time.Sleep(time.Second)
		// TODO: Add appropriate readiness check when available
		p.t.Logf("Sui client ready check (attempt %d)\n", i+1)
		ready = true

		break
	}
	require.True(p.t, ready, "Sui network not ready")

	err = fundAccount(fmt.Sprintf("http://%s:%s", "127.0.0.1", "9123"), address)
	require.NoError(p.t, err)

	return url, client
}

func fundAccount(url string, address string) error {
	r := resty.New().SetBaseURL(url)
	b := &models.FaucetRequest{
		FixedAmountRequest: &models.FaucetFixedAmountRequest{
			Recipient: address,
		},
	}
	resp, err := r.R().SetBody(b).SetHeader("Content-Type", "application/json").Post("/gas")
	if err != nil {
		return err
	}
	framework.L.Info().Any("Resp", resp).Msg("Address is funded!")
	return nil
}

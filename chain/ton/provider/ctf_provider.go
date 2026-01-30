package provider

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/testcontainers/testcontainers-go"

	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-testing-framework/framework"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/smartcontractkit/freeport"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	cldf_ton "github.com/smartcontractkit/chainlink-deployments-framework/chain/ton"
)

const (
	// defaultTONImage is the default Docker image used for TON localnet.
	// Only images from this repository are supported.
	defaultTONImage = "ghcr.io/neodix42/mylocalton-docker:v3.7"

	// supportedTONImageRepository is the only supported Docker image repository for TON localnet.
	supportedTONImageRepository = "ghcr.io/neodix42/mylocalton-docker"
)

// CTFChainProviderConfig holds the configuration to initialize the CTFChainProvider.
type CTFChainProviderConfig struct {
	// Required: A sync.Once instance to ensure that the CTF framework only sets up the new
	// DefaultNetwork once
	Once *sync.Once

	// Optional: Docker image to use for the TON localnet. If empty, defaults to defaultTONImage.
	// Note: Only images from supportedTONImageRepository are supported.
	Image string

	// Optional: Custom environment variables to pass to the TON container.
	// Example: map[string]string{"NEXT_BLOCK_GENERATION_DELAY": "0.5"}
	CustomEnv map[string]string

	// Optional: Port to expose the TON container on. If 0 or not specified, a free port will be
	// automatically selected using freeport.
	Port int
}

// validate checks if the CTFChainProviderConfig is valid.
func (c CTFChainProviderConfig) validate() error {
	if c.Once == nil {
		return errors.New("sync.Once instance is required")
	}

	if c.Image != "" && !strings.HasPrefix(c.Image, supportedTONImageRepository) {
		return fmt.Errorf("unsupported image %q: must be from %s", c.Image, supportedTONImageRepository)
	}

	return nil
}

var _ chain.Provider = (*CTFChainProvider)(nil)

// CTFChainProvider manages a Ton chain instance running inside a Chainlink Testing Framework (CTF) Docker container.
//
// This provider requires Docker to be installed and operational. Spinning up a new container can be slow,
// so it is recommended to initialize the provider only once per test suite or parent test to optimize performance.
type CTFChainProvider struct {
	t        *testing.T
	selector uint64
	config   CTFChainProviderConfig

	chain *cldf_ton.Chain
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

// Initialize sets up the Ton chain by validating the configuration, starting a CTF container,
// generating a deployer signer account, and constructing the chain instance.
func (p *CTFChainProvider) Initialize(ctx context.Context) (chain.BlockChain, error) {
	if p.chain != nil {
		return *p.chain, nil // Already initialized
	}

	if err := p.config.validate(); err != nil {
		return nil, fmt.Errorf("failed to validate provider config: %w", err)
	}

	// Get the Chain ID
	chainID, err := chainsel.GetChainIDFromSelector(p.selector)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID from selector: %w", err)
	}

	url, nodeClient, err := p.startContainer(ctx, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// mylocalton uses a global_id of -217 by default
	// https://github.com/neodix42/mylocalton-docker/blob/8f9c6ea27cd608dc6370c4191554b42b5a797905/docker/scripts/start-genesis.sh#L62
	tonWallet, err := createTonWallet(nodeClient, wallet.ConfigV5R1Final{NetworkGlobalID: -217}, wallet.WithWorkchain(0))
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	p.chain = &cldf_ton.Chain{
		ChainMetadata: cldf_ton.ChainMetadata{Selector: p.selector},
		Client:        nodeClient,
		Wallet:        tonWallet,
		WalletAddress: tonWallet.WalletAddress(),
		URL:           url,
	}

	return *p.chain, nil
}

func (p *CTFChainProvider) startContainer(ctx context.Context, chainID string) (string, *ton.APIClient, error) {
	var (
		attempts = uint(10)
		url      string
	)

	// initialize the docker network used by CTF
	err := framework.DefaultNetwork(p.config.Once)
	if err != nil {
		return "", nil, fmt.Errorf("failed to initialize default network: %w", err)
	}

	url, err = retry.DoWithData(func() (string, error) {
		port, usedFreeport := p.getPort()

		// spin up mylocalton with CTFv2
		output, rerr := blockchain.NewBlockchainNetwork(&blockchain.Input{
			Type:      blockchain.TypeTon,
			ChainID:   chainID,
			Port:      strconv.Itoa(port),
			Image:     p.getImage(),
			CustomEnv: p.config.CustomEnv,
		})
		if rerr != nil {
			// Return the ports to freeport to avoid leaking them during retries
			// Only return if we obtained the port from freeport
			if usedFreeport {
				freeport.Return([]int{port})
			}

			return "", rerr
		}

		testcontainers.CleanupContainer(p.t, output.Container)

		return output.Nodes[0].ExternalHTTPUrl, nil
	},
		retry.Context(ctx),
		retry.Attempts(attempts),
		retry.Delay(1*time.Second),
		retry.DelayType(retry.FixedDelay),
		retry.OnRetry(func(attempt uint, err error) {
			p.t.Logf("Attempt %d/%d: Failed to start CTF Ton container: %v", attempt+1, attempts, err)
		}),
	)
	if err != nil {
		return "", nil, fmt.Errorf("failed to start CTF Ton container after %d attempts: %w", attempts, err)
	}

	connectionPool, err := createLiteclientConnectionPool(ctx, url)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create liteclient connection pool: %w", err)
	}

	client := ton.NewAPIClient(connectionPool, ton.ProofCheckPolicyFast)

	// check connection, CTFv2 handles the readiness
	mb, err := getMasterchainBlockID(ctx, client)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get masterchain block ID: %w", err)
	}

	// set starting point to verify master block proofs chain
	client.SetTrustedBlock(mb)

	return url, client, nil
}

// Note: this utility functions can be replaced once we have in the chainlink-ton utils package
func createTonWallet(client ton.APIClientWrapped, versionConfig wallet.VersionConfig, option wallet.Option) (*wallet.Wallet, error) {
	seed := wallet.NewSeed()
	rw, err := wallet.FromSeed(client, seed, versionConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet from seed: %w", err)
	}
	pw, perr := wallet.FromPrivateKeyWithOptions(client, rw.PrivateKey(), versionConfig, option)
	if perr != nil {
		return nil, fmt.Errorf("failed to create wallet from private key: %w", perr)
	}

	return pw, nil
}

func getMasterchainBlockID(ctx context.Context, client ton.APIClientWrapped) (*ton.BlockIDExt, error) {
	var masterchainBlockID *ton.BlockIDExt
	// check connection, CTFv2 handles the readiness
	err := retry.Do(func() error {
		var err error
		masterchainBlockID, err = client.GetMasterchainInfo(ctx)

		return err
	},
		retry.Context(ctx),
		retry.Attempts(30),
		retry.Delay(1*time.Second),
		retry.DelayType(retry.FixedDelay),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get masterchain info: %w", err)
	}

	// return masterchain block for setting trusted block
	return masterchainBlockID, nil
}

// Name returns the name of the CTFChainProvider.
func (*CTFChainProvider) Name() string {
	return "TON CTF Chain Provider"
}

// ChainSelector returns the chain selector of the TON chain managed by this provider.
func (p *CTFChainProvider) ChainSelector() uint64 {
	return p.selector
}

// BlockChain returns the Ton chain instance managed by this provider. You must call Initialize
// before using this method to ensure the chain is properly set up.
func (p *CTFChainProvider) BlockChain() chain.BlockChain {
	return *p.chain
}

// getImage returns the configured Docker image, or the default if not specified.
func (p *CTFChainProvider) getImage() string {
	if p.config.Image != "" {
		return p.config.Image
	}

	return defaultTONImage
}

// getPort returns the configured port if specified, otherwise gets a free port using freeport.
// The second return value indicates whether the port was obtained from freeport.
func (p *CTFChainProvider) getPort() (port int, usedFreeport bool) {
	if p.config.Port != 0 {
		return p.config.Port, false
	}

	return freeport.GetOne(p.t), true
}

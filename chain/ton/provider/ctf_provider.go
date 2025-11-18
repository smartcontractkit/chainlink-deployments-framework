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
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
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
	// DefaultTONImage is the default Docker image used for TON localnet.
	// Only images from this repository are supported.
	DefaultTONImage = "ghcr.io/neodix42/mylocalton-docker:v3.7"

	// SupportedTONImageRepository is the only supported Docker image repository for TON localnet.
	SupportedTONImageRepository = "ghcr.io/neodix42/mylocalton-docker"
)

// CTFChainProviderConfig holds the configuration to initialize the CTFChainProvider.
type CTFChainProviderConfig struct {
	// Required: A sync.Once instance to ensure that the CTF framework only sets up the new
	// DefaultNetwork once
	Once *sync.Once

	// Optional: Docker image to use for the TON localnet. If empty, defaults to DefaultTONImage.
	// Note: Only images from SupportedTONImageRepository are supported.
	Image string

	// Optional: Retry count for APIClient. Default is 0 (unlimited retries).
	// Set to positive value for specific retry count.
	RetryCount int

	// Optional: Custom environment variables to pass to the TON container.
	// Example: map[string]string{"NEXT_BLOCK_GENERATION_DELAY": "0.5"}
	CustomEnv map[string]string
}

// validate checks if the CTFChainProviderConfig is valid.
func (c CTFChainProviderConfig) validate() error {
	if c.Once == nil {
		return errors.New("sync.Once instance is required")
	}

	if c.Image != "" && !strings.Contains(c.Image, SupportedTONImageRepository) {
		return fmt.Errorf("unsupported image %q: must be from %s", c.Image, SupportedTONImageRepository)
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
func (p *CTFChainProvider) Initialize(_ context.Context) (chain.BlockChain, error) {
	if p.chain != nil {
		return *p.chain, nil // Already initialized
	}

	if err := p.config.validate(); err != nil {
		return nil, fmt.Errorf("failed to validate provider config: %w", err)
	}

	// Get the Chain ID
	chainID, err := chainsel.GetChainIDFromSelector(p.selector)
	require.NoError(p.t, err, "failed to get chain ID from selector")

	url, nodeClient := p.startContainer(chainID)
	// mylocalton uses a global_id of -217 by default
	// https://github.com/neodix42/mylocalton-docker/blob/8f9c6ea27cd608dc6370c4191554b42b5a797905/docker/scripts/start-genesis.sh#L62
	tonWallet := createTonWallet(p.t, nodeClient, wallet.ConfigV5R1Final{NetworkGlobalID: -217}, wallet.WithWorkchain(0))
	// airdrop the deployer wallet
	fundTonWallets(p.t, nodeClient, []*address.Address{tonWallet.Address()}, []tlb.Coins{tlb.MustFromTON("1000")})
	p.chain = &cldf_ton.Chain{
		ChainMetadata: cldf_ton.ChainMetadata{Selector: p.selector},
		Client:        nodeClient,
		Wallet:        tonWallet,
		WalletAddress: tonWallet.Address(),
		URL:           url,
	}

	return *p.chain, nil
}

func (p *CTFChainProvider) startContainer(chainID string) (string, ton.APIClientWrapped) {
	var (
		attempts = uint(10)
		url      string
	)

	// initialize the docker network used by CTF
	err := framework.DefaultNetwork(p.config.Once)
	require.NoError(p.t, err)

	url, err = retry.DoWithData(func() (string, error) {
		// Initialize a port for the container
		port := freeport.GetOne(p.t)

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
			freeport.Return([]int{port})

			return "", rerr
		}

		testcontainers.CleanupContainer(p.t, output.Container)

		return output.Nodes[0].ExternalHTTPUrl, nil
	},
		retry.Context(p.t.Context()),
		retry.Attempts(attempts),
		retry.Delay(1*time.Second),
		retry.DelayType(retry.FixedDelay),
		retry.OnRetry(func(attempt uint, err error) {
			p.t.Logf("Attempt %d/%d: Failed to start CTF Ton container: %v", attempt+1, attempts, err)
		}),
	)
	require.NoError(p.t, err, "Failed to start CTF Ton container after %d attempts", attempts)

	connectionPool, err := createLiteclientConnectionPool(p.t.Context(), url)
	require.NoError(p.t, err)

	client := ton.NewAPIClient(connectionPool, ton.ProofCheckPolicyFast)

	// check connection, CTFv2 handles the readiness
	mb := getMasterchainBlockID(p.t, client)
	// set starting point to verify master block proofs chain
	client.SetTrustedBlock(mb)

	retryCount := p.getRetryCount()

	return url, client.WithRetry(retryCount)
}

// Note: this utility functions can be replaced once we have in the chainlink-ton utils package
func createTonWallet(t *testing.T, client ton.APIClientWrapped, versionConfig wallet.VersionConfig, option wallet.Option) *wallet.Wallet {
	t.Helper()

	seed := wallet.NewSeed()
	rw, err := wallet.FromSeed(client, seed, versionConfig)
	require.NoError(t, err)
	pw, perr := wallet.FromPrivateKeyWithOptions(client, rw.PrivateKey(), versionConfig, option)
	require.NoError(t, perr)

	return pw
}

func fundTonWallets(t *testing.T, client ton.APIClientWrapped, recipients []*address.Address, amounts []tlb.Coins) {
	t.Helper()

	require.Len(t, amounts, len(recipients), "recipients and amounts must have the same length")
	// initialize the prefunded wallet(Highload-V2), for other wallets, see https://github.com/neodix42/mylocalton-docker#pre-installed-wallets
	version := wallet.HighloadV2Verified //nolint:staticcheck // SA1019: only available option in mylocalton-docker
	rawHlWallet, err := wallet.FromSeed(client, strings.Fields(blockchain.DefaultTonHlWalletMnemonic), version)
	require.NoError(t, err)

	mcFunderWallet, err := wallet.FromPrivateKeyWithOptions(client, rawHlWallet.PrivateKey(), version, wallet.WithWorkchain(-1))
	require.NoError(t, err)

	funder, err := mcFunderWallet.GetSubwallet(uint32(42))
	require.NoError(t, err)
	// double check funder address
	require.Equal(t, blockchain.DefaultTonHlWalletAddress, funder.Address().StringRaw(), "funder address mismatch")
	// create transfer messages for each recipient
	messages := make([]*wallet.Message, len(recipients))
	for i, addr := range recipients {
		transfer, terr := funder.BuildTransfer(addr, amounts[i], false, "")
		require.NoError(t, terr)
		messages[i] = transfer
	}
	_, _, txerr := funder.SendManyWaitTransaction(t.Context(), messages)
	require.NoError(t, txerr, "airdrop transaction failed")
	// we don't wait for the transaction to be confirmed here, as it may take some time
}

func getMasterchainBlockID(t *testing.T, client *ton.APIClient) *ton.BlockIDExt {
	t.Helper()

	var masterchainBlockID *ton.BlockIDExt
	// check connection, CTFv2 handles the readiness
	err := retry.Do(func() error {
		var err error
		masterchainBlockID, err = client.GetMasterchainInfo(t.Context())

		return err
	},
		retry.Context(t.Context()),
		retry.Attempts(30),
		retry.Delay(1*time.Second),
		retry.DelayType(retry.FixedDelay),
	)
	require.NoError(t, err, "TON network not ready")

	// return masterchain block for setting trusted block
	return masterchainBlockID
}

// Name returns the name of the CTFChainProvider.
func (*CTFChainProvider) Name() string {
	return "Ton CTF Chain Provider"
}

// ChainSelector returns the chain selector of the Aptos chain managed by this provider.
func (p *CTFChainProvider) ChainSelector() uint64 {
	return p.selector
}

// BlockChain returns the Ton chain instance managed by this provider. You must call Initialize
// before using this method to ensure the chain is properly set up.
func (p *CTFChainProvider) BlockChain() chain.BlockChain {
	return *p.chain
}

func (p *CTFChainProvider) getRetryCount() int {
	return p.config.RetryCount
}

// getImage returns the configured Docker image, or the default if not specified.
func (p *CTFChainProvider) getImage() string {
	if p.config.Image != "" {
		return p.config.Image
	}

	return DefaultTONImage
}

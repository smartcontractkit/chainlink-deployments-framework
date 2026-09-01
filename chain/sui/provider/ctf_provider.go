package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/block-vision/sui-go-sdk/models"
	"github.com/go-resty/resty/v2"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	cslclient "github.com/smartcontractkit/chainlink-sui/relayer/client"
	"github.com/smartcontractkit/chainlink-testing-framework/framework"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/smartcontractkit/freeport"
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
	// Default: unset, which uses CTF blockchain defaults (blockchain.DefaultSuiImage /
	// blockchain.DefaultSuiImageARM64 from chainlink-testing-framework for the host GOARCH).
	Image *string

	// Optional: OCI platform for the CTF image (ImagePlatform).
	// Default: unset; on arm64 hosts this provider sets linux/arm64, otherwise CTF defaults apply.
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
	url, faucetUrl, client := p.startContainer(chainID, deployerSigner)

	// Construct the chain
	p.chain = &sui.Chain{
		ChainMetadata: sui.ChainMetadata{
			Selector: p.selector,
		},
		Client:    client,
		Signer:    deployerSigner,
		URL:       url,
		FaucetURL: faucetUrl,
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
) (string, string, cslclient.SuiPTBClient) {
	var (
		attempts  = uint(10)
		url       string
		fauceturl string
	)

	// initialize the docker network used by CTF
	err := framework.DefaultNetwork(p.config.Once)
	require.NoError(p.t, err)

	// Get address from signer
	address, err := account.GetAddress()
	require.NoError(p.t, err)

	type containerResult struct {
		url           string
		faucetPort    string
		containerName string
	}

	result, err := retry.DoWithData(func() (containerResult, error) {
		ports := freeport.GetN(p.t, 2)
		port := ports[0]
		faucetPort := ports[1]

		// Image: when unset, blockchain.NewBlockchainNetwork applies CTF defaults
		// (DefaultSuiImage / DefaultSuiImageARM64) inside defaultSui.
		input := &blockchain.Input{
			Type:       blockchain.TypeSui,
			ChainID:    chainID,
			PublicKey:  address,
			Port:       strconv.Itoa(port),
			FaucetPort: strconv.Itoa(faucetPort),
		}
		if p.config.Image != nil {
			input.Image = *p.config.Image
		}
		if p.config.Platform != nil {
			input.ImagePlatform = p.config.Platform
		} else if runtime.GOARCH == "arm64" {
			// Mysten arm64 images need an explicit platform; CTF only sets this when Image is empty.
			arm := "linux/arm64"
			input.ImagePlatform = &arm
		}

		output, rerr := blockchain.NewBlockchainNetwork(input)
		if rerr != nil {
			// Return the ports to freeport to avoid leaking them during retries
			freeport.Return([]int{port, faucetPort})

			return containerResult{}, rerr
		}

		testcontainers.CleanupContainer(p.t, output.Container)

		return containerResult{
			url:           output.Nodes[0].ExternalHTTPUrl,
			faucetPort:    input.FaucetPort,
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
	fauceturl = fmt.Sprintf("http://%s:%s", "127.0.0.1", result.faucetPort)

	log, logErr := logger.New()
	require.NoError(p.t, logErr, "Failed to create logger")

	client, clientErr := sui.NewPTBClientFromNodeURL(log, url, "")
	require.NoError(p.t, clientErr, "Failed to create Sui PTB client")

	var ready bool
	for i := range 30 {
		time.Sleep(time.Second)
		// TODO: Add appropriate readiness check when available
		p.t.Logf("Sui client ready check (attempt %d)\n", i+1)
		ready = true

		break
	}
	require.True(p.t, ready, "Sui network not ready")

	err = fundAccount(p.t.Context(), fauceturl, address)
	require.NoError(p.t, err)

	return url, fauceturl, client
}

// faucetFundTimeout bounds the total time spent retrying Sui faucet /gas
// funding. Without it, a faucet endpoint that hangs at the TCP/HTTP layer can
// block each attempt indefinitely and significantly prolong test teardown. The
// context carrying this deadline is attached to both retry-go and the Resty
// request so an in-flight hang is cancelled as soon as the budget expires.
const faucetFundTimeout = 2 * time.Minute

// faucetRequestTimeout bounds a single faucet /gas HTTP request so that one
// hung attempt does not consume the entire faucetFundTimeout budget, leaving
// room for subsequent retries.
const faucetRequestTimeout = 10 * time.Second

// faucetFundAttempts is the maximum number of /gas funding attempts before
// giving up. The faucetFundTimeout context still bounds the wall-clock total.
const faucetFundAttempts = uint(15)

func fundAccount(ctx context.Context, url string, address string) error {
	// Bound the overall retry. A child of the caller's context wins on the
	// earlier deadline, so callers (tests) can tighten it further.
	ctx, cancel := context.WithTimeout(ctx, faucetFundTimeout)
	defer cancel()

	r := resty.New().
		SetBaseURL(url).
		SetTimeout(faucetRequestTimeout)

	b := &models.FaucetRequest{
		FixedAmountRequest: &models.FaucetFixedAmountRequest{
			Recipient: address,
		},
	}
	// The Sui faucet is served by the same container that just started, so the first
	// /gas request can race the faucet's own readiness and fail with a connection
	// reset. Retry with backoff until the faucet accepts the request, treating a
	// successful POST (non-error HTTP response) as readiness. RetryIf classifies
	// errors so transient failures are retried while failures that retrying cannot
	// fix stop immediately instead of burning the budget.
	_, err := retry.DoWithData(func() (*resty.Response, error) {
		resp, perr := r.R().
			SetContext(ctx).
			SetBody(b).
			SetHeader("Content-Type", "application/json").
			Post("/gas")
		if perr != nil {
			return nil, perr
		}
		if resp.IsError() {
			return nil, &faucetStatusError{status: resp.StatusCode(), body: resp.Body()}
		}

		return resp, nil
	},
		retry.Context(ctx),
		retry.Attempts(faucetFundAttempts),
		retry.Delay(time.Second),
		retry.DelayType(retry.BackOffDelay),
		retry.RetryIf(isRetryableFaucetErr),
		retry.LastErrorOnly(true),
		retry.OnRetry(func(n uint, err error) {
			framework.L.Warn().Err(err).Uint("attempt", n+1).Uint("attempts", faucetFundAttempts).
				Str("recipient", address).Msg("Retrying Sui faucet /gas funding")
		}),
	)
	if err != nil {
		return fmt.Errorf("fund account via Sui faucet: %w", err)
	}
	framework.L.Info().Str("recipient", address).Msg("Address is funded!")

	return nil
}

// faucetStatusError carries the HTTP status and body of a non-2xx faucet
// response so callers can inspect why funding failed without re-issuing the
// request, and so isRetryableFaucetErr can classify it.
type faucetStatusError struct {
	status int
	body   []byte
}

func (e *faucetStatusError) Error() string {
	if trimmed := bytes.TrimSpace(e.body); len(trimmed) > 0 {
		return fmt.Sprintf("faucet returned status %d: %s", e.status, trimmed)
	}

	return fmt.Sprintf("faucet returned status %d", e.status)
}

// isRetryableFaucetErr classifies faucet /gas errors. Transient failures (faucet
// still warming up, brief network blips, rate limiting, 5xx) are retried; failures
// that retrying cannot fix (a malformed request, an already-cancelled context)
// stop immediately so they don't burn the retry budget and prolong teardown.
func isRetryableFaucetErr(err error) bool {
	if err == nil {
		return false
	}
	// A cancelled/deadline-exceeded context means the caller (or the total
	// budget) has given up; never retry, just propagate.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var se *faucetStatusError
	if errors.As(err, &se) {
		switch {
		case se.status == http.StatusTooManyRequests:
			// Rate limited; back off and try again.
			return true
		case se.status >= 400 && se.status < 500:
			// Client-side error (bad recipient, unauthorized, not found, ...).
			// Retrying an identical request won't fix it.
			return false
		case se.status >= 500:
			// Server-side error / faucet not yet ready.
			return true
		}
	}
	// Transport-level failures (connection refused/reset, DNS, timeouts) are
	// treated as transient while the faucet container comes up.
	return true
}

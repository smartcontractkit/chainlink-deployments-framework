package provider

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/avast/retry-go/v4"
	sollib "github.com/gagliardetto/solana-go"
	solrpc "github.com/gagliardetto/solana-go/rpc"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
	solCommonUtil "github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/common"
	"github.com/smartcontractkit/chainlink-testing-framework/framework"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/smartcontractkit/freeport"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana/provider/rpcclient"
)

// CTFChainProviderConfig holds the configuration to initialize the CTFChainProvider.
type CTFChainProviderConfig struct {
	// Required: A generator for the deployer key. Use PrivateKeyFromRaw to create a deployer
	// key from a private key.
	DeployerKeyGen PrivateKeyGenerator
	// Required: The absolute path to the directory containing the Solana CLI binaries.
	ProgramsPath string
	// Required: A map of program names to their program IDs. You may set this as an empty map if
	// you do not have any programs to deploy.
	ProgramIDs map[string]string
	// Required: A sync.Once instance to ensure that the CTF framework only sets up the new
	// DefaultNetwork once
	Once *sync.Once
	// Optional: WaitDelayAfterContainerStart is the duration to wait after starting the CTF
	// container. This is useful to ensure the container is fully initialized before attempting to
	// interact with it.
	//
	// Default: 0s (no delay)
	WaitDelayAfterContainerStart time.Duration
}

// validate checks if the RPCChainProviderConfig is valid.
func (c CTFChainProviderConfig) validate() error {
	if c.DeployerKeyGen == nil {
		return errors.New("deployer key generator is required")
	}
	if c.ProgramsPath == "" {
		return errors.New("programs path is required")
	}
	if c.ProgramIDs == nil {
		return errors.New("program ids is required")
	}
	if err := isValidFilepath(c.ProgramsPath); err != nil {
		return err
	}

	return nil
}

var _ chain.Provider = (*CTFChainProvider)(nil)

// CTFChainProvider manages an Solana chain instance running inside a Chainlink Testing Framework
// (CTF) Docker container.
//
// This provider requires Docker to be installed and operational. Spinning up a new container
// can be slow, so it is recommended to initialize the provider only once per test suite or parent
// test to optimize performance.
type CTFChainProvider struct {
	t        *testing.T
	selector uint64
	config   CTFChainProviderConfig

	chain *solana.Chain
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

// Initialize sets up the Solana chain by validating the configuration, starting a CTF container,
// generating a deployer key, and constructing the chain instance.
func (p *CTFChainProvider) Initialize(_ context.Context) (chain.BlockChain, error) {
	if p.chain != nil {
		return *p.chain, nil // Already initialized
	}

	if err := p.config.validate(); err != nil {
		return nil, fmt.Errorf("failed to validate provider config: %w", err)
	}

	// Get the Solana Chain ID
	chainID, err := chain_selectors.GetChainIDFromSelector(p.selector)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID from selector %d: %w", p.selector, err)
	}

	// Generate the deployer keypair
	privKey, err := p.config.DeployerKeyGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate deployer keypair: %w", err)
	}

	// Persist the deployer keypair to a temporary for the Solana CLI to use
	keypairDir := p.t.TempDir()

	keypairPath := filepath.Join(keypairDir, "solana-keypair.json")
	if err = writePrivateKeyToPath(keypairPath, privKey); err != nil {
		return nil, fmt.Errorf("failed to write deployer keypair to file: %w", err)
	}

	// Start the CTF Container
	httpURL, wsURL := p.startContainer(chainID, privKey.PublicKey())

	// Initialize the Solana client with the container HTTP URL
	client := rpcclient.New(solrpc.New(httpURL), privKey)

	// Initialize the Solana chain instance with the provided configuration
	p.chain = &solana.Chain{
		Selector:     p.selector,
		Client:       client.Client,
		URL:          httpURL,
		WSURL:        wsURL,
		DeployerKey:  &privKey,
		ProgramsPath: p.config.ProgramsPath,
		KeypairPath:  keypairPath,
		SendAndConfirm: func(
			ctx context.Context, instructions []sollib.Instruction, txMods ...rpcclient.TxModifier,
		) error {
			_, err := client.SendAndConfirmTx(ctx, instructions,
				rpcclient.WithTxModifiers(txMods...),
				rpcclient.WithRetry(500, 50*time.Millisecond),
			)

			return err
		},
		Confirm: func(instructions []sollib.Instruction, opts ...solCommonUtil.TxModifier) error {
			emptyLookupTables := map[sollib.PublicKey]sollib.PublicKeySlice{}
			_, err := solCommonUtil.SendAndConfirmWithLookupTablesAndRetries(
				context.Background(),
				client.Client,
				instructions,
				privKey,
				solrpc.CommitmentConfirmed,
				emptyLookupTables,
				opts...,
			)

			return err
		},
	}

	return *p.chain, nil
}

// Name returns the name of the CTFChainProvider.
func (*CTFChainProvider) Name() string {
	return "Solana CTF Chain Provider"
}

// ChainSelector returns the chain selector of the Solana chain managed by this provider.
func (p *CTFChainProvider) ChainSelector() uint64 {
	return p.selector
}

// BlockChain returns the Solana chain instance managed by this provider. You must call Initialize
// before using this method to ensure the chain is properly set up.
func (p *CTFChainProvider) BlockChain() chain.BlockChain {
	return p.chain
}

// startContainer starts a CTF container for the Solana chain.
func (p *CTFChainProvider) startContainer(
	chainID string,
	adminPubKey sollib.PublicKey,
) (string, string) {
	var (
		attempts       = uint(10)
		httpURL, wsURL string
	)

	// initialize the docker network used by CTF
	err := framework.DefaultNetwork(p.config.Once)
	require.NoError(p.t, err)

	err = retry.Do(func() error {
		// solana requires 2 ports, one for http and one for ws, but only allows one to be specified
		// the other is +1 of the first one
		// must reserve 2 to avoid port conflicts in the freeport library with other tests
		// https://github.com/smartcontractkit/chainlink-testing-framework/blob/e109695d311e6ed42ca3194907571ce6454fae8d/framework/components/blockchain/blockchain.go#L39
		ports := freeport.GetN(p.t, 2)

		image := ""
		if runtime.GOOS == "linux" {
			image = "solanalabs/solana:v1.18.26" // workaround on linux to load a separate image
		}

		input := &blockchain.Input{
			Image:          image,
			Type:           "solana",
			ChainID:        chainID,
			PublicKey:      adminPubKey.String(),
			Port:           strconv.Itoa(ports[0]),
			ContractsDir:   p.config.ProgramsPath, // Programs are contracts in the context of CTF
			SolanaPrograms: p.config.ProgramIDs,
		}

		output, rerr := blockchain.NewBlockchainNetwork(input)
		if rerr != nil {
			// Return the ports to freeport to avoid leaking them during retries
			freeport.Return(ports)

			return rerr
		}

		testcontainers.CleanupContainer(p.t, output.Container)

		httpURL = output.Nodes[0].ExternalHTTPUrl
		wsURL = output.Nodes[0].ExternalWSUrl

		return nil
	},
		retry.Context(p.t.Context()),
		retry.Attempts(attempts),
		retry.Delay(1*time.Second),
		retry.DelayType(retry.FixedDelay),
	)
	require.NoError(p.t, err, "Failed to start CTF Solana container after %d attempts", attempts)

	checkSolanaNodeHealth(p.t, httpURL)

	// Wait for the configured delay after starting the container to ensure the chain is fully booted.
	if p.config.WaitDelayAfterContainerStart > 0 {
		time.Sleep(p.config.WaitDelayAfterContainerStart)
	}

	return httpURL, wsURL
}

// checkSolanaNodeHealth checks the health of the Solana node by querying its health endpoint.
// We expect that node will be available within 30 seconds, with a 1 second delay between attempts,
// however this is an assumption.
func checkSolanaNodeHealth(t *testing.T, httpURL string) {
	t.Helper()

	solclient := solrpc.New(httpURL)
	err := retry.Do(func() error {
		out, rerr := solclient.GetHealth(t.Context())
		if rerr != nil {
			return rerr
		}
		if out != solrpc.HealthOk {
			return fmt.Errorf("API server not healthy yet: %s", out)
		}

		return nil
	},
		retry.Context(t.Context()),
		retry.Attempts(30),
		retry.Delay(1*time.Second),
		retry.DelayType(retry.FixedDelay),
	)
	require.NoError(t, err, "API server is not healthy")
}

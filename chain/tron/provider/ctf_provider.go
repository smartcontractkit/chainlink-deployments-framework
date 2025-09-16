package provider

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/http/common"
	"github.com/fbsobreira/gotron-sdk/pkg/http/soliditynode"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-testing-framework/framework"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/smartcontractkit/chainlink-tron/relayer/sdk"
	"github.com/smartcontractkit/freeport"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron/provider/rpcclient"
)

// CTFChainProviderConfig holds the configuration to initialize the CTFChainProvider.
type CTFChainProviderConfig struct {
	// Required: A generator for the deployer signer. Use SignerGenCTFDefault to
	// create a deployer signer from the default CTF account. Alternatively, you can use
	// SignerRandom to create a new random signer.
	DeployerSignerGen SignerGenerator

	// Required: A sync.Once instance to ensure that the CTF framework only sets up the new
	// DefaultNetwork once
	Once *sync.Once
}

// validate checks whether the configuration contains all required values.
func (c CTFChainProviderConfig) validate() error {
	if c.DeployerSignerGen == nil {
		return errors.New("deployer signer generator is required")
	}
	if c.Once == nil {
		return errors.New("sync.Once instance is required")
	}

	return nil
}

// Ensure interface implementation
var _ chain.Provider = (*CTFChainProvider)(nil)

// CTFChainProvider manages a TRON chain instance running inside a Chainlink Testing Framework (CTF) Docker container.
//
// This provider requires Docker to be installed and operational. Spinning up a new container can be slow,
// so it is recommended to initialize the provider only once per test suite or parent test to optimize performance.
type CTFChainProvider struct {
	t        *testing.T             // Test context for logging and assertions
	selector uint64                 // Unique chain selector identifier.
	config   CTFChainProviderConfig // Configuration used to set up the provider.

	chain *tron.Chain // Cached reference to the initialized Tron chain instance.
}

// NewCTFChainProvider creates a new CTFChainProvider with the given selector and configuration.
// The actual connection is deferred until Initialize is called.
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

// Initialize sets up the TRON chain by validating the configuration, starting a CTF container,
// generating a deployer account, and constructing the chain instance.
func (p *CTFChainProvider) Initialize(_ context.Context) (chain.BlockChain, error) {
	// If already initialized, return cached chain
	if p.chain != nil {
		return *p.chain, nil
	}

	// Validate config
	if err := p.config.validate(); err != nil {
		return nil, fmt.Errorf("failed to validate provider config: %w", err)
	}

	// Get the TRON Chain ID
	chainID, err := chainsel.GetChainIDFromSelector(p.selector)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID from selector %d: %w", p.selector, err)
	}

	// Start the CTF container and get the full node and solidity node URLs
	fullNodeURL, solidityNodeURL := p.startContainer(chainID)

	// Parse URLs for node connections
	fullNodeUrlObj, err := url.Parse(fullNodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse full node URL: %w", err)
	}
	solidityNodeUrlObj, err := url.Parse(solidityNodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse solidity node URL: %w", err)
	}

	// Create a client that wraps both full node and solidity node connections
	combinedClient, err := sdk.CreateCombinedClient(fullNodeUrlObj, solidityNodeUrlObj)
	if err != nil {
		return nil, fmt.Errorf("failed to create combined client: %w", err)
	}

	// Construct and cache the Tron chain instance with helper methods for deploying and interacting with contracts
	chain, err := GetTronChain(p.selector, combinedClient, p.config.DeployerSignerGen)
	if err != nil {
		return nil, fmt.Errorf("failed to get tron chain: %w", err)
	}
	p.chain = &chain

	return *p.chain, nil
}

// GetTronChain creates and returns a tron.Chain instance with the provided parameters.
func GetTronChain(selector uint64, combinedClient sdk.CombinedClient, signerGen SignerGenerator) (tron.Chain, error) {
	deployerAddr, err := signerGen.GetAddress()
	if err != nil {
		return tron.Chain{}, fmt.Errorf("failed to get deployer address: %w", err)
	}

	fullNodeURL := combinedClient.FullNodeClient().BaseURL

	client := rpcclient.New(combinedClient, signerGen.Sign)

	return tron.Chain{
		ChainMetadata: tron.ChainMetadata{
			Selector: selector,
		},
		Client:   combinedClient, // Underlying client for Tron node communication
		SignHash: signerGen.Sign, // Function for signing transactions
		Address:  deployerAddr,   // Default "from" address for transactions
		URL:      fullNodeURL,
		// Helper for sending and confirming transactions
		SendAndConfirm: func(ctx context.Context, tx *common.Transaction, opts *tron.ConfirmRetryOptions) (*soliditynode.TransactionInfo, error) {
			options := tron.DefaultConfirmRetryOptions()
			if opts != nil {
				options = opts
			}

			// Send transaction and wait for confirmation
			return client.SendAndConfirmTx(ctx, tx, options)
		},
		// Helper for deploying a contract and waiting for confirmation
		DeployContractAndConfirm: func(
			ctx context.Context, contractName string, abi string, bytecode string, params []interface{}, opts *tron.DeployOptions,
		) (address.Address, *soliditynode.TransactionInfo, error) {
			options := tron.DefaultDeployOptions()
			if opts != nil {
				options = opts
			}

			// Create deploy contract transaction
			deployResponse, err := combinedClient.DeployContract(
				deployerAddr, contractName, abi, bytecode, options.OeLimit, options.CurPercent, options.FeeLimit, params,
			)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to create deploy contract transaction: %w", err)
			}

			// Send transaction and wait for confirmation
			txInfo, err := client.SendAndConfirmTx(ctx, &deployResponse.Transaction, options.ConfirmRetryOptions)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to confirm deploy contract transaction: %w", err)
			}

			// Parse resulting contract address
			contractAddr, err := address.StringToAddress(txInfo.ContractAddress)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse contract address: %w", err)
			}

			// Ensure contract is actually deployed on-chain
			if err := client.CheckContractDeployed(contractAddr); err != nil {
				return nil, nil, fmt.Errorf("contract deployment check failed: %w", err)
			}

			return contractAddr, txInfo, nil
		},
		// Helper for triggering a contract method and waiting for confirmation
		TriggerContractAndConfirm: func(
			ctx context.Context, contractAddr address.Address, functionName string, params []interface{}, opts *tron.TriggerOptions,
		) (*soliditynode.TransactionInfo, error) {
			options := tron.DefaultTriggerOptions()
			if opts != nil {
				options = opts
			}

			// Ensure contract is actually deployed on-chain
			if err := client.CheckContractDeployed(contractAddr); err != nil {
				return nil, fmt.Errorf("contract deployment check failed: %w", err)
			}

			// Create trigger contract transaction
			contractResponse, err := combinedClient.TriggerSmartContract(
				deployerAddr, contractAddr, functionName, params, options.FeeLimit, options.TAmount,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create trigger contract transaction: %w", err)
			}

			// Send transaction and wait for confirmation
			return client.SendAndConfirmTx(ctx, contractResponse.Transaction, options.ConfirmRetryOptions)
		},
	}, nil
}

// Name returns the name of the CTFChainProvider.
func (*CTFChainProvider) Name() string {
	return "TRON CTF Chain Provider"
}

// ChainSelector returns the chain selector of the TRON chain managed by this provider.
func (p *CTFChainProvider) ChainSelector() uint64 {
	return p.selector
}

// BlockChain returns the TRON chain instance managed by this provider. You must call Initialize
// before using this method to ensure the chain is properly set up.
func (p *CTFChainProvider) BlockChain() chain.BlockChain {
	return *p.chain
}

// startContainer starts a CTF container for the TRON chain with the given chain ID.
// It returns the URLs of the full node and solidity node to interact with it.
func (p *CTFChainProvider) startContainer(chainID string) (string, string) {
	var (
		attempts = uint(10)
		bc       *blockchain.Output
	)

	// initialize the docker network used by CTF
	err := framework.DefaultNetwork(p.config.Once)
	require.NoError(p.t, err)

	// Retry logic to handle port conflicts using retry.DoWithData
	bc, err = retry.DoWithData(func() (*blockchain.Output, error) {
		port := freeport.GetOne(p.t)

		output, rerr := blockchain.NewBlockchainNetwork(&blockchain.Input{
			Type:    blockchain.TypeTron,
			ChainID: chainID,
			Port:    strconv.Itoa(port),
			Image:   "tronbox/tre:dev", // dev supports arm (mac) and amd (ci)
		})
		if rerr != nil {
			// Return the ports to freeport to avoid leaking them during retries
			freeport.Return([]int{port})
			return nil, rerr
		}

		return output, nil
	},
		retry.Context(p.t.Context()),
		retry.Attempts(attempts),
		retry.Delay(1*time.Second),
		retry.DelayType(retry.FixedDelay),
		retry.OnRetry(func(attempt uint, err error) {
			p.t.Logf("Attempt %d/%d: Failed to start CTF TRON container: %v", attempt+1, attempts, err)
		}),
	)
	require.NoError(p.t, err, "Failed to start CTF TRON container after %d attempts", attempts)

	testcontainers.CleanupContainer(p.t, bc.Container)

	fullNodeURL := bc.Nodes[0].ExternalHTTPUrl + "/wallet"
	solidityNodeURL := bc.Nodes[0].ExternalHTTPUrl + "/walletsolidity"

	// Wait for the TRON node to be ready
	p.waitForTronNode(fullNodeURL, solidityNodeURL)

	return fullNodeURL, solidityNodeURL
}

// waitForTronNode waits for the TRON node to be ready by checking if it can get the current block.
func (p *CTFChainProvider) waitForTronNode(fullNodeURL, solidityNodeURL string) {
	fullNodeUrlObj, err := url.Parse(fullNodeURL)
	require.NoError(p.t, err, "Failed to parse full node URL")

	solidityNodeUrlObj, err := url.Parse(solidityNodeURL)
	require.NoError(p.t, err, "Failed to parse solidity node URL")

	combinedClient, err := sdk.CreateCombinedClient(fullNodeUrlObj, solidityNodeUrlObj)
	require.NoError(p.t, err, "Failed to create combined client")

	var ready bool
	for i := range 30 {
		time.Sleep(time.Second)
		blockInfo, err := combinedClient.GetNowBlock()
		if err != nil {
			p.t.Logf("TRON node not ready yet (attempt %d): %+v\n", i+1, err)
			continue
		}

		if blockInfo != nil && len(blockInfo.BlockID) > 0 {
			// Extract chain ID from block for verification
			blockId := blockInfo.BlockID
			chainIdHex := blockId[len(blockId)-8:]
			chainIdInt := new(big.Int)
			chainIdInt.SetString(chainIdHex, 16)
			chainId := chainIdInt.String()
			p.t.Logf("TRON node ready, chain ID: %s", chainId)
			ready = true

			break
		}
	}
	require.True(p.t, ready, "TRON network not ready")
}

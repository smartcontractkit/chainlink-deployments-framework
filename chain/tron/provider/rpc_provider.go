package provider

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/http/common"
	"github.com/fbsobreira/gotron-sdk/pkg/http/soliditynode"
	"github.com/smartcontractkit/chainlink-tron/relayer/sdk"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron/provider/rpcclient"
)

// RPCChainProviderConfig holds the configuration required to initialize a Tron RPC chain provider.
type RPCChainProviderConfig struct {
	FullNodeURL       string          // URL of the full node.
	SolidityNodeURL   string          // URL of the solidity node.
	DeployerSignerGen SignerGenerator // Generator used to create the deployer's signer and address.
}

// validate checks whether the configuration contains all required values.
func (c RPCChainProviderConfig) validate() error {
	if c.FullNodeURL == "" {
		return errors.New("full node url is required")
	}
	if c.SolidityNodeURL == "" {
		return errors.New("solidity node url is required")
	}
	if c.DeployerSignerGen == nil {
		return errors.New("deployer signer generator is required")
	}

	return nil
}

// Ensure interface implementation
var _ chain.Provider = (*RPCChainProvider)(nil)

// RPCChainProvider implements the Chainlink `chain.Provider` interface for interacting with a Tron blockchain using RPC.
// It encapsulates configuration and connection details needed to interact with a live or local Tron node.
type RPCChainProvider struct {
	selector uint64                 // Unique chain selector identifier.
	config   RPCChainProviderConfig // Configuration used to set up the provider.
	chain    *tron.Chain            // Cached reference to the initialized Tron chain instance.
}

// NewRPCChainProvider creates a new Tron RPC provider instance with the given chain selector and configuration.
// The actual connection is deferred until Initialize is called.
func NewRPCChainProvider(selector uint64, config RPCChainProviderConfig) *RPCChainProvider {
	return &RPCChainProvider{
		selector: selector,
		config:   config,
	}
}

// Initialize sets up the Tron chain provider and returns a Chain instance.
// It connects to the configured full and solidity nodes, initializes the keystore, and wires up helper methods.
func (p *RPCChainProvider) Initialize(ctx context.Context) (chain.BlockChain, error) {
	// If already initialized, return cached chain
	if p.chain != nil {
		return *p.chain, nil
	}

	// Validate config
	if err := p.config.validate(); err != nil {
		return nil, fmt.Errorf("invalid Tron RPC config: %w", err)
	}

	// Parse URLs for node connections
	fullNodeUrlObj, err := url.Parse(p.config.FullNodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse full node URL: %w", err)
	}
	solidityNodeUrlObj, err := url.Parse(p.config.SolidityNodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse solidity node URL: %w", err)
	}

	// Create a client that wraps both full node and solidity node connections
	combinedClient, err := sdk.CreateCombinedClient(fullNodeUrlObj, solidityNodeUrlObj)
	if err != nil {
		return nil, fmt.Errorf("failed to create combined client: %w", err)
	}

	// Get deployer address from the signer generator
	deployerAddr, err := p.config.DeployerSignerGen.GetAddress()
	if err != nil {
		return nil, fmt.Errorf("failed to get deployer address: %w", err)
	}

	// Initialize local RPC client wrapper that uses the signer generator's signing function
	client := rpcclient.New(combinedClient, p.config.DeployerSignerGen.Sign)

	// Construct and cache the Tron chain instance with helper methods for deploying and interacting with contracts
	p.chain = &tron.Chain{
		ChainMetadata: tron.ChainMetadata{
			Selector: p.selector,
		},
		Client:   combinedClient,                  // Underlying client for Tron node communication
		SignHash: p.config.DeployerSignerGen.Sign, // Function for signing transactions
		Address:  deployerAddr,                    // Default "from" address for transactions
		URL:      p.config.FullNodeURL,
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
	}

	return *p.chain, nil
}

// Name returns the name of the provider.
func (p *RPCChainProvider) Name() string {
	return "Tron RPC Chain Provider"
}

// ChainSelector returns the chain selector value used to identify this chain.
func (p *RPCChainProvider) ChainSelector() uint64 {
	return p.selector
}

// BlockChain returns the initialized Tron chain instance.
func (p *RPCChainProvider) BlockChain() chain.BlockChain {
	return *p.chain
}

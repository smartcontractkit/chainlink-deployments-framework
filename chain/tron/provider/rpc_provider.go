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
	cldf_tron "github.com/smartcontractkit/chainlink-deployments-framework/chain/tron"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron/provider/rpcclient"
)

// RPCChainProviderConfig holds the configuration required to initialize a Tron RPC chain provider.
type RPCChainProviderConfig struct {
	FullNodeURL       string           // URL of the full node (used for submitting transactions).
	SolidityNodeURL   string           // URL of the solidity node (used for confirmed state queries).
	DeployerSignerGen AccountGenerator // Generator used to create the deployer's keystore and address.
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
type RPCChainProvider struct {
	selector uint64                 // Unique chain selector identifier.
	config   RPCChainProviderConfig // Configuration used to set up the provider.
	chain    *cldf_tron.Chain       // Reference to the initialized Tron chain instance.
}

// NewRPCChainProvider creates a new Tron RPC provider instance with the given chain selector and configuration.
func NewRPCChainProvider(selector uint64, config RPCChainProviderConfig) *RPCChainProvider {
	return &RPCChainProvider{
		selector: selector,
		config:   config,
	}
}

// Initialize sets up the Tron chain provider and returns a Chain instance.
// It connects to the configured full and solidity nodes, initializes the keystore, and wires up helper methods.
func (p *RPCChainProvider) Initialize(ctx context.Context) (chain.BlockChain, error) {
	if p.chain != nil {
		return *p.chain, nil
	}

	if err := p.config.validate(); err != nil {
		return nil, fmt.Errorf("invalid Tron RPC config: %w", err)
	}

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

	// Generate deployer keystore and address
	ks, addr, err := p.config.DeployerSignerGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate signer: %w", err)
	}

	// Initialize local RPC client wrapper
	client := rpcclient.New(combinedClient, ks, addr)

	// Construct and cache the Tron chain instance with helper methods
	p.chain = &cldf_tron.Chain{
		ChainMetadata: cldf_tron.ChainMetadata{
			Selector: p.selector,
		},
		Client:   combinedClient,
		Keystore: ks,
		Address:  addr,
		URL:      p.config.FullNodeURL,

		// Helper for sending and confirming transactions
		SendAndConfirm: func(ctx context.Context, tx *common.Transaction, opts ...cldf_tron.ConfirmRetryOptions) (*soliditynode.TransactionInfo, error) {
			options := cldf_tron.DefaultConfirmRetryOptions()
			if len(opts) > 0 {
				options = opts[0]
			}
			return client.SendAndConfirmTx(ctx, tx, options)
		},

		// Helper for deploying a contract and waiting for confirmation
		DeployContractAndConfirm: func(
			ctx context.Context, contractName string, abi string, bytecode string, params []interface{}, opts ...cldf_tron.DeployOptions,
		) (address.Address, *soliditynode.TransactionInfo, error) {
			options := cldf_tron.DefaultDeployOptions()
			if len(opts) > 0 {
				options = opts[0]
			}

			deployResponse, err := combinedClient.DeployContract(
				addr, contractName, abi, bytecode, options.OeLimit, options.CurPercent, options.FeeLimit, params,
			)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to create deploy contract transaction: %w", err)
			}

			txInfo, err := client.SendAndConfirmTx(ctx, &deployResponse.Transaction, options.ConfirmRetryOptions)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to confirm deploy contract transaction: %w", err)
			}

			contractAddress, err := address.StringToAddress(txInfo.ContractAddress)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse contract address: %w", err)
			}

			if err := client.CheckContractDeployed(contractAddress); err != nil {
				return nil, nil, fmt.Errorf("contract deployment check failed: %w", err)
			}

			return contractAddress, txInfo, nil
		},

		// Helper for triggering a contract method and waiting for confirmation
		TriggerContractAndConfirm: func(
			ctx context.Context, contractAddr address.Address, functionName string, params []interface{}, opts ...cldf_tron.TriggerOptions,
		) (*soliditynode.TransactionInfo, error) {
			options := cldf_tron.DefaultTriggerOptions()
			if len(opts) > 0 {
				options = opts[0]
			}

			contractResponse, err := combinedClient.TriggerSmartContract(
				addr, contractAddr, functionName, params, options.FeeLimit, options.TAmount,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create trigger contract transaction: %w", err)
			}

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

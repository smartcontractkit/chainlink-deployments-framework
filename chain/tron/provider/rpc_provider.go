package provider

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/http/common"
	"github.com/fbsobreira/gotron-sdk/pkg/http/soliditynode"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	cldf_tron "github.com/smartcontractkit/chainlink-deployments-framework/chain/tron"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron/provider/rpcclient"
	"github.com/smartcontractkit/chainlink-tron/relayer/sdk"
)

// RPCChainProviderConfig holds configuration for Tron RPC provider
type RPCChainProviderConfig struct {
	FullNodeURL       string
	SolidityNodeURL   string
	DeployerSignerGen AccountGenerator
}

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

// RPCChainProvider implements the Chainlink RPC provider for Tron
type RPCChainProvider struct {
	selector uint64
	config   RPCChainProviderConfig
	chain    *cldf_tron.Chain
}

// NewRPCChainProvider creates a new instance of Tron RPC provider
func NewRPCChainProvider(selector uint64, config RPCChainProviderConfig) *RPCChainProvider {
	return &RPCChainProvider{
		selector: selector,
		config:   config,
	}
}

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

	// Initialize combined client
	combinedClient, err := sdk.CreateCombinedClient(fullNodeUrlObj, solidityNodeUrlObj)
	if err != nil {
		return nil, fmt.Errorf("failed to create combined client: %w", err)
	}

	// Generate keystore and address using the deployer signer generator
	ks, addr, err := p.config.DeployerSignerGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate signer: %w", err)
	}

	// Initialize the Tron client with the combined client, keystore, and address
	client := rpcclient.New(combinedClient, ks, addr)

	p.chain = &cldf_tron.Chain{
		ChainMetadata: cldf_tron.ChainMetadata{
			Selector: p.selector,
		},
		Client:   &combinedClient,
		Keystore: ks,
		Address:  addr,
		URL:      p.config.FullNodeURL,
		SendAndConfirm: func(ctx context.Context, tx *common.Transaction, opts ...cldf_tron.ConfirmRetryOptions) (*soliditynode.TransactionInfo, error) {
			options := cldf_tron.DefaultConfirmRetryOptions()
			if len(opts) > 0 {
				options = opts[0]
			}

			return client.SendAndConfirmTx(ctx, tx, options)
		},
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

			err = client.CheckContractDeployed(contractAddress)
			if err != nil {
				return nil, nil, fmt.Errorf("contract deployment check failed: %w", err)
			}

			return contractAddress, txInfo, nil
		},
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

func (p *RPCChainProvider) Name() string {
	return "Tron RPC Chain Provider"
}

func (p *RPCChainProvider) ChainSelector() uint64 {
	return p.selector
}

func (p *RPCChainProvider) BlockChain() chain.BlockChain {
	return *p.chain
}

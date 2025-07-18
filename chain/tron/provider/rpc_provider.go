package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	cldf_tron "github.com/smartcontractkit/chainlink-deployments-framework/chain/tron"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron/provider/rpcclient"
)

// RPCChainProviderConfig holds configuration for Tron RPC provider
type RPCChainProviderConfig struct {
	RPCURL            string
	DeployerSignerGen AccountGenerator
}

func (c RPCChainProviderConfig) validate() error {
	if c.RPCURL == "" {
		return errors.New("rpc url is required")
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

	// Initialize gRPC client
	grpcClient := client.NewGrpcClient(p.config.RPCURL)
	if err := grpcClient.Start(grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, ""))); err != nil {
		return nil, fmt.Errorf("failed to start Tron gRPC client: %w", err)
	}

	// Generate keystore and account using the deployer signer generator
	ks, acc, err := p.config.DeployerSignerGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate signer: %w", err)
	}

	// Initialize the Tron client with the gRPC client, keystore, and account
	client := rpcclient.New(grpcClient, ks, acc)

	p.chain = &cldf_tron.Chain{
		ChainMetadata: cldf_tron.ChainMetadata{
			Selector: p.selector,
		},
		Client:   grpcClient,
		Keystore: ks,
		Account:  acc,
		URL:      p.config.RPCURL,
		SendAndConfirm: func(ctx context.Context, tx *api.TransactionExtention, opts ...cldf_tron.ConfirmRetryOptions) (*core.TransactionInfo, error) {
			options := cldf_tron.DefaultConfirmRetryOptions()
			if len(opts) > 0 {
				options = opts[0]
			}

			return client.SendAndConfirmTx(ctx, tx, options)
		},
		DeployContractAndConfirm: func(
			ctx context.Context, contractName string, abi *core.SmartContract_ABI, bytecode string, opts ...cldf_tron.DeployOptions,
		) (*core.TransactionInfo, error) {
			options := cldf_tron.DefaultDeployOptions()
			if len(opts) > 0 {
				options = opts[0]
			}

			tx, err := grpcClient.DeployContract(
				acc.Address.String(), contractName, abi, bytecode, options.FeeLimit, options.CurPercent, options.EnergyLimit,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create deploy contract transaction: %w", err)
			}

			return client.SendAndConfirmTx(ctx, tx, options.ConfirmRetryOptions)
		},
		TriggerContractAndConfirm: func(
			ctx context.Context, contractAddr address.Address, functionName string, jsonParams string, opts ...cldf_tron.TriggerOptions,
		) (*core.TransactionInfo, error) {
			options := cldf_tron.DefaultTriggerOptions()
			if len(opts) > 0 {
				options = opts[0]
			}

			tx, err := grpcClient.TriggerContract(
				acc.Address.String(), contractAddr.String(), functionName, jsonParams, options.FeeLimit, options.TAmount, options.TTokenID, options.TTokenAmount,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create deploy contract transaction: %w", err)
			}

			return client.SendAndConfirmTx(ctx, tx, options.ConfirmRetryOptions)
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

package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron"
	cldf_tron "github.com/smartcontractkit/chainlink-deployments-framework/chain/tron"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron/provider/rpcclient"
)

// RPCChainProviderConfig holds configuration for TRON RPC provider
type RPCChainProviderConfig struct {
	RPCURL            string
	DeployerSignerGen AccountGenerator // Provides keystore.Wallet
	Mnemonic          string           // Optional, for deterministic wallet generation
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

// RPCChainProvider implements the Chainlink RPC provider for TRON
type RPCChainProvider struct {
	selector uint64
	config   RPCChainProviderConfig
	chain    *cldf_tron.Chain
}

// NewRPCChainProvider creates a new instance of TRON RPC provider
func NewRPCChainProvider(selector uint64, config RPCChainProviderConfig) *RPCChainProvider {
	return &RPCChainProvider{
		selector: selector,
		config:   config,
	}
}

func DefaultDeployOptions() cldf_tron.DeployOptions {
	return cldf_tron.DeployOptions{
		FeeLimit:    10_000_000,
		CurPercent:  100,
		EnergyLimit: 10_000_000,
	}
}

func DefaultTriggerOptions() cldf_tron.TriggerOptions {
	return cldf_tron.TriggerOptions{
		FeeLimit:     10_000_000,
		TAmount:      0,
		TTokenID:     "",
		TTokenAmount: 0,
	}
}

func (p *RPCChainProvider) Initialize(ctx context.Context) (chain.BlockChain, error) {
	if p.chain != nil {
		return p.chain, nil
	}

	if err := p.config.validate(); err != nil {
		return nil, fmt.Errorf("invalid TRON RPC config: %w", err)
	}

	// Initialize gRPC client
	grpcClient := client.NewGrpcClient(p.config.RPCURL)
	if err := grpcClient.Start(); err != nil {
		return nil, fmt.Errorf("failed to start TRON gRPC client: %w", err)
	}

	// Generate signer (provides keystore.Wallet)
	ks, acc, err := p.config.DeployerSignerGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate signer: %w", err)
	}

	// Initialize the TRON client with the gRPC client, keystore, and account
	client := rpcclient.New(grpcClient, ks, acc)

	p.chain = &tron.Chain{
		ChainMetadata: tron.ChainMetadata{
			Selector: p.selector,
		},
		Client:   grpcClient,
		Keystore: ks,
		Account:  acc,
		URL:      p.config.RPCURL,
		SendAndConfirmTx: func(ctx context.Context, tx *api.TransactionExtention) (*core.TransactionInfo, error) {
			return client.SendAndConfirmTx(ctx, tx, rpcclient.WithRetry(500, 50*time.Millisecond))
		},
		DeployContractAndConfirm: func(
			ctx context.Context, contractName string, abi *core.SmartContract_ABI, bytecode string, opts ...cldf_tron.DeployOptions,
		) (*core.TransactionInfo, error) {
			option := DefaultDeployOptions()
			if len(opts) > 0 {
				option = opts[0]
			}

			tx, err := grpcClient.DeployContract(
				string(acc.Address), contractName, abi, bytecode, option.FeeLimit, option.CurPercent, option.EnergyLimit,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create deploy contract transaction: %w", err)
			}

			return client.SendAndConfirmTx(ctx, tx, rpcclient.WithRetry(500, 50*time.Millisecond))
		},
		TriggerContractAndConfirm: func(
			ctx context.Context, contractAddr common.Address, functionName string, jsonParams string, opts ...cldf_tron.TriggerOptions,
		) (*core.TransactionInfo, error) {
			option := DefaultTriggerOptions()
			if len(opts) > 0 {
				option = opts[0]
			}

			tx, err := grpcClient.TriggerContract(
				string(acc.Address), contractAddr.String(), functionName, jsonParams, option.FeeLimit, option.TAmount, option.TTokenID, option.TTokenAmount,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create deploy contract transaction: %w", err)
			}

			return client.SendAndConfirmTx(ctx, tx, rpcclient.WithRetry(500, 50*time.Millisecond))
		},
	}

	return *p.chain, nil
}

func (p *RPCChainProvider) Name() string {
	return "TRON RPC Provider"
}

func (p *RPCChainProvider) ChainSelector() uint64 {
	return p.selector
}

func (p *RPCChainProvider) BlockChain() chain.BlockChain {
	return p.chain
}

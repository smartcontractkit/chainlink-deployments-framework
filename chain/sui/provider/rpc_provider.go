package provider

import (
	"context"
	"errors"
	"fmt"

	sui_sdk "github.com/block-vision/sui-go-sdk/sui"
	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/sui"
)

// RPCChainProviderConfig holds the configuration to initialize the RPCChainProvider.
type RPCChainProviderConfig struct {
	// Required: The RPC URL to connect to the Sui node
	RPCURL string

	DeployerSignerGen AccountGenerator
}

// validate checks if the RPCChainProviderConfig is valid.
func (c RPCChainProviderConfig) validate() error {
	if c.RPCURL == "" {
		return errors.New("rpc url is required")
	}
	if c.DeployerSignerGen == nil {
		return errors.New("deployer signer generator is required")
	}
	return nil
}

var _ chain.Provider = (*RPCChainProvider)(nil)

// RPCChainProvider is a chain provider that provides a chain that connects to an Sui node via RPC
type RPCChainProvider struct {
	// Sui chain selector, used to identify the chain.
	selector uint64

	// RPCChainProviderConfig holds the configuration for the RPCChainProvider.
	config RPCChainProviderConfig

	// chain is the Sui chain instance that this provider manages. The Initialize method
	// sets up the chain.
	chain *sui.Chain
}

// NewRPCChainProvider creates a new RPCChainProvider with the given selector and configuration.
func NewRPCChainProvider(selector uint64, config RPCChainProviderConfig) *RPCChainProvider {
	p := &RPCChainProvider{
		selector: selector,
		config:   config,
	}

	return p
}

// Initialize initializes the RPCChainProvider, validating the configuration and setting up the
// Aptos chain client.
func (p *RPCChainProvider) Initialize(_ context.Context) (chain.BlockChain, error) {
	if p.chain != nil {
		return p.chain, nil // Already initialized
	}

	// Validate the provider configuration
	if err := p.config.validate(); err != nil {
		return nil, fmt.Errorf("failed to validate provider config: %w", err)
	}

	// Generate the deployer account
	deployerSigner, err := p.config.DeployerSignerGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate deployer account: %w", err)
	}

	// Validate that the chain selector is known
	_, err = chain_selectors.GetChainIDFromSelector(p.selector)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID from selector %d: %w", p.selector, err)
	}

	client := sui_sdk.NewSuiClient(p.config.RPCURL)

	p.chain = &sui.Chain{
		ChainMetadata: sui.ChainMetadata{
			Selector: p.selector,
		},
		Client: client,
		Signer: deployerSigner,
		URL:    p.config.RPCURL,
	}

	return *p.chain, nil
}

// Name returns the name of the RPCChainProvider.
func (*RPCChainProvider) Name() string {
	return "Sui RPC Chain Provider"
}

// ChainSelector returns the chain selector of the Aptos chain managed by this provider.
func (p *RPCChainProvider) ChainSelector() uint64 {
	return p.selector
}

// BlockChain returns the Aptos chain instance managed by this provider. You must call Initialize
// before using this method to ensure the chain is properly set up.
func (p *RPCChainProvider) BlockChain() chain.BlockChain {
	return *p.chain
}

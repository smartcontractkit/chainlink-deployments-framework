package provider

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/smartcontractkit/chainlink-tron/relayer/sdk"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/tron"
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

	// Create the Tron chain instance using the extracted function
	chain, err := GetTronChain(p.selector, combinedClient, p.config.DeployerSignerGen)
	if err != nil {
		return nil, fmt.Errorf("failed to create tron chain: %w", err)
	}

	// Cache the chain instance
	p.chain = &chain

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

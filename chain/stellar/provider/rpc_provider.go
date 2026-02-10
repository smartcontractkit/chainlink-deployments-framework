package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/stellar/go-stellar-sdk/clients/rpcclient"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/stellar"
)

type RPCChainProviderConfig struct {
	// Required: The Soroban RPC URL to connect to the Stellar network
	SorobanRPCURL string

	// Required: The network passphrase identifying the Stellar network
	NetworkPassphrase string

	// Optional: The Friendbot URL for funding test accounts (only required for testing environments)
	FriendbotURL string

	// Required: A generator for the deployer keypair. Use KeypairFromHex to create a deployer
	// keypair from a hex-encoded private key.
	DeployerKeypairGen KeypairGenerator
}

func (c RPCChainProviderConfig) validate() error {
	if c.SorobanRPCURL == "" {
		return errors.New("soroban RPC URL is required")
	}
	if c.NetworkPassphrase == "" {
		return errors.New("network passphrase is required")
	}
	// Note: FriendbotURL is optional; it's only required for testing environments
	if c.DeployerKeypairGen == nil {
		return errors.New("deployer keypair generator is required")
	}

	return nil
}

type RPCChainProvider struct {
	selector uint64
	config   RPCChainProviderConfig

	chain *stellar.Chain
}

var _ chain.Provider = (*RPCChainProvider)(nil)

func NewRPCChainProvider(selector uint64, config RPCChainProviderConfig) *RPCChainProvider {
	return &RPCChainProvider{
		selector: selector,
		config:   config,
	}
}

func (p *RPCChainProvider) Initialize(_ context.Context) (chain.BlockChain, error) {
	if p.chain != nil {
		return p.chain, nil // already initialized
	}

	if err := p.config.validate(); err != nil {
		return nil, fmt.Errorf("failed to validate provider config: %w", err)
	}

	// Generate the deployer keypair
	deployerSigner, err := p.config.DeployerKeypairGen.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate deployer keypair: %w", err)
	}

	// Create the Soroban RPC client
	client := rpcclient.NewClient(p.config.SorobanRPCURL, &http.Client{
		Timeout: 60 * time.Second,
	})

	p.chain = &stellar.Chain{
		ChainMetadata:     stellar.ChainMetadata{Selector: p.selector},
		Client:            client,
		Signer:            deployerSigner,
		URL:               p.config.SorobanRPCURL,
		FriendbotURL:      p.config.FriendbotURL,
		NetworkPassphrase: p.config.NetworkPassphrase,
	}

	return p.chain, nil
}

func (p *RPCChainProvider) Name() string {
	return "Stellar RPC Chain Provider"
}

func (p *RPCChainProvider) ChainSelector() uint64 {
	return p.selector
}

func (p *RPCChainProvider) BlockChain() chain.BlockChain {
	return p.chain
}

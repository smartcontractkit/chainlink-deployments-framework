package provider

import (
	"fmt"
	"strconv"

	aptoslib "github.com/aptos-labs/aptos-go-sdk"
	aptoscrypto "github.com/aptos-labs/aptos-go-sdk/crypto"
	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/aptos"
)

var _ chain.Provider = (*RPCChainProvider)(nil)

type RPCChainProvider struct {
	selector uint64

	// Required
	rpcURL string
	// Required
	deployerKey string

	chain *aptos.Chain
}

func (p *RPCChainProvider) WithRPCURL(rpcURL string) *RPCChainProvider {
	p.rpcURL = rpcURL
	return p
}

func (p *RPCChainProvider) WithDeployerKey(deployerKey string) *RPCChainProvider {
	p.deployerKey = deployerKey
	return p
}

func (p *RPCChainProvider) Initialize() error {
	if err := p.validate(); err != nil {
		return fmt.Errorf("failed to validate provider: %w", err)
	}

	// Get the Aptos Chain ID
	chainIDStr, err := chain_selectors.GetChainIDFromSelector(p.selector)
	if err != nil {
		return fmt.Errorf("failed to get chain ID from selector %d: %w", p.selector, err)
	}

	chainID, err := strconv.ParseUint(chainIDStr, 10, 8)
	if err != nil {
		return fmt.Errorf("failed to parse chain ID %s: %w", chainIDStr, err)
	}

	// Generate the deployer account
	deployerKey := &aptoscrypto.Ed25519PrivateKey{}
	if err = deployerKey.FromHex(p.deployerKey); err != nil {
		return fmt.Errorf("failed to parse deployer key %s: %w", p.deployerKey, err)
	}

	deployerSigner, err := aptoslib.NewAccountFromSigner(deployerKey)
	if err != nil {
		return fmt.Errorf("failed to create Aptos deployer account: %w", err)
	}

	client, err := aptoslib.NewNodeClient(p.rpcURL, uint8(chainID))
	if err != nil {
		return fmt.Errorf("failed to create Aptos RPC client for chain %d: %w", p.selector, err)
	}

	p.chain = &aptos.Chain{
		Selector:       p.selector,
		Client:         client,
		DeployerSigner: deployerSigner,
	}

	return nil
}

func (*RPCChainProvider) Name() string {
	return "Aptos RPC Chain Provider"
}

func (p *RPCChainProvider) ChainSelector() uint64 {
	return p.selector
}

func (p *RPCChainProvider) BlockChain() chain.BlockChain {
	return p.chain
}

func (p *RPCChainProvider) Chain() *aptos.Chain {
	return p.chain
}

func (p *RPCChainProvider) validate() error {
	if p.rpcURL == "" {
		return fmt.Errorf("RPC URL is required")
	}
	if p.deployerKey == "" {
		return fmt.Errorf("deployer key is required")
	}

	return nil
}

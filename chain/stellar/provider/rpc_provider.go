package provider

import (
	"context"
	"errors"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/stellar"
)

type RPCChainProviderConfig struct {
	NetworkPassphrase string
	FriendbotURL      string
	SorobanRPCURL     string
}

func (c RPCChainProviderConfig) validate() error {
	if c.NetworkPassphrase == "" {
		return errors.New("network passphrase is required")
	}
	if c.FriendbotURL == "" {
		return errors.New("Friendbot URL is required")
	}
	if c.SorobanRPCURL == "" {
		return errors.New("Soroban RPC URL is required")
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
		return nil, err
	}

	p.chain = &stellar.Chain{
		ChainMetadata: stellar.ChainMetadata{Selector: p.selector},
	}

	return *p.chain, nil
}

func (p *RPCChainProvider) Name() string {
	return "Stellar RPC Chain Provider"
}

func (p *RPCChainProvider) ChainSelector() uint64 {
	return p.selector
}

func (p *RPCChainProvider) BlockChain() chain.BlockChain {
	if p.chain == nil {
		return nil
	}
	return *p.chain
}

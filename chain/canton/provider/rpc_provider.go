package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/canton"
)

type RPCChainProviderConfig struct {
	Endpoints    []canton.ParticipantEndpoints
	JWTProviders []canton.JWTProvider
}

func (c RPCChainProviderConfig) validate() error {
	if len(c.Endpoints) == 0 {
		return errors.New("no participants specified")
	}
	if len(c.Endpoints) != len(c.JWTProviders) {
		return errors.New("number of participants must match number of JWT providers")
	}

	return nil
}

var _ chain.Provider = (*RPCChainProvider)(nil)

type RPCChainProvider struct {
	selector uint64
	config   RPCChainProviderConfig

	chain *canton.Chain
}

func NewRPCChainProvider(selector uint64, config RPCChainProviderConfig) *RPCChainProvider {
	return &RPCChainProvider{
		selector: selector,
		config:   config,
	}
}

func (p RPCChainProvider) Initialize(_ context.Context) (chain.BlockChain, error) {
	if p.chain != nil {
		return p.chain, nil // already initialized
	}

	p.chain = &canton.Chain{
		Selector:     p.selector,
		Participants: make([]canton.Participant, len(p.config.Endpoints)),
	}

	for i, participantEndpoints := range p.config.Endpoints {
		p.chain.Participants[i] = canton.Participant{
			Name:        fmt.Sprintf("Participant %v", i+1),
			Endpoints:   participantEndpoints,
			JWTProvider: p.config.JWTProviders[i],
		}
	}

	return p.chain, nil
}

func (p RPCChainProvider) Name() string {
	return "Canton RPC Chain Provider"
}

func (p RPCChainProvider) ChainSelector() uint64 {
	return p.selector
}

func (p RPCChainProvider) BlockChain() chain.BlockChain {
	return p.chain
}

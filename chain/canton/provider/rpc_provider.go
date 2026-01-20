package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/canton"
	chaincommon "github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
)

// RPCChainProviderConfig is the configuration for the RPCChainProvider.
// The number of provided endpoints must match the number of provided JWT providers
// The order of endpoints must correspond to the order of JWT providers
// At least one participant must be provided
type RPCChainProviderConfig struct {
	// Required: List of participant endpoints to connect to
	Endpoints []canton.ParticipantEndpoints
	// Required: List of JWT providers for authentication with the participants
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

// RPCChainProvider initializes a Canton chain instance connecting to existing Canton participants
// via their RPC endpoints.
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

func (p *RPCChainProvider) Initialize(_ context.Context) (chain.BlockChain, error) {
	if p.chain != nil {
		return p.chain, nil // already initialized
	}

	if err := p.config.validate(); err != nil {
		return nil, err
	}

	p.chain = &canton.Chain{
		ChainMetadata: chaincommon.ChainMetadata{Selector: p.selector},
		Participants:  make([]canton.Participant, len(p.config.Endpoints)),
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

func (p *RPCChainProvider) Name() string {
	return "Canton RPC Chain Provider"
}

func (p *RPCChainProvider) ChainSelector() uint64 {
	return p.selector
}

func (p *RPCChainProvider) BlockChain() chain.BlockChain {
	return *p.chain
}

package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/canton"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	ctfCanton "github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain/canton"
)

type RPCChainProviderConfig struct {
	Participants []blockchain.CantonParticipantEndpoints
}

func (c RPCChainProviderConfig) validate() error {
	if len(c.Participants) == 0 {
		return errors.New("no participants specified")
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
		Participants: make([]canton.Participant, len(p.config.Participants)),
	}

	// TODO - this only applies to localnet/CTF, need to adjust for other networks
	for i, participantEndpoints := range p.config.Participants {
		p.chain.Participants[i] = canton.Participant{
			Name:      fmt.Sprintf("Participant %v", i+1),
			Endpoints: participantEndpoints,
			JWT: func(_ context.Context) (string, error) {
				return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
					Issuer:    "",
					Subject:   fmt.Sprintf("user-participant%v", i+1),
					Audience:  []string{ctfCanton.AuthProviderAudience},
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
					NotBefore: jwt.NewNumericDate(time.Now()),
					IssuedAt:  jwt.NewNumericDate(time.Now()),
					ID:        "",
				}).SignedString([]byte(ctfCanton.AuthProviderSecret))
			},
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

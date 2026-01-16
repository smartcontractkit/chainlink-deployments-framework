package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/smartcontractkit/freeport"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/canton"
	"github.com/smartcontractkit/chainlink-testing-framework/framework"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	ctfCanton "github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain/canton"
)

type CTFChainProviderConfig struct {
	NumberOfValidators int

	// Required: A sync.Once instance to ensure that the CTF framework only sets up the new
	// DefaultNetwork once
	Once *sync.Once
}

func (c CTFChainProviderConfig) validate() error {
	if c.NumberOfValidators <= 0 {
		return errors.New("number of validators must be greater than zero")
	}
	if c.Once == nil {
		return errors.New("sync.Once instance is required")
	}

	return nil
}

var _ chain.Provider = (*CTFChainProvider)(nil)

type CTFChainProvider struct {
	t        *testing.T
	selector uint64
	config   CTFChainProviderConfig

	chain *canton.Chain
}

func NewCTFChainProvider(t *testing.T, selector uint64, config CTFChainProviderConfig) *CTFChainProvider {
	t.Helper()

	p := &CTFChainProvider{
		t:        t,
		selector: selector,
		config:   config,
	}

	return p
}

func (p CTFChainProvider) Initialize(ctx context.Context) (chain.BlockChain, error) {
	if p.chain != nil {
		return p.chain, nil // already initialized
	}

	if err := p.config.validate(); err != nil {
		return nil, err
	}

	// initialize the docker network used by CTF
	if err := framework.DefaultNetwork(p.config.Once); err != nil {
		return nil, err
	}

	port := freeport.GetOne(p.t)
	fmt.Println("Port for Canton CTF:", port)
	input := &blockchain.Input{
		Type:                     blockchain.TypeCanton,
		Image:                    "",
		Port:                     strconv.Itoa(port),
		NumberOfCantonValidators: p.config.NumberOfValidators,
	}
	output, err := blockchain.NewBlockchainNetwork(input)
	if err != nil {
		p.t.Logf("Error creating Canton blockchain network: %v", err)
		freeport.Return([]int{port})
		return nil, err
	}

	// Test HTTP health endpoint
	for i, participant := range output.NetworkSpecificData.CantonEndpoints.Participants {
		resp, err := http.Get(fmt.Sprintf("%s/health", participant.HTTPHealthCheckURL))
		require.NoErrorf(p.t, err, "Error reaching Canton participant %d health endpoint", i+1)
		_ = resp.Body.Close()
		require.EqualValues(p.t, http.StatusOK, resp.StatusCode, "Unexpected status code from Canton participant %d health endpoint", i+1)
	}

	p.chain = &canton.Chain{
		Selector:     p.selector,
		Participants: make([]canton.Participant, len(output.NetworkSpecificData.CantonEndpoints.Participants)),
	}

	for i, participantEndpoints := range output.NetworkSpecificData.CantonEndpoints.Participants {
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

func (p CTFChainProvider) Name() string {
	return "Canton CTF Chain Provider"
}

func (p CTFChainProvider) ChainSelector() uint64 {
	return p.selector
}

func (p CTFChainProvider) BlockChain() chain.BlockChain {
	return p.chain
}

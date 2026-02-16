package provider

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/grpc"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/canton"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/canton/provider/authentication"
)

// RPCChainProviderConfig is the configuration for the RPCChainProvider.
// The number of provided endpoints must match the number of provided JWT providers
// The order of endpoints must correspond to the order of JWT providers
// At least one participant must be provided
type RPCChainProviderConfig struct {
	// Required: List of participants to connect to
	Participants []ParticipantConfig
	// (HTTP) The URL to access the SV's Registry API
	// https://docs.sync.global/app_dev/token_standard/index.html#api-references
	RegistryAPIURL string
}

type ParticipantConfig struct {
	// (HTTP) The URL to access the participant's JSON Ledger API
	// https://docs.digitalasset.com/build/3.5/reference/json-api/json-api.html
	JSONLedgerAPIURL string
	// (gRPC) The URL to access the participant's gRPC Ledger API
	// https://docs.digitalasset.com/build/3.5/reference/lapi-proto-docs.html
	GRPCLedgerAPIURL string
	// (gRPC) The URL to access the participant's Admin API
	// Optional - if not set, admin services will not be populated for this participant
	// https://docs.digitalasset.com/operate/3.5/howtos/configure/apis/admin_api.html
	AdminAPIURL string
	// (HTTP) The URL to access the participant's Validator API
	// https://docs.sync.global/app_dev/validator_api/index.html
	ValidatorAPIURL string
	// The UserID of the user that should be used for accessing the participant's API endpoints.
	UserID string
	// An authentication.Provider implementation that provides the credentials for authenticating with the participant's API endpoints.
	AuthProvider authentication.Provider
}

func (c RPCChainProviderConfig) validate() error {
	if len(c.Participants) == 0 {
		return errors.New("no participants specified")
	}
	for i, participant := range c.Participants {
		if participant.AuthProvider == nil {
			return fmt.Errorf("participant %d has no authentication provider set", i+1)
		}
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
		ChainMetadata:  canton.ChainMetadata{Selector: p.selector},
		Participants:   make([]canton.Participant, len(p.config.Participants)),
		RegistryAPIURL: p.config.RegistryAPIURL,
	}

	for i, participant := range p.config.Participants {
		tokenSource := participant.AuthProvider.TokenSource()
		transportCredentials := participant.AuthProvider.TransportCredentials()
		perRPCCredentials := participant.AuthProvider.PerRPCCredentials()

		// Dial Ledger API endpoint
		ledgerApiConn, err := grpc.NewClient(
			participant.GRPCLedgerAPIURL,
			grpc.WithTransportCredentials(transportCredentials),
			grpc.WithPerRPCCredentials(perRPCCredentials),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create Ledger API gRPC client for participant %d(%s): %w", i+1, participant.GRPCLedgerAPIURL, err)
		}
		ledgerServices := canton.CreateLedgerServiceClients(ledgerApiConn)

		// Dial Admin API endpoint (if set)
		var adminServices *canton.AdminServiceClients
		if participant.AdminAPIURL != "" {
			adminApiConn, err := grpc.NewClient(
				participant.AdminAPIURL,
				grpc.WithTransportCredentials(transportCredentials),
				grpc.WithPerRPCCredentials(perRPCCredentials),
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create Admin API gRPC client for participant %d(%s): %w", i+1, participant.AdminAPIURL, err)
			}
			services := canton.CreateAdminServiceClients(adminApiConn)
			adminServices = &services
		}

		p.chain.Participants[i] = canton.Participant{
			Name: fmt.Sprintf("Participant %v", i+1),
			Endpoints: canton.ParticipantEndpoints{
				JSONLedgerAPIURL: participant.JSONLedgerAPIURL,
				GRPCLedgerAPIURL: participant.GRPCLedgerAPIURL,
				AdminAPIURL:      participant.AdminAPIURL,
				ValidatorAPIURL:  participant.ValidatorAPIURL,
			},
			LedgerServices: ledgerServices,
			AdminServices:  adminServices,
			TokenSource:    tokenSource,
			UserID:         participant.UserID,
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

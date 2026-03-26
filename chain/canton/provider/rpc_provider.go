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
// At least one participant must be provided
type RPCChainProviderConfig struct {
	// Required: List of participants to connect to
	Participants []ParticipantConfig
}

// ParticipantConfig is the configuration of a single participant.
// It contains the configuration details to connect and authenticate against a participant's APIs.
type ParticipantConfig struct {
	// The endpoints used to connect to this participant's APIs.
	Endpoints
	// The (Docker) internal endpoints used to connect to the participant's APIs.
	// If Specified, the resulting chain will have its InternalEndpoints field populated with these values.
	// This is useful when having to connect Canton from within another Docker container.
	// Optional
	InternalEndpoints *Endpoints
	// The UserID of the user that should be used for accessing the participant's API endpoints.
	// Required
	UserID string
	// The PartyID of the party that should be used for accessing the participant's API endpoints.
	// Required
	PartyID string
	// An authentication.Provider implementation that provides the credentials for authenticating with the participant's API endpoints.
	// Required
	AuthProvider authentication.Provider
}

type Endpoints struct {
	// (HTTP) The URL to access the participant's JSON Ledger API
	// Optional
	// https://docs.digitalasset.com/build/3.5/reference/json-api/json-api.html
	JSONLedgerAPIURL string
	// (gRPC) The URL to access the participant's gRPC Ledger API
	// Required
	// https://docs.digitalasset.com/build/3.5/reference/lapi-proto-docs.html
	GRPCLedgerAPIURL string
	// (gRPC) The URL to access the participant's Admin API
	// Optional - if not set, admin services will not be populated for this participant
	// https://docs.digitalasset.com/operate/3.5/howtos/configure/apis/admin_api.html
	AdminAPIURL string
	// (HTTP) The URL to access the participant's Validator API
	// Optional
	// https://docs.sync.global/app_dev/validator_api/index.html
	ValidatorAPIURL string
}

func (c RPCChainProviderConfig) validate() error {
	if len(c.Participants) == 0 {
		return errors.New("no participants specified")
	}
	for i, participant := range c.Participants {
		if participant.GRPCLedgerAPIURL == "" {
			return fmt.Errorf("participant %d has no gRPC Ledger API URL set", i+1)
		}
		if participant.UserID == "" {
			return fmt.Errorf("participant %d has no User ID set", i+1)
		}
		if participant.PartyID == "" {
			return fmt.Errorf("participant %d has no Party ID set", i+1)
		}
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
		ChainMetadata: canton.ChainMetadata{Selector: p.selector},
		Participants:  make([]canton.Participant, len(p.config.Participants)),
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

		// Populate internal endpoints (if set)
		var internalEndpoints *canton.ParticipantEndpoints
		if participant.InternalEndpoints != nil {
			internalEndpoints = &canton.ParticipantEndpoints{
				JSONLedgerAPIURL: participant.InternalEndpoints.JSONLedgerAPIURL,
				GRPCLedgerAPIURL: participant.InternalEndpoints.GRPCLedgerAPIURL,
				AdminAPIURL:      participant.InternalEndpoints.AdminAPIURL,
				ValidatorAPIURL:  participant.InternalEndpoints.ValidatorAPIURL,
			}
		}

		p.chain.Participants[i] = canton.Participant{
			Name: fmt.Sprintf("Participant %v", i+1),
			Endpoints: canton.ParticipantEndpoints{
				JSONLedgerAPIURL: participant.JSONLedgerAPIURL,
				GRPCLedgerAPIURL: participant.GRPCLedgerAPIURL,
				AdminAPIURL:      participant.AdminAPIURL,
				ValidatorAPIURL:  participant.ValidatorAPIURL,
			},
			InternalEndpoints: internalEndpoints,
			LedgerServices:    ledgerServices,
			AdminServices:     adminServices,
			TokenSource:       tokenSource,
			UserID:            participant.UserID,
			PartyID:           participant.PartyID,
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

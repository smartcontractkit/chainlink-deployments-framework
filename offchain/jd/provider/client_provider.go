package provider

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/oauth2"
	"google.golang.org/grpc/credentials"

	"github.com/smartcontractkit/chainlink-deployments-framework/offchain"
	"github.com/smartcontractkit/chainlink-deployments-framework/offchain/jd"
)

// ClientOffchainProviderConfig holds the configuration to initialize the ClientOffchainProvider.
type ClientOffchainProviderConfig struct {
	// Required: The gRPC URL to connect to the Job Distributor service.
	GRPC string
	// Optional: The WebSocket RPC URL for the Job Distributor service.
	WSRPC string
	// Optional: Transport credentials for secure gRPC connections. Defaults to insecure.NewCredentials()
	Creds credentials.TransportCredentials
	// Optional: OAuth2 token source for authentication.
	Auth oauth2.TokenSource
}

// validate checks if the ClientOffchainProviderConfig is valid.
func (c ClientOffchainProviderConfig) validate() error {
	if c.GRPC == "" {
		return errors.New("gRPC URL is required")
	}

	return nil
}

var _ offchain.Provider = (*ClientOffchainProvider)(nil)

// ClientOffchainProvider is a JD provider that connects to a Job Distributor service via gRPC.
type ClientOffchainProvider struct {
	config ClientOffchainProviderConfig
	client offchain.Client
}

// NewClientOffchainProvider creates a new ClientOffchainProvider with the given configuration.
func NewClientOffchainProvider(config ClientOffchainProviderConfig) *ClientOffchainProvider {
	return &ClientOffchainProvider{
		config: config,
	}
}

// Initialize initializes the ClientOffchainProvider, setting up the JD client with the provided
// configuration. It returns the initialized offchain.Client or an error if initialization fails.
func (p *ClientOffchainProvider) Initialize(ctx context.Context) (offchain.Client, error) {
	if p.client != nil {
		return p.client, nil // Already initialized
	}

	// Validate the provider configuration
	if err := p.config.validate(); err != nil {
		return nil, fmt.Errorf("failed to validate provider config: %w", err)
	}

	// Create JD configuration from provider config
	jdConfig := jd.JDConfig{
		GRPC:  p.config.GRPC,
		WSRPC: p.config.WSRPC,
		Creds: p.config.Creds,
		Auth:  p.config.Auth,
	}

	// Create the JD client
	client, err := jd.NewJDClient(jdConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create JD client: %w", err)
	}

	p.client = client

	return client, nil
}

// Name returns the name of the ClientOffchainProvider.
func (*ClientOffchainProvider) Name() string {
	return "Job Distributor Client Offchain Provider"
}

// OffchainClient returns the JD client instance managed by this provider.
// You must call Initialize before using this method to ensure the client is properly set up.
func (p *ClientOffchainProvider) OffchainClient() offchain.Client {
	return p.client
}

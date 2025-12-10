package provider

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/oauth2"
	"google.golang.org/grpc/credentials"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	"github.com/smartcontractkit/chainlink-deployments-framework/offchain"
	"github.com/smartcontractkit/chainlink-deployments-framework/offchain/jd"
)

// ClientProviderOption is a functional option for configuring ClientOffchainProvider.
type ClientProviderOption func(*ClientOffchainProviderConfig)

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

	// Private fields for dry run configuration
	dryRun       bool
	dryRunLogger logger.Logger
}

// WithDryRun enables dry run mode, which simulates write operations without executing them.
// Read operations are still forwarded to the real backend.
func WithDryRun(lggr logger.Logger) ClientProviderOption {
	return func(c *ClientOffchainProviderConfig) {
		c.dryRun = true
		c.dryRunLogger = lggr
	}
}

// validate checks if the ClientOffchainProviderConfig is valid.
func (c ClientOffchainProviderConfig) validate() error {
	if c.GRPC == "" {
		return errors.New("gRPC URL is required")
	}

	if c.dryRun && c.dryRunLogger == nil {
		return errors.New("dry run logger is required when dry run mode is enabled")
	}

	return nil
}

var _ offchain.Provider = (*ClientOffchainProvider)(nil)

// ClientOffchainProvider is a JD provider that connects to a Job Distributor service via gRPC.
type ClientOffchainProvider struct {
	config ClientOffchainProviderConfig
	client offchain.Client
}

// NewClientOffchainProvider creates a new ClientOffchainProvider with the given configuration and options.
// Available options:
// - WithDryRun(lggr logger.Logger) ClientProviderOption
func NewClientOffchainProvider(config ClientOffchainProviderConfig, opts ...ClientProviderOption) *ClientOffchainProvider {
	for _, opt := range opts {
		opt(&config)
	}

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
	jdClient, err := jd.NewJDClient(jdConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create JD client: %w", err)
	}

	// Conditionally wrap with dry run client if dry run mode is enabled
	var client offchain.Client = jdClient
	if p.config.dryRun {
		client = jd.NewDryRunJobDistributor(jdClient, p.config.dryRunLogger)
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

package provider

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/avast/retry-go/v4"
	csav1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/csa"
	ctfjd "github.com/smartcontractkit/chainlink-testing-framework/framework/components/jd"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/postgres"

	"github.com/smartcontractkit/chainlink-deployments-framework/offchain"
	"github.com/smartcontractkit/chainlink-deployments-framework/offchain/jd"
)

const (
	// DefaultCSAEncryptionKey is the default CSA encryption key used when none is provided
	DefaultCSAEncryptionKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
)

// CTFOffchainProviderConfig holds the configuration to initialize the CTFOffchainProvider.
type CTFOffchainProviderConfig struct {
	// Optional: Docker image for the JD service. If not provided, will use environment variable CTF_JD_IMAGE.
	Image string
	// Optional: GRPC port for the JD service. Defaults to 14231.
	GRPCPort string
	// Optional: WebSocket RPC port for the JD service. Defaults to 8080.
	WSRPCPort string
	// Optional: CSA encryption key. Defaults to a 64-character hex string.
	CSAEncryptionKey string
	// Optional: Docker file path for building JD image locally.
	DockerFilePath string
	// Optional: Docker context for building JD image locally.
	DockerContext string
	// Optional: SQL dump path for JD database initialization.
	JDSQLDumpPath string
	// Optional: PostgreSQL database configuration. If not provided, will use default.
	DBInput *postgres.Input
}

// validate checks if the CTFOffchainProviderConfig is valid.
func (c CTFOffchainProviderConfig) validate() error {
	// Check if either Image is provided or CTF_JD_IMAGE environment variable is set
	if c.Image == "" {
		ctfJDImage := os.Getenv("CTF_JD_IMAGE")
		if ctfJDImage == "" {
			return errors.New("either Image must be provided in config or CTF_JD_IMAGE environment variable must be set")
		}
	}

	return nil
}

var _ offchain.Provider = (*CTFOffchainProvider)(nil)

// CTFOffchainProvider manages a Job Distributor (JD) instance running inside a Chainlink Testing Framework
// (CTF) Docker container.
//
// This provider requires Docker to be installed and operational. Spinning up a new container
// can be slow, so it is recommended to initialize the provider only once per test suite or parent
// test to optimize performance.
type CTFOffchainProvider struct {
	t      *testing.T
	config CTFOffchainProviderConfig

	client offchain.Client
}

// NewCTFOffchainProvider creates a new CTFOffchainProvider with the given configuration.
func NewCTFOffchainProvider(
	t *testing.T, config CTFOffchainProviderConfig,
) *CTFOffchainProvider {
	t.Helper()

	p := &CTFOffchainProvider{
		t:      t,
		config: config,
	}

	return p
}

// Initialize sets up the Job Distributor by validating the configuration, starting a CTF container,
// and constructing the JD client instance.
func (p *CTFOffchainProvider) Initialize(ctx context.Context) (offchain.Client, error) {
	if p.client != nil {
		return p.client, nil // Already initialized
	}

	// Validate the provider configuration
	if err := p.config.validate(); err != nil {
		return nil, err
	}

	// Set default CSA encryption key if not provided
	csaEncryptionKey := p.config.CSAEncryptionKey
	if csaEncryptionKey == "" {
		csaEncryptionKey = DefaultCSAEncryptionKey
	}

	// Create JD input configuration from provider config
	jdInput := &ctfjd.Input{
		Image:            p.config.Image,
		GRPCPort:         p.config.GRPCPort,
		WSRPCPort:        p.config.WSRPCPort,
		CSAEncryptionKey: csaEncryptionKey,
		DockerFilePath:   p.config.DockerFilePath,
		DockerContext:    p.config.DockerContext,
		JDSQLDumpPath:    p.config.JDSQLDumpPath,
		DBInput:          p.config.DBInput,
	}

	// Create the JD container using CTF
	jdOutput, err := ctfjd.NewJD(jdInput)
	if err != nil {
		return nil, err
	}

	// Create JD configuration from the CTF output
	jdConfig := jd.JDConfig{
		GRPC: jdOutput.ExternalGRPCUrl,
		// Note: Using insecure credentials for testing
		Creds: nil,
		Auth:  nil,
	}

	// Create the JD client
	client, err := jd.NewJDClient(jdConfig)
	if err != nil {
		return nil, err
	}

	p.client = client

	// Perform health check to ensure JD service is ready
	if err := p.healthCheck(ctx, client); err != nil {
		return nil, err
	}

	return client, nil
}

// healthCheck verifies that the JD service is ready by calling GetKeypair with retry logic.
func (p *CTFOffchainProvider) healthCheck(ctx context.Context, client offchain.Client) error {
	p.t.Helper()

	err := retry.Do(func() error {
		// Try to Get CSA keypair as a health check
		_, err := client.GetKeypair(ctx, &csav1.GetKeypairRequest{})
		return err
	},
		retry.Context(ctx),
		retry.Attempts(10),
		retry.Delay(2*time.Second),
		retry.DelayType(retry.FixedDelay),
		retry.OnRetry(func(attempt uint, err error) {
			p.t.Logf("JD health check attempt %d/10: %v", attempt+1, err)
		}),
	)

	if err != nil {
		return errors.New("JD service health check failed: service is not ready after retries")
	}

	p.t.Log("JD service health check passed: service is ready")

	return nil
}

// Name returns the name of the CTFOffchainProvider.
func (*CTFOffchainProvider) Name() string {
	return "Job Distributor CTF Offchain Provider"
}

// OffchainClient returns the JD client instance managed by this provider.
// You must call Initialize before using this method to ensure the client is properly set up.
func (p *CTFOffchainProvider) OffchainClient() offchain.Client {
	return p.client
}

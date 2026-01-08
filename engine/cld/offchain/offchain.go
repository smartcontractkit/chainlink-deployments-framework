package offchain

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"

	csav1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/csa"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/credentials"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	cfgenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	foffchain "github.com/smartcontractkit/chainlink-deployments-framework/offchain"
	fjd "github.com/smartcontractkit/chainlink-deployments-framework/offchain/jd"
	fjdprov "github.com/smartcontractkit/chainlink-deployments-framework/offchain/jd/provider"
)

// ErrEndpointsRequired is returned during loading of the offchain client when gRPC endpoint is
// required.
var ErrEndpointsRequired = errors.New("gRPC endpoint is required")

// loadConfig contains the configuration for loading an offchain client.
type loadConfig struct {
	// dryRun is true if the offchain client should perform read operations but no write operations.
	dryRun bool

	// logger is the logger for the offchain client.
	logger logger.Logger

	// creds is the gRPC transport credentials for the offchain client.
	creds credentials.TransportCredentials

	// provider is the offchain provider for the offchain client. Used for testing.
	provider foffchain.Provider

	// tokenSource is the oauth2 token source for the offchain client. Used for testing.
	tokenSource oauth2.TokenSource
}

// defaultLoadConfig creates a new loadConfig with default values.
func defaultLoadConfig() (*loadConfig, error) {
	lggr, err := logger.New()
	if err != nil {
		return nil, err
	}

	return &loadConfig{
		dryRun: false,
		logger: lggr,
		creds: credentials.NewTLS(&tls.Config{ // default to require TLS credentials
			MinVersion: tls.VersionTLS12,
		}),
	}, nil
}

// LoadOffchainClientOpt defines an option for configuring how the offchain client is loaded.
type LoadOffchainClientOpt func(*loadConfig)

// WithDryRun sets the dry run mode for the offchain client.
//
// When true, the offchain client will perform read operations but no write operations.
func WithDryRun(dryRun bool) LoadOffchainClientOpt {
	return func(c *loadConfig) {
		c.dryRun = dryRun
	}
}

// WithLogger sets the logger for the offchain client.
func WithLogger(lggr logger.Logger) LoadOffchainClientOpt {
	return func(c *loadConfig) {
		c.logger = lggr
	}
}

// WithCredentials sets the gRPC transport credentials for the offchain client.
func WithCredentials(creds credentials.TransportCredentials) LoadOffchainClientOpt {
	return func(c *loadConfig) {
		c.creds = creds
	}
}

// withOffchainProvider sets the offchain provider for the offchain client.
//
// Private function used only for testing.
func withOffchainProvider(provider foffchain.Provider) LoadOffchainClientOpt {
	return func(c *loadConfig) {
		c.provider = provider
	}
}

// withTokenSource sets the oauth2 token source for the offchain client.
//
// Private function used only for testing.
func withTokenSource(tokenSource oauth2.TokenSource) LoadOffchainClientOpt {
	return func(c *loadConfig) {
		c.tokenSource = tokenSource
	}
}

// LoadOffchainClient loads an offchain client for the specified domain and environment.
func LoadOffchainClient(
	ctx context.Context,
	dom domain.Domain,
	cfg cfgenv.JobDistributorConfig,
	opts ...LoadOffchainClientOpt,
) (foffchain.Client, error) {
	loadCfg, err := defaultLoadConfig()
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		opt(loadCfg)
	}

	var (
		lggr      = loadCfg.logger
		endpoints = cfg.Endpoints
		auth      = cfg.Auth
	)

	// TODO: Remove this domain specific check
	if dom.Key() == "keystone" && endpoints.GRPC == "" {
		lggr.Warn("Skipping JD initialization for Keystone, fallback to CLO data")

		return nil, nil //nolint:nilnil // We want to return nil if the JD is not initialized for now.
	}

	if endpoints.GRPC == "" {
		return nil, ErrEndpointsRequired
	}

	lggr.Info("Initializing JD client")

	// Setup the oauth2 token source.
	var oauth oauth2.TokenSource
	if auth != nil {
		if loadCfg.tokenSource != nil {
			oauth = loadCfg.tokenSource // Used for injecting a mock token source for testing.
		} else {
			oauth, err = newCognitoTokenSource(ctx, auth)
			if err != nil {
				return nil, err
			}
		}
	}

	// Load the provider and initialize the offchain client.
	var provOpts []fjdprov.ClientProviderOption
	if loadCfg.dryRun {
		lggr.Info("Using a dry-run JD client")

		provOpts = append(provOpts, fjdprov.WithDryRun(lggr))
	}

	var provider foffchain.Provider
	if loadCfg.provider != nil {
		provider = loadCfg.provider // Used for injecting a mock provider for testing.
	} else {
		provider = fjdprov.NewClientOffchainProvider(fjdprov.ClientOffchainProviderConfig{
			GRPC:  endpoints.GRPC,
			Creds: loadCfg.creds,
			Auth:  oauth,
		}, provOpts...)
	}

	jd, err := provider.Initialize(ctx)
	if err != nil {
		return nil, err
	}

	var kp *csav1.ListKeypairsResponse
	kp, err = jd.ListKeypairs(ctx, &csav1.ListKeypairsRequest{})
	if err != nil {
		return jd, fmt.Errorf("unable to reach the JD instance %s: %w", endpoints.GRPC, err)
	}
	lggr.Debugw("JD CSA Key", "key", kp.Keypairs[0].PublicKey)

	return jd, nil
}

// newCognitoTokenSource creates a new CognitoTokenSource for the given authentication configuration.
func newCognitoTokenSource(ctx context.Context, auth *cfgenv.JobDistributorAuth) (*fjd.CognitoTokenSource, error) {
	source := fjd.NewCognitoTokenSource(fjd.CognitoAuth{
		AppClientID:     auth.CognitoAppClientID,
		AppClientSecret: auth.CognitoAppClientSecret,
		Username:        auth.Username,
		Password:        auth.Password,
		AWSRegion:       auth.AWSRegion,
	})

	if err := source.Authenticate(ctx); err != nil {
		return nil, err
	}

	return source, nil
}

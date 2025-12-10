package offchain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials/insecure"

	csav1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/csa"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	cfgenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/offchain/internal/mocks"
)

func TestDefaultLoadConfig(t *testing.T) {
	t.Parallel()

	cfg, err := defaultLoadConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.False(t, cfg.dryRun)
	assert.NotNil(t, cfg.logger)
	assert.NotNil(t, cfg.creds)
	assert.Equal(t, "tls", cfg.creds.Info().SecurityProtocol)
}

func TestWithDryRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		dryRun bool
		want   bool
	}{
		{
			name:   "enable dry run",
			dryRun: true,
			want:   true,
		},
		{
			name:   "disable dry run",
			dryRun: false,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := defaultLoadConfig()
			require.NoError(t, err)

			// Apply the option
			opt := WithDryRun(tt.dryRun)
			opt(cfg)

			// Verify dry run setting
			assert.Equal(t, tt.want, cfg.dryRun)
		})
	}
}

func TestWithLogger(t *testing.T) {
	t.Parallel()

	cfg, err := defaultLoadConfig()
	require.NoError(t, err)

	// Store original logger for comparison
	originalLogger := cfg.logger

	// Create and apply custom logger
	customLogger := logger.Nop()
	opt := WithLogger(customLogger)
	opt(cfg)

	// Verify logger was changed
	assert.NotEqual(t, originalLogger, cfg.logger, "logger should be different from original")
	assert.Equal(t, customLogger, cfg.logger, "logger should be the custom logger")
}

func TestWithCredentials(t *testing.T) {
	t.Parallel()

	cfg, err := defaultLoadConfig()
	require.NoError(t, err)

	// Store original credentials for comparison
	originalCreds := cfg.creds

	// Create and apply custom credentials
	customCreds := insecure.NewCredentials()
	opt := WithCredentials(customCreds)
	opt(cfg)

	// Verify credentials were changed
	assert.NotEqual(t, originalCreds, cfg.creds, "credentials should be different from original")
	assert.Equal(t, customCreds, cfg.creds, "credentials should be the custom credentials")
	assert.Equal(t, "insecure", cfg.creds.Info().SecurityProtocol, "should be insecure credentials")
}

func TestLoadOffchainClient(t *testing.T) {
	t.Parallel()

	// Create test domains
	var (
		testDomain     = domain.NewDomain("/tmp", "test")
		keystoneDomain = domain.NewDomain("/tmp", "keystone")
		endpoints      = cfgenv.JobDistributorEndpoints{
			WSRPC: "ws://localhost:8080",
			GRPC:  "localhost:9090",
		}
	)

	tests := []struct {
		name       string
		beforeFunc func(*testing.T, *mocks.MockProvider, *mocks.MockClient)
		domain     domain.Domain
		cfg        cfgenv.JobDistributorConfig
		opts       []LoadOffchainClientOpt
		wantErr    string
		wantNil    bool
		want       any // The type of Offchain Client
	}{
		{
			name:   "valid config with all endpoints and auth",
			domain: testDomain,
			beforeFunc: func(t *testing.T, provider *mocks.MockProvider, client *mocks.MockClient) {
				t.Helper()

				provider.EXPECT().Initialize(t.Context()).Return(client, nil)
				client.EXPECT().ListKeypairs(t.Context(), &csav1.ListKeypairsRequest{}).Return(&csav1.ListKeypairsResponse{
					Keypairs: []*csav1.Keypair{
						{PublicKey: "test-public-key"},
					},
				}, nil)
			},
			cfg: cfgenv.JobDistributorConfig{
				Endpoints: endpoints,
				Auth: &cfgenv.JobDistributorAuth{
					CognitoAppClientID:     "test-client-id",
					CognitoAppClientSecret: "test-client-secret",
					Username:               "test-user",
					Password:               "test-pass",
					AWSRegion:              "us-west-2",
				},
			},
			opts: []LoadOffchainClientOpt{},
		},
		{
			name:   "valid config without auth",
			domain: testDomain,
			cfg: cfgenv.JobDistributorConfig{
				Endpoints: endpoints,
				Auth:      nil, // No auth
			},
			beforeFunc: func(t *testing.T, provider *mocks.MockProvider, client *mocks.MockClient) {
				t.Helper()

				provider.EXPECT().Initialize(t.Context()).Return(client, nil)
				client.EXPECT().ListKeypairs(t.Context(), &csav1.ListKeypairsRequest{}).Return(&csav1.ListKeypairsResponse{
					Keypairs: []*csav1.Keypair{
						{PublicKey: "test-public-key"},
					},
				}, nil)
			},
			opts: []LoadOffchainClientOpt{},
		},
		{
			name: "with dry run option",
			beforeFunc: func(t *testing.T, provider *mocks.MockProvider, client *mocks.MockClient) {
				t.Helper()

				provider.EXPECT().Initialize(t.Context()).Return(client, nil)
				client.EXPECT().ListKeypairs(t.Context(), &csav1.ListKeypairsRequest{}).Return(&csav1.ListKeypairsResponse{
					Keypairs: []*csav1.Keypair{
						{PublicKey: "test-public-key"},
					},
				}, nil)
			},
			domain: testDomain,
			cfg: cfgenv.JobDistributorConfig{
				Endpoints: cfgenv.JobDistributorEndpoints{
					WSRPC: "ws://localhost:8080",
					GRPC:  "localhost:9090",
				},
			},
			opts: []LoadOffchainClientOpt{
				WithDryRun(true),
			},
		},
		{
			name:   "missing WSRPC endpoint",
			domain: testDomain,
			cfg: cfgenv.JobDistributorConfig{
				Endpoints: cfgenv.JobDistributorEndpoints{
					WSRPC: "", // Missing
					GRPC:  "localhost:9090",
				},
			},
			opts:    []LoadOffchainClientOpt{},
			wantErr: "both gRPC and wsRPC endpoints are required",
		},
		{
			name:   "missing GRPC endpoint",
			domain: testDomain,
			cfg: cfgenv.JobDistributorConfig{
				Endpoints: cfgenv.JobDistributorEndpoints{
					WSRPC: "ws://localhost:8080",
					GRPC:  "", // Missing
				},
			},
			opts:    []LoadOffchainClientOpt{},
			wantErr: "both gRPC and wsRPC endpoints are required",
		},
		{
			name:   "keystone domain with missing WSRPC",
			domain: keystoneDomain,
			cfg: cfgenv.JobDistributorConfig{
				Endpoints: cfgenv.JobDistributorEndpoints{
					WSRPC: "", // Missing for keystone
					GRPC:  "localhost:9090",
				},
			},
			opts:    []LoadOffchainClientOpt{},
			wantNil: true,
		},
		{
			name: "provider initializer error",
			beforeFunc: func(t *testing.T, provider *mocks.MockProvider, client *mocks.MockClient) {
				t.Helper()

				provider.EXPECT().Initialize(t.Context()).Return(nil, errors.New("provider initializer error"))
			},
			domain: testDomain,
			cfg: cfgenv.JobDistributorConfig{
				Endpoints: endpoints,
			},
			opts:    []LoadOffchainClientOpt{},
			wantErr: "provider initializer error",
		},
		{
			name: "list keypairs error",
			beforeFunc: func(t *testing.T, provider *mocks.MockProvider, client *mocks.MockClient) {
				t.Helper()

				provider.EXPECT().Initialize(t.Context()).Return(client, nil)
				client.EXPECT().ListKeypairs(t.Context(), &csav1.ListKeypairsRequest{}).Return(nil, errors.New("list keypairs error"))
			},
			domain: testDomain,
			cfg: cfgenv.JobDistributorConfig{
				Endpoints: endpoints,
			},
			opts:    []LoadOffchainClientOpt{},
			wantErr: "list keypairs error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				provider    = mocks.NewMockProvider(t)
				client      = mocks.NewMockClient(t)
				tokenSource = mocks.NewMockTokenSource(t)
			)

			if tt.beforeFunc != nil {
				tt.beforeFunc(t, provider, client)
			}

			opts := append(tt.opts,
				withOffchainProvider(provider),
				withTokenSource(tokenSource),
			)

			got, err := LoadOffchainClient(t.Context(), tt.domain, tt.cfg, opts...)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			if tt.wantNil {
				require.Nil(t, got)
				require.NoError(t, err)
			}
		})
	}
}

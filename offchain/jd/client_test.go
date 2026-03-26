package jd

import (
	"testing"

	csav1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/csa"
	jobv1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/job"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/smartcontractkit/chainlink-deployments-framework/internal/testing/jd/mocks"
)

func TestNewJDClient_ConfigurationScenarios(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      JDConfig
		description string
	}{
		{
			name: "basic config with credentials",
			config: JDConfig{
				GRPC:  "localhost:9090",
				Creds: insecure.NewCredentials(),
			},
			description: "Basic configuration with insecure credentials",
		},
		{
			name: "config with OAuth2 auth",
			config: JDConfig{
				GRPC:  "localhost:9090",
				Creds: insecure.NewCredentials(),
				Auth:  oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
			},
			description: "Configuration with OAuth2 authentication",
		},
		{
			name: "complete config",
			config: JDConfig{
				GRPC:  "localhost:9090",
				Creds: insecure.NewCredentials(),
				Auth:  oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
			},
			description: "Complete configuration with all options",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, err := NewJDClient(tt.config)

			// gRPC connection creation typically succeeds even without server
			// The actual connection failure happens on first RPC call
			if err != nil {
				t.Logf("Connection failed for %s: %v", tt.description, err)
				assert.Contains(t, err.Error(), "failed to connect Job Distributor service")
			} else {
				require.NotNil(t, client, "Client should not be nil for %s", tt.description)

				// Verify fields are set correctly
				assert.NotNil(t, client.NodeServiceClient)
				assert.NotNil(t, client.JobServiceClient)
				assert.NotNil(t, client.CSAServiceClient)
			}
		})
	}
}

func TestJDClient_GetCSAPublicKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		beforeFunc func(t *testing.T, mockCSA *mocks.MockCSAServiceClient)
		want       string
		wantErr    string
	}{
		{
			name: "success with single keypair",
			beforeFunc: func(t *testing.T, mockCSA *mocks.MockCSAServiceClient) {
				t.Helper()

				mockCSA.EXPECT().ListKeypairs(
					t.Context(),
					&csav1.ListKeypairsRequest{},
				).Return(&csav1.ListKeypairsResponse{
					Keypairs: []*csav1.Keypair{
						{PublicKey: "test-public-key-123"},
					},
				}, nil)
			},
			want: "test-public-key-123",
		},
		{
			name: "success with multiple keypairs returns first",
			beforeFunc: func(t *testing.T, mockCSA *mocks.MockCSAServiceClient) {
				t.Helper()

				mockCSA.EXPECT().ListKeypairs(
					t.Context(),
					&csav1.ListKeypairsRequest{},
				).Return(&csav1.ListKeypairsResponse{
					Keypairs: []*csav1.Keypair{
						{PublicKey: "first-key"},
						{PublicKey: "second-key"},
					},
				}, nil)
			},
			want: "first-key",
		},
		{
			name: "error when ListKeypairs fails",
			beforeFunc: func(t *testing.T, mockCSA *mocks.MockCSAServiceClient) {
				t.Helper()

				mockCSA.EXPECT().ListKeypairs(
					t.Context(),
					&csav1.ListKeypairsRequest{},
				).Return(nil, assert.AnError)
			},
			wantErr: assert.AnError.Error(),
		},
		{
			name: "error when response is nil",
			beforeFunc: func(t *testing.T, mockCSA *mocks.MockCSAServiceClient) {
				t.Helper()

				mockCSA.EXPECT().ListKeypairs(
					t.Context(),
					&csav1.ListKeypairsRequest{},
				).Return(nil, nil)
			},
			wantErr: "no keypairs found",
		},
		{
			name: "error when keypairs slice is empty",
			beforeFunc: func(t *testing.T, mockCSA *mocks.MockCSAServiceClient) {
				t.Helper()

				mockCSA.EXPECT().ListKeypairs(
					t.Context(),
					&csav1.ListKeypairsRequest{},
				).Return(&csav1.ListKeypairsResponse{
					Keypairs: []*csav1.Keypair{},
				}, nil)
			},
			wantErr: "no keypairs found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create mock CSA service client
			mockClients := mocks.NewMockClients(t)
			tt.beforeFunc(t, mockClients.CSAServiceClient)

			// Create JobDistributor with mock
			jd := &JobDistributor{
				CSAServiceClient: mockClients.CSAServiceClient,
			}

			// Execute the method under test
			got, err := jd.GetCSAPublicKey(t.Context())

			// Assert results
			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
				assert.Empty(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestJDClient_ProposeJob(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		beforeFunc func(t *testing.T, mockJob *mocks.MockJobServiceClient, request *jobv1.ProposeJobRequest)
		request    *jobv1.ProposeJobRequest
		want       *jobv1.ProposeJobResponse
		wantErr    string
	}{
		{
			name: "success with valid proposal",
			beforeFunc: func(t *testing.T, mockJob *mocks.MockJobServiceClient, request *jobv1.ProposeJobRequest) {
				t.Helper()

				response := &jobv1.ProposeJobResponse{
					Proposal: &jobv1.Proposal{
						Id:    "proposal-123",
						JobId: "job-456",
						Spec:  "test-job-spec",
					},
				}

				mockJob.EXPECT().ProposeJob(
					t.Context(),
					request,
				).Return(response, nil)
			},
			request: &jobv1.ProposeJobRequest{
				NodeId: "test-node-123",
				Spec:   "test-job-spec",
			},
			want: &jobv1.ProposeJobResponse{
				Proposal: &jobv1.Proposal{
					Id:    "proposal-123",
					JobId: "job-456",
					Spec:  "test-job-spec",
				},
			},
		},
		{
			name: "error when ProposeJob service call fails",
			beforeFunc: func(t *testing.T, mockJob *mocks.MockJobServiceClient, request *jobv1.ProposeJobRequest) {
				t.Helper()

				mockJob.EXPECT().ProposeJob(
					t.Context(),
					request,
				).Return(nil, assert.AnError)
			},
			request: &jobv1.ProposeJobRequest{
				NodeId: "test-node-123",
				Spec:   "test-job-spec",
			},
			wantErr: "failed to propose job. err:",
		},
		{
			name: "error when proposal is nil in response",
			beforeFunc: func(t *testing.T, mockJob *mocks.MockJobServiceClient, request *jobv1.ProposeJobRequest) {
				t.Helper()

				response := &jobv1.ProposeJobResponse{
					Proposal: nil, // This should cause an error
				}

				mockJob.EXPECT().ProposeJob(
					t.Context(),
					request,
				).Return(response, nil)
			},
			request: &jobv1.ProposeJobRequest{
				NodeId: "test-node-123",
				Spec:   "test-job-spec",
			},
			wantErr: "failed to propose job. err: proposal is nil",
		},
		{
			name: "success with empty request",
			beforeFunc: func(t *testing.T, mockJob *mocks.MockJobServiceClient, request *jobv1.ProposeJobRequest) {
				t.Helper()

				response := &jobv1.ProposeJobResponse{
					Proposal: &jobv1.Proposal{
						Id: "empty-proposal-123",
					},
				}

				mockJob.EXPECT().ProposeJob(
					t.Context(),
					request,
				).Return(response, nil)
			},
			request: &jobv1.ProposeJobRequest{},
			want: &jobv1.ProposeJobResponse{
				Proposal: &jobv1.Proposal{
					Id: "empty-proposal-123",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create mock job service client
			mockClients := mocks.NewMockClients(t)
			tt.beforeFunc(t, mockClients.JobServiceClient, tt.request)

			// Create JobDistributor with mock
			jd := &JobDistributor{
				JobServiceClient: mockClients.JobServiceClient,
			}

			// Execute the method under test
			result, err := jd.ProposeJob(t.Context(), tt.request)

			// Assert results
			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

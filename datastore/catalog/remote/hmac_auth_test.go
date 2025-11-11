package remote

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	pb "github.com/smartcontractkit/chainlink-protos/op-catalog/v1/datastore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

// mockKMSClient is a mock implementation of the kmsClient interface for testing
type mockKMSClient struct {
	generateMacFunc func(ctx context.Context, params *kms.GenerateMacInput, optFns ...func(*kms.Options)) (*kms.GenerateMacOutput, error)
}

// GenerateMac implements the kmsClient interface
func (m *mockKMSClient) GenerateMac(ctx context.Context, params *kms.GenerateMacInput, optFns ...func(*kms.Options)) (*kms.GenerateMacOutput, error) {
	if m.generateMacFunc != nil {
		return m.generateMacFunc(ctx, params, optFns...)
	}

	return nil, errors.New("mock not configured")
}

func TestKmsHMACClientHelper_generateHMACSignature(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		keyID           string
		method          string
		authority       string
		payload         []byte
		mockGenerateMac func(ctx context.Context, params *kms.GenerateMacInput, optFns ...func(*kms.Options)) (*kms.GenerateMacOutput, error)
		wantSignature   string
		wantErr         bool
		wantErrContains string
	}{
		{
			name:      "successful signature generation",
			keyID:     "test-key-id",
			method:    "/op_catalog.v1.datastore.Datastore/DataAccess",
			authority: "catalog.example.com",
			payload:   []byte("test payload"),
			mockGenerateMac: func(ctx context.Context, params *kms.GenerateMacInput, optFns ...func(*kms.Options)) (*kms.GenerateMacOutput, error) {
				// Verify inputs
				assert.Equal(t, "test-key-id", *params.KeyId)
				assert.Equal(t, types.MacAlgorithmSpecHmacSha256, params.MacAlgorithm)
				assert.NotNil(t, params.Message)

				// Return mock MAC
				mockMac := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

				return &kms.GenerateMacOutput{
					Mac: mockMac,
				}, nil
			},
			wantSignature: "0102030405060708",
			wantErr:       false,
		},
		{
			name:      "large payload",
			keyID:     "test-key-id",
			method:    "/op_catalog.v1.datastore.Datastore/DataAccess",
			authority: "catalog.example.com",
			payload:   make([]byte, 10000), // 10KB payload
			mockGenerateMac: func(ctx context.Context, params *kms.GenerateMacInput, optFns ...func(*kms.Options)) (*kms.GenerateMacOutput, error) {
				// Verify the message length stays under KMS limits (4096 bytes) (should be hashed)
				// Message format: method\nauthority\ntimestamp\nsha256(payload)
				assert.Less(t, len(params.Message), 4096, "Message should be hashed to stay under KMS limits")
				return &kms.GenerateMacOutput{
					Mac: []byte{0xaa, 0xbb},
				}, nil
			},
			wantSignature: "aabb",
			wantErr:       false,
		},
		{
			name:      "kms error",
			keyID:     "test-key-id",
			method:    "/op_catalog.v1.datastore.Datastore/DataAccess",
			authority: "catalog.example.com",
			payload:   []byte("test payload"),
			mockGenerateMac: func(ctx context.Context, params *kms.GenerateMacInput, optFns ...func(*kms.Options)) (*kms.GenerateMacOutput, error) {
				return nil, errors.New("kms service unavailable")
			},
			wantErr:         true,
			wantErrContains: "failed to generate MAC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			mockClient := &mockKMSClient{
				generateMacFunc: tt.mockGenerateMac,
			}

			helper := &kmsHMACClientHelper{
				kmsClient: mockClient,
				keyID:     tt.keyID,
			}

			signature, timestamp, err := helper.generateHMACSignature(ctx, tt.method, tt.authority, tt.payload)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrContains != "" {
					assert.Contains(t, err.Error(), tt.wantErrContains)
				}
				assert.Empty(t, signature)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantSignature, signature)
				assert.NotEmpty(t, timestamp, "timestamp should not be empty")
				// Verify timestamp is a valid unix timestamp string
				assert.Regexp(t, `^\d+$`, timestamp, "timestamp should be numeric")
			}
		})
	}
}

func TestCatalogClient_prepareHMACContextWithClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		hmacConfig      *HMACAuthConfig
		request         *pb.DataAccessRequest
		existingMeta    metadata.MD
		mockGenerateMac func(ctx context.Context, params *kms.GenerateMacInput, optFns ...func(*kms.Options)) (*kms.GenerateMacOutput, error)
		wantErr         bool
		wantErrContains string
		validateResult  func(t *testing.T, ctx context.Context)
	}{
		{
			name: "successful context preparation with metadata",
			hmacConfig: &HMACAuthConfig{
				KeyID:     "test-key-id",
				KeyRegion: "us-west-2",
				Authority: "catalog.example.com",
			},
			request: &pb.DataAccessRequest{},
			mockGenerateMac: func(ctx context.Context, params *kms.GenerateMacInput, optFns ...func(*kms.Options)) (*kms.GenerateMacOutput, error) {
				mockMac := []byte{0x01, 0x02, 0x03, 0x04}
				return &kms.GenerateMacOutput{
					Mac: mockMac,
				}, nil
			},
			wantErr: false,
			validateResult: func(t *testing.T, ctx context.Context) {
				t.Helper()

				md, ok := metadata.FromOutgoingContext(ctx)
				require.True(t, ok, "metadata should be present")
				assert.NotEmpty(t, md.Get("x-hmac-signature"), "signature should be present")
				assert.NotEmpty(t, md.Get("x-hmac-timestamp"), "timestamp should be present")
				assert.Equal(t, "01020304", md.Get("x-hmac-signature")[0])
			},
		},
		{
			name: "merge with existing metadata",
			hmacConfig: &HMACAuthConfig{
				KeyID:     "test-key-id",
				KeyRegion: "us-west-2",
				Authority: "catalog.example.com",
			},
			request: &pb.DataAccessRequest{},
			existingMeta: metadata.Pairs(
				"existing-key", "existing-value",
				"another-key", "another-value",
			),
			mockGenerateMac: func(ctx context.Context, params *kms.GenerateMacInput, optFns ...func(*kms.Options)) (*kms.GenerateMacOutput, error) {
				mockMac := []byte{0xaa, 0xbb}
				return &kms.GenerateMacOutput{
					Mac: mockMac,
				}, nil
			},
			wantErr: false,
			validateResult: func(t *testing.T, ctx context.Context) {
				t.Helper()

				md, ok := metadata.FromOutgoingContext(ctx)
				require.True(t, ok, "metadata should be present")
				// Check existing metadata is preserved
				assert.Equal(t, "existing-value", md.Get("existing-key")[0])
				assert.Equal(t, "another-value", md.Get("another-key")[0])
				// Check new HMAC metadata is added
				assert.NotEmpty(t, md.Get("x-hmac-signature"))
				assert.NotEmpty(t, md.Get("x-hmac-timestamp"))
			},
		},
		{
			name: "kms error propagates",
			hmacConfig: &HMACAuthConfig{
				KeyID:     "test-key-id",
				KeyRegion: "us-west-2",
				Authority: "catalog.example.com",
			},
			request: &pb.DataAccessRequest{},
			mockGenerateMac: func(ctx context.Context, params *kms.GenerateMacInput, optFns ...func(*kms.Options)) (*kms.GenerateMacOutput, error) {
				return nil, errors.New("kms access denied")
			},
			wantErr:         true,
			wantErrContains: "failed to generate HMAC signature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			// Add existing metadata to context if provided
			if tt.existingMeta != nil {
				ctx = metadata.NewOutgoingContext(ctx, tt.existingMeta)
			}

			mockClient := &mockKMSClient{
				generateMacFunc: tt.mockGenerateMac,
			}

			catalogClient := &CatalogClient{
				hmacConfig: tt.hmacConfig,
			}

			resultCtx, err := catalogClient.prepareHMACContextWithClient(ctx, tt.request, mockClient)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrContains != "" {
					assert.Contains(t, err.Error(), tt.wantErrContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, resultCtx)
				if tt.validateResult != nil {
					tt.validateResult(t, resultCtx)
				}
			}
		})
	}
}

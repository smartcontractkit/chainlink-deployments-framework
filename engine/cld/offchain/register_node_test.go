package offchain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	nodev1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/node"
	"github.com/smartcontractkit/chainlink-protos/job-distributor/v1/shared/ptypes"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/internal/pointer"
	"github.com/smartcontractkit/chainlink-deployments-framework/internal/testing/jd/mocks"
)

func TestRegisterNode_Success_Bootstrap(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockClient := mocks.NewMockJDClient(t)
	testDomain := domain.NewDomain("/test", "ccip")

	// Expected request for bootstrap node
	expectedRequest := &nodev1.RegisterNodeRequest{
		Name:      "test-bootstrap-node",
		PublicKey: "test-csa-key-123",
		Labels: []*ptypes.Label{
			{
				Key:   "product",
				Value: pointer.To("ccip"),
			},
			{
				Key:   "environment",
				Value: pointer.To("testnet"),
			},
			{
				Key:   "region",
				Value: pointer.To("us-east-1"),
			},
			{
				Key:   "type",
				Value: pointer.To("bootstrap"),
			},
		},
	}

	// Mock successful response
	expectedResponse := &nodev1.RegisterNodeResponse{
		Node: &nodev1.Node{
			Id:        "node-123",
			Name:      "test-bootstrap-node",
			PublicKey: "test-csa-key-123",
		},
	}

	mockClient.MockNodeServiceClient.On("RegisterNode", ctx, expectedRequest).Return(expectedResponse, nil)

	nodeID, err := RegisterNode(
		ctx,
		mockClient,
		"test-bootstrap-node",
		"test-csa-key-123",
		true, // isBootstrap
		testDomain,
		"testnet",
		map[string]string{"region": "us-east-1"},
	)

	require.NoError(t, err)
	assert.Equal(t, "node-123", nodeID)

	mockClient.MockNodeServiceClient.AssertExpectations(t)
}

func TestRegisterNode_Success_Plugin(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockClient := mocks.NewMockJDClient(t)
	testDomain := domain.NewDomain("/test", "keystone")

	// Expected request for plugin node
	expectedRequest := &nodev1.RegisterNodeRequest{
		Name:      "test-plugin-node",
		PublicKey: "test-csa-key-456",
		Labels: []*ptypes.Label{
			{
				Key:   "product",
				Value: pointer.To("keystone"),
			},
			{
				Key:   "environment",
				Value: pointer.To("staging"),
			},
			{
				Key:   "datacenter",
				Value: pointer.To("dc1"),
			},
			{
				Key:   "tier",
				Value: pointer.To("premium"),
			},
			{
				Key:   "type",
				Value: pointer.To("plugin"),
			},
		},
	}

	// Mock successful response
	expectedResponse := &nodev1.RegisterNodeResponse{
		Node: &nodev1.Node{
			Id:        "node-456",
			Name:      "test-plugin-node",
			PublicKey: "test-csa-key-456",
		},
	}

	mockClient.MockNodeServiceClient.On("RegisterNode", ctx, expectedRequest).Return(expectedResponse, nil)

	nodeID, err := RegisterNode(
		ctx,
		mockClient,
		"test-plugin-node",
		"test-csa-key-456",
		false, // isBootstrap = false (plugin node)
		testDomain,
		"staging",
		map[string]string{
			"datacenter": "dc1",
			"tier":       "premium",
		},
	)

	require.NoError(t, err)
	assert.Equal(t, "node-456", nodeID)

	mockClient.MockNodeServiceClient.AssertExpectations(t)
}

func TestRegisterNode_Success_NoExtraLabels(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockClient := mocks.NewMockJDClient(t)
	testDomain := domain.NewDomain("/test", "ccip")

	// Expected request with only required labels
	expectedRequest := &nodev1.RegisterNodeRequest{
		Name:      "minimal-node",
		PublicKey: "minimal-csa-key",
		Labels: []*ptypes.Label{
			{
				Key:   "product",
				Value: pointer.To("ccip"),
			},
			{
				Key:   "environment",
				Value: pointer.To("mainnet"),
			},
			{
				Key:   "type",
				Value: pointer.To("plugin"),
			},
		},
	}

	// Mock successful response
	expectedResponse := &nodev1.RegisterNodeResponse{
		Node: &nodev1.Node{
			Id:        "node-minimal",
			Name:      "minimal-node",
			PublicKey: "minimal-csa-key",
		},
	}

	mockClient.MockNodeServiceClient.On("RegisterNode", ctx, expectedRequest).Return(expectedResponse, nil)

	nodeID, err := RegisterNode(
		ctx,
		mockClient,
		"minimal-node",
		"minimal-csa-key",
		false, // plugin node
		testDomain,
		"mainnet",
		nil, // no extra labels
	)

	require.NoError(t, err)
	assert.Equal(t, "node-minimal", nodeID)

	mockClient.MockNodeServiceClient.AssertExpectations(t)
}

func TestRegisterNode_ClientError(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockClient := mocks.NewMockJDClient(t)
	testDomain := domain.NewDomain("/test", "ccip")

	// Mock client error
	expectedError := errors.New("registration failed: node already exists")
	mockClient.MockNodeServiceClient.On("RegisterNode", ctx, mock.AnythingOfType("*node.RegisterNodeRequest")).Return(nil, expectedError)

	nodeID, err := RegisterNode(
		ctx,
		mockClient,
		"duplicate-node",
		"duplicate-csa-key",
		false,
		testDomain,
		"testnet",
		nil,
	)

	require.Error(t, err)
	assert.Empty(t, nodeID)
	assert.Contains(t, err.Error(), "failed to register node duplicate-node")
	assert.Contains(t, err.Error(), "registration failed: node already exists")

	mockClient.MockNodeServiceClient.AssertExpectations(t)
}

func TestRegisterNode_NilResponse(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockClient := mocks.NewMockJDClient(t)
	testDomain := domain.NewDomain("/test", "ccip")

	// Mock nil response
	mockClient.MockNodeServiceClient.On("RegisterNode", ctx, mock.AnythingOfType("*node.RegisterNodeRequest")).Return(nil, nil)

	nodeID, err := RegisterNode(
		ctx,
		mockClient,
		"nil-response-node",
		"nil-response-csa-key",
		false,
		testDomain,
		"testnet",
		nil,
	)

	require.Error(t, err)
	assert.Empty(t, nodeID)
	assert.Contains(t, err.Error(), "failed to register node nil-response-node, blank response received")

	mockClient.MockNodeServiceClient.AssertExpectations(t)
}

func TestRegisterNode_NilNodeInResponse(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockClient := mocks.NewMockJDClient(t)
	testDomain := domain.NewDomain("/test", "ccip")

	// Mock response with nil node
	response := &nodev1.RegisterNodeResponse{
		Node: nil,
	}
	mockClient.MockNodeServiceClient.On("RegisterNode", ctx, mock.AnythingOfType("*node.RegisterNodeRequest")).Return(response, nil)

	nodeID, err := RegisterNode(
		ctx,
		mockClient,
		"nil-node-response",
		"nil-node-csa-key",
		false,
		testDomain,
		"testnet",
		nil,
	)

	require.Error(t, err)
	assert.Empty(t, nodeID)
	assert.Contains(t, err.Error(), "failed to register node nil-node-response, blank response received")

	mockClient.MockNodeServiceClient.AssertExpectations(t)
}

func TestRegisterNode_EmptyNodeIDInResponse(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockClient := mocks.NewMockJDClient(t)
	testDomain := domain.NewDomain("/test", "ccip")

	// Mock response with empty node ID
	response := &nodev1.RegisterNodeResponse{
		Node: &nodev1.Node{
			Id:        "", // empty ID
			Name:      "empty-id-node",
			PublicKey: "empty-id-csa-key",
		},
	}
	mockClient.MockNodeServiceClient.On("RegisterNode", ctx, mock.AnythingOfType("*node.RegisterNodeRequest")).Return(response, nil)

	nodeID, err := RegisterNode(
		ctx,
		mockClient,
		"empty-id-node",
		"empty-id-csa-key",
		false,
		testDomain,
		"testnet",
		nil,
	)

	require.Error(t, err)
	assert.Empty(t, nodeID)
	assert.Contains(t, err.Error(), "failed to register node empty-id-node, blank response received")

	mockClient.MockNodeServiceClient.AssertExpectations(t)
}

func TestRegisterNode_BootstrapVsPluginLabels(t *testing.T) {
	t.Parallel()

	testDomain := domain.NewDomain("/test", "ccip")

	tests := []struct {
		name         string
		isBootstrap  bool
		expectedType string
	}{
		{
			name:         "bootstrap node",
			isBootstrap:  true,
			expectedType: "bootstrap",
		},
		{
			name:         "plugin node",
			isBootstrap:  false,
			expectedType: "plugin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			mockClient := mocks.NewMockJDClient(t)

			// Capture the actual request to verify the type label
			var capturedRequest *nodev1.RegisterNodeRequest
			mockClient.MockNodeServiceClient.On("RegisterNode", ctx, mock.AnythingOfType("*node.RegisterNodeRequest")).
				Run(func(args mock.Arguments) {
					capturedRequest = args.Get(1).(*nodev1.RegisterNodeRequest)
				}).
				Return(&nodev1.RegisterNodeResponse{
					Node: &nodev1.Node{
						Id:        "test-node-id",
						Name:      "test-node",
						PublicKey: "test-csa-key",
					},
				}, nil)

			_, err := RegisterNode(
				ctx,
				mockClient,
				"test-node",
				"test-csa-key",
				tt.isBootstrap,
				testDomain,
				"testnet",
				nil,
			)

			require.NoError(t, err)

			// Verify the type label is set correctly
			var typeLabel *ptypes.Label
			for _, label := range capturedRequest.Labels {
				if label.Key == "type" {
					typeLabel = label
					break
				}
			}

			require.NotNil(t, typeLabel, "type label should be present")
			assert.Equal(t, tt.expectedType, *typeLabel.Value)

			mockClient.MockNodeServiceClient.AssertExpectations(t)
		})
	}
}

func TestRegisterNode_Labels(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockClient := mocks.NewMockJDClient(t)
	testDomain := domain.NewDomain("/test", "automation")

	// Capture the actual request to verify label structure
	var capturedRequest *nodev1.RegisterNodeRequest
	mockClient.MockNodeServiceClient.On("RegisterNode", ctx, mock.AnythingOfType("*node.RegisterNodeRequest")).
		Run(func(args mock.Arguments) {
			capturedRequest = args.Get(1).(*nodev1.RegisterNodeRequest)
		}).
		Return(&nodev1.RegisterNodeResponse{
			Node: &nodev1.Node{
				Id:        "test-node-id",
				Name:      "test-node",
				PublicKey: "test-csa-key",
			},
		}, nil)

	extraLabels := map[string]string{
		"zone":     "zone-a",
		"version":  "v1.2.3",
		"priority": "high",
	}

	_, err := RegisterNode(
		ctx,
		mockClient,
		"test-node",
		"test-csa-key",
		true, // bootstrap
		testDomain,
		"production",
		extraLabels,
	)

	require.NoError(t, err)

	// Verify all expected labels are present
	labelMap := make(map[string]string)
	for _, label := range capturedRequest.Labels {
		labelMap[label.Key] = *label.Value
	}

	expectedLabels := map[string]string{
		"product":     "automation",
		"environment": "production",
		"zone":        "zone-a",
		"version":     "v1.2.3",
		"priority":    "high",
		"type":        "bootstrap",
	}

	assert.Equal(t, expectedLabels, labelMap)

	mockClient.MockNodeServiceClient.AssertExpectations(t)
}

func TestRegisterNode_EmptyExtraLabelsMap(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockClient := mocks.NewMockJDClient(t)
	testDomain := domain.NewDomain("/test", "ccip")

	// Capture the actual request to verify labels
	var capturedRequest *nodev1.RegisterNodeRequest
	mockClient.MockNodeServiceClient.On("RegisterNode", ctx, mock.AnythingOfType("*node.RegisterNodeRequest")).
		Run(func(args mock.Arguments) {
			capturedRequest = args.Get(1).(*nodev1.RegisterNodeRequest)
		}).
		Return(&nodev1.RegisterNodeResponse{
			Node: &nodev1.Node{
				Id:        "test-node-id",
				Name:      "test-node",
				PublicKey: "test-csa-key",
			},
		}, nil)

	_, err := RegisterNode(
		ctx,
		mockClient,
		"test-node",
		"test-csa-key",
		false, // plugin
		testDomain,
		"testnet",
		map[string]string{}, // empty map
	)

	require.NoError(t, err)

	// Verify only required labels are present
	labelMap := make(map[string]string)
	for _, label := range capturedRequest.Labels {
		labelMap[label.Key] = *label.Value
	}

	expectedLabels := map[string]string{
		"product":     "ccip",
		"environment": "testnet",
		"type":        "plugin",
	}

	assert.Equal(t, expectedLabels, labelMap)

	mockClient.MockNodeServiceClient.AssertExpectations(t)
}

func TestRegisterNode_DifferentDomains(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	testCases := []struct {
		name      string
		domainKey string
		expectKey string
	}{
		{
			name:      "ccip domain",
			domainKey: "ccip",
			expectKey: "ccip",
		},
		{
			name:      "keystone domain",
			domainKey: "keystone",
			expectKey: "keystone",
		},
		{
			name:      "automation domain",
			domainKey: "automation",
			expectKey: "automation",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockClient := mocks.NewMockJDClient(t)
			testDomain := domain.NewDomain("/test", tc.domainKey)

			// Capture the actual request to verify domain key
			var capturedRequest *nodev1.RegisterNodeRequest
			mockClient.MockNodeServiceClient.On("RegisterNode", ctx, mock.AnythingOfType("*node.RegisterNodeRequest")).
				Run(func(args mock.Arguments) {
					capturedRequest = args.Get(1).(*nodev1.RegisterNodeRequest)
				}).
				Return(&nodev1.RegisterNodeResponse{
					Node: &nodev1.Node{
						Id:        "test-node-id",
						Name:      "test-node",
						PublicKey: "test-csa-key",
					},
				}, nil)

			_, err := RegisterNode(
				ctx,
				mockClient,
				"test-node",
				"test-csa-key",
				false,
				testDomain,
				"testnet",
				nil,
			)

			require.NoError(t, err)

			// Find the product label
			var productLabel *ptypes.Label
			for _, label := range capturedRequest.Labels {
				if label.Key == "product" {
					productLabel = label
					break
				}
			}

			require.NotNil(t, productLabel, "product label should be present")
			assert.Equal(t, tc.expectKey, *productLabel.Value)

			mockClient.MockNodeServiceClient.AssertExpectations(t)
		})
	}
}

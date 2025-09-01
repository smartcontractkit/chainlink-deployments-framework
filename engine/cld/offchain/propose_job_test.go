package offchain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	jobv1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/job"
	nodev1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/node"
	"github.com/smartcontractkit/chainlink-protos/job-distributor/v1/shared/ptypes"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/internal/pointer"
	"github.com/smartcontractkit/chainlink-deployments-framework/internal/testing/jd/mocks"
)

func TestProposeJobRequest_Validate(t *testing.T) {
	t.Parallel()

	mockClient := mocks.NewMockJDClient(t)
	mockLogger := logger.Test(t)
	testDomain := domain.NewDomain("/test", "ccip")

	tests := []struct {
		name    string
		req     ProposeJobRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			req: ProposeJobRequest{
				Job:            "test job spec",
				Domain:         testDomain,
				Environment:    "testnet",
				NodeLabels:     map[string]string{"region": "us-east-1"},
				JobLabels:      map[string]string{"version": "1.0"},
				OffchainClient: mockClient,
				Lggr:           mockLogger,
			},
			wantErr: false,
		},
		{
			name: "empty job",
			req: ProposeJobRequest{
				Job:            "",
				Domain:         testDomain,
				Environment:    "testnet",
				OffchainClient: mockClient,
				Lggr:           mockLogger,
			},
			wantErr: true,
			errMsg:  "job is empty",
		},
		{
			name: "empty domain",
			req: ProposeJobRequest{
				Job:            "test job spec",
				Domain:         domain.NewDomain("/test", ""),
				Environment:    "testnet",
				OffchainClient: mockClient,
				Lggr:           mockLogger,
			},
			wantErr: true,
			errMsg:  "domain is empty",
		},
		{
			name: "empty environment",
			req: ProposeJobRequest{
				Job:            "test job spec",
				Domain:         testDomain,
				Environment:    "",
				OffchainClient: mockClient,
				Lggr:           mockLogger,
			},
			wantErr: true,
			errMsg:  "environment is empty",
		},
		{
			name: "nil offchain client",
			req: ProposeJobRequest{
				Job:            "test job spec",
				Domain:         testDomain,
				Environment:    "testnet",
				OffchainClient: nil,
				Lggr:           mockLogger,
			},
			wantErr: true,
			errMsg:  "offchain client is nil",
		},
		{
			name: "nil logger",
			req: ProposeJobRequest{
				Job:            "test job spec",
				Domain:         testDomain,
				Environment:    "testnet",
				OffchainClient: mockClient,
				Lggr:           nil,
			},
			wantErr: true,
			errMsg:  "logger is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.req.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestProposeJob_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockClient := mocks.NewMockJDClient(t)
	mockLogger := logger.Test(t)
	testDomain := domain.NewDomain("/test", "ccip")

	// Mock nodes response
	nodes := []*nodev1.Node{
		{
			Id:   "node-1",
			Name: "test-node-1",
		},
		{
			Id:   "node-2",
			Name: "test-node-2",
		},
	}
	listNodesResponse := &nodev1.ListNodesResponse{
		Nodes: nodes,
	}

	// Set up expectations
	expectedFilter := &nodev1.ListNodesRequest_Filter{
		Enabled: 1,
		Selectors: []*ptypes.Selector{
			{
				Key:   "product",
				Op:    ptypes.SelectorOp_EQ,
				Value: pointer.To("ccip"),
			},
			{
				Key:   "environment",
				Op:    ptypes.SelectorOp_EQ,
				Value: pointer.To("testnet"),
			},
			{
				Key:   "region",
				Op:    ptypes.SelectorOp_EQ,
				Value: pointer.To("us-east-1"),
			},
		},
	}

	mockClient.MockNodeServiceClient.On("ListNodes", ctx, &nodev1.ListNodesRequest{Filter: expectedFilter}).Return(listNodesResponse, nil)

	// Mock successful job proposals
	proposeJobResponse := &jobv1.ProposeJobResponse{
		Proposal: &jobv1.Proposal{
			Id: "proposal-1",
		},
	}

	expectedJobLabels := []*ptypes.Label{
		{
			Key:   "version",
			Value: pointer.To("1.0"),
		},
	}

	mockClient.MockJobServiceClient.On("ProposeJob", ctx, &jobv1.ProposeJobRequest{
		NodeId: "node-1",
		Spec:   "test job spec",
		Labels: expectedJobLabels,
	}).Return(proposeJobResponse, nil)

	mockClient.MockJobServiceClient.On("ProposeJob", ctx, &jobv1.ProposeJobRequest{
		NodeId: "node-2",
		Spec:   "test job spec",
		Labels: expectedJobLabels,
	}).Return(proposeJobResponse, nil)

	req := ProposeJobRequest{
		Job:            "test job spec",
		Domain:         testDomain,
		Environment:    "testnet",
		NodeLabels:     map[string]string{"region": "us-east-1"},
		JobLabels:      map[string]string{"version": "1.0"},
		OffchainClient: mockClient,
		Lggr:           mockLogger,
	}

	err := ProposeJob(ctx, req)
	require.NoError(t, err)

	mockClient.MockJobServiceClient.AssertExpectations(t)
	mockClient.MockNodeServiceClient.AssertExpectations(t)
}

func TestProposeJob_ListNodesError(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockClient := mocks.NewMockJDClient(t)
	mockLogger := logger.Test(t)
	testDomain := domain.NewDomain("/test", "ccip")

	// Mock ListNodes to return an error
	expectedError := errors.New("failed to list nodes")
	mockClient.MockNodeServiceClient.On("ListNodes", ctx, mock.AnythingOfType("*node.ListNodesRequest")).Return(nil, expectedError)

	req := ProposeJobRequest{
		Job:            "test job spec",
		Domain:         testDomain,
		Environment:    "testnet",
		OffchainClient: mockClient,
		Lggr:           mockLogger,
	}

	err := ProposeJob(ctx, req)
	require.Error(t, err)
	assert.Equal(t, expectedError, err)

	mockClient.MockNodeServiceClient.AssertExpectations(t)
}

func TestProposeJob_PartialFailure(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockClient := mocks.NewMockJDClient(t)
	mockLogger := logger.Test(t)
	testDomain := domain.NewDomain("/test", "ccip")

	// Mock nodes response
	nodes := []*nodev1.Node{
		{
			Id:   "node-1",
			Name: "test-node-1",
		},
		{
			Id:   "node-2",
			Name: "test-node-2",
		},
	}
	listNodesResponse := &nodev1.ListNodesResponse{
		Nodes: nodes,
	}

	mockClient.MockNodeServiceClient.On("ListNodes", ctx, mock.AnythingOfType("*node.ListNodesRequest")).Return(listNodesResponse, nil)

	// Mock first job proposal to succeed
	proposeJobResponse := &jobv1.ProposeJobResponse{
		Proposal: &jobv1.Proposal{
			Id: "proposal-1",
		},
	}

	mockClient.MockJobServiceClient.On("ProposeJob", ctx, &jobv1.ProposeJobRequest{
		NodeId: "node-1",
		Spec:   "test job spec",
		Labels: []*ptypes.Label{},
	}).Return(proposeJobResponse, nil)

	// Mock second job proposal to fail
	proposeJobError := errors.New("failed to propose job")
	mockClient.MockJobServiceClient.On("ProposeJob", ctx, &jobv1.ProposeJobRequest{
		NodeId: "node-2",
		Spec:   "test job spec",
		Labels: []*ptypes.Label{},
	}).Return(nil, proposeJobError)

	req := ProposeJobRequest{
		Job:            "test job spec",
		Domain:         testDomain,
		Environment:    "testnet",
		OffchainClient: mockClient,
		Lggr:           mockLogger,
	}

	err := ProposeJob(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error proposing job to node node-2")
	assert.Contains(t, err.Error(), "failed to propose job")

	mockClient.MockJobServiceClient.AssertExpectations(t)
	mockClient.MockNodeServiceClient.AssertExpectations(t)
}

func TestProposeJob_InvalidRequest(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	req := ProposeJobRequest{
		Job: "", // Invalid: empty job
	}

	err := ProposeJob(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid request")
	assert.Contains(t, err.Error(), "job is empty")
}

func TestProposeJob_EmptyNodesList(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockClient := mocks.NewMockJDClient(t)
	mockLogger := logger.Test(t)
	testDomain := domain.NewDomain("/test", "ccip")

	// Mock empty nodes response
	listNodesResponse := &nodev1.ListNodesResponse{
		Nodes: []*nodev1.Node{},
	}

	mockClient.MockNodeServiceClient.On("ListNodes", ctx, mock.AnythingOfType("*node.ListNodesRequest")).Return(listNodesResponse, nil)

	req := ProposeJobRequest{
		Job:            "test job spec",
		Domain:         testDomain,
		Environment:    "testnet",
		OffchainClient: mockClient,
		Lggr:           mockLogger,
	}

	err := ProposeJob(ctx, req)
	require.NoError(t, err) // Should succeed with no nodes to propose to

	mockClient.MockNodeServiceClient.AssertExpectations(t)
}

func TestConvertLabels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  map[string]string
		expect []*ptypes.Label
	}{
		{
			name:   "empty map",
			input:  map[string]string{},
			expect: []*ptypes.Label{},
		},
		{
			name: "single label",
			input: map[string]string{
				"key1": "value1",
			},
			expect: []*ptypes.Label{
				{
					Key:   "key1",
					Value: pointer.To("value1"),
				},
			},
		},
		{
			name: "multiple labels",
			input: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			expect: []*ptypes.Label{
				{
					Key:   "key1",
					Value: pointer.To("value1"),
				},
				{
					Key:   "key2",
					Value: pointer.To("value2"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := convertLabels(tt.input)

			// Since map iteration order is not guaranteed, we need to compare by content
			require.Len(t, result, len(tt.expect))

			resultMap := make(map[string]string)
			for _, label := range result {
				resultMap[label.Key] = *label.Value
			}

			expectedMap := make(map[string]string)
			for _, label := range tt.expect {
				expectedMap[label.Key] = *label.Value
			}

			assert.Equal(t, expectedMap, resultMap)
		})
	}
}

func TestProposeJob_WithComplexSelectors(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockClient := mocks.NewMockJDClient(t)
	mockLogger := logger.Test(t)
	testDomain := domain.NewDomain("/test", "keystone")

	nodes := []*nodev1.Node{
		{
			Id:   "node-1",
			Name: "test-node-1",
		},
	}
	listNodesResponse := &nodev1.ListNodesResponse{
		Nodes: nodes,
	}

	// Use mock.MatchedBy to handle selector ordering since maps don't guarantee iteration order
	mockClient.MockNodeServiceClient.On("ListNodes", ctx, mock.MatchedBy(func(req *nodev1.ListNodesRequest) bool {
		if req.Filter == nil || req.Filter.Enabled != 1 {
			return false
		}

		// Convert selectors to map for comparison
		selectorMap := make(map[string]string)
		for _, selector := range req.Filter.Selectors {
			if selector.Value != nil {
				selectorMap[selector.Key] = *selector.Value
			}
		}

		// Check expected selectors
		expectedSelectors := map[string]string{
			"product":     "keystone",
			"environment": "staging",
			"region":      "us-west-2",
			"type":        "plugin",
		}

		if len(selectorMap) != len(expectedSelectors) {
			return false
		}

		for key, expectedValue := range expectedSelectors {
			if actualValue, exists := selectorMap[key]; !exists || actualValue != expectedValue {
				return false
			}
		}

		return true
	})).Return(listNodesResponse, nil)

	proposeJobResponse := &jobv1.ProposeJobResponse{
		Proposal: &jobv1.Proposal{
			Id: "proposal-1",
		},
	}

	// Use mock.MatchedBy to handle label ordering since maps don't guarantee iteration order
	mockClient.MockJobServiceClient.On("ProposeJob", ctx, mock.MatchedBy(func(req *jobv1.ProposeJobRequest) bool {
		if req.NodeId != "node-1" || req.Spec != "complex job spec" {
			return false
		}

		// Convert labels to map for comparison
		labelMap := make(map[string]string)
		for _, label := range req.Labels {
			labelMap[label.Key] = *label.Value
		}

		expectedLabels := map[string]string{
			"team":     "platform",
			"priority": "high",
		}

		if len(labelMap) != len(expectedLabels) {
			return false
		}

		for key, expectedValue := range expectedLabels {
			if actualValue, exists := labelMap[key]; !exists || actualValue != expectedValue {
				return false
			}
		}

		return true
	})).Return(proposeJobResponse, nil)

	req := ProposeJobRequest{
		Job:         "complex job spec",
		Domain:      testDomain,
		Environment: "staging",
		NodeLabels: map[string]string{
			"region": "us-west-2",
			"type":   "plugin",
		},
		JobLabels: map[string]string{
			"team":     "platform",
			"priority": "high",
		},
		OffchainClient: mockClient,
		Lggr:           mockLogger,
	}

	err := ProposeJob(ctx, req)
	require.NoError(t, err)

	mockClient.MockJobServiceClient.AssertExpectations(t)
	mockClient.MockNodeServiceClient.AssertExpectations(t)
}

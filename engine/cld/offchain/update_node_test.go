package offchain

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	nodev1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/node"
	"github.com/smartcontractkit/chainlink-protos/job-distributor/v1/shared/ptypes"

	fpointer "github.com/smartcontractkit/chainlink-deployments-framework/internal/pointer"
	jdmocks "github.com/smartcontractkit/chainlink-deployments-framework/internal/testing/jd/mocks"
	fnode "github.com/smartcontractkit/chainlink-deployments-framework/offchain/node"
)

func TestGetNode(t *testing.T) {
	t.Parallel()

	var (
		node1 = &nodev1.Node{
			Id:        "1",
			Name:      "node1",
			PublicKey: "csa_key1",
			Labels: plabels(map[string]string{
				"p2p_id": "p2p_id1",
			}),
		}
		node2 = &nodev1.Node{
			Id:        "2",
			Name:      "node2",
			PublicKey: "csa_key2",
			Labels: plabels(map[string]string{
				"p2p_id": "p2p_id2",
			}),
		}
		resp = &nodev1.ListNodesResponse{
			Nodes: []*nodev1.Node{
				node1, node2,
			},
		}
	)
	type args struct {
		resp    *nodev1.ListNodesResponse
		keyType NodeKey
		key     *string
		value   string
	}
	tests := []struct {
		name    string
		args    args
		want    *nodev1.Node
		wantErr bool
	}{
		{
			name: "id",
			args: args{
				resp:    resp,
				keyType: NodeKey_ID,
				value:   "1",
			},
			want: node1,
		},
		{
			name: "non existent id",
			args: args{
				resp:    resp,
				keyType: NodeKey_ID,
				value:   "not_an_id",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "name",
			args: args{
				resp:    resp,
				keyType: NodeKey_Name,
				value:   "node2",
			},
			want: node2,
		},
		{
			name: "non existent name",
			args: args{
				resp:    resp,
				keyType: NodeKey_Name,
				value:   "not_a_name",
			},
			want:    nil,
			wantErr: true,
		},

		{
			name: "csa key",
			args: args{
				resp:    resp,
				keyType: NodeKey_CSAKey,
				value:   "csa_key1",
			},
			want: node1,
		},
		{
			name: "non existent csa key",
			args: args{
				resp:    resp,
				keyType: NodeKey_CSAKey,
				value:   "not_a_csa_key",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "p2p id label",
			args: args{
				resp:    resp,
				keyType: NodeKey_Label,
				key:     fpointer.To("p2p_id"),
				value:   "p2p_id2",
			},
			want: node2,
		},
		{
			name: "missing label",
			args: args{
				resp:    resp,
				keyType: NodeKey_Label,
				key:     fpointer.To("not_a_label"),
				value:   "foo",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "label with wrong value",
			args: args{
				resp:    resp,
				keyType: NodeKey_Label,
				key:     fpointer.To("p2p_id"),
				value:   "not_a_p2p_id",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := getNode(tt.args.resp, tt.args.keyType, tt.args.value, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetNode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func plabels(m map[string]string) []*ptypes.Label {
	labels := make([]*ptypes.Label, 0, len(m))
	for k, v := range m {
		labels = append(labels, &ptypes.Label{
			Key:   k,
			Value: &v,
		})
	}

	return labels
}

func TestNodeFinderCfg_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     NodeFinderCfg
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid id key type",
			cfg: NodeFinderCfg{
				KeyType: NodeKey_ID,
			},
			wantErr: false,
		},
		{
			name: "valid label key type with label name",
			cfg: NodeFinderCfg{
				KeyType:   NodeKey_Label,
				LabelName: fpointer.To("p2p_id"),
			},
			wantErr: false,
		},
		{
			name: "invalid label key type without label name",
			cfg: NodeFinderCfg{
				KeyType:   NodeKey_Label,
				LabelName: nil,
			},
			wantErr: true,
			errMsg:  "label name is required for label search",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewUpdateNodeRequest(t *testing.T) {
	t.Parallel()

	validNodeCfg := fnode.NodeCfg{
		MinimalNodeCfg: fnode.MinimalNodeCfg{
			Name:                "test-node",
			CSAKey:              "test-csa-key",
			NOP:                 "test-nop",
			EncryptionPublicKey: "test-encryption-key",
		},
		P2PID:     "test-p2p-id",
		AdminAddr: "0x1234567890123456789012345678901234567890",
		Tags: map[string]string{
			"region": "us-east-1",
			"env":    "test",
		},
	}

	tests := []struct {
		name        string
		cfg         fnode.NodeCfg
		finderCfg   NodeFinderCfg
		wantErr     bool
		errContains string
	}{
		{
			name: "valid request with name key type",
			cfg:  validNodeCfg,
			finderCfg: NodeFinderCfg{
				KeyType: NodeKey_Name,
			},
			wantErr: false,
		},
		{
			name: "valid request with csa key type",
			cfg:  validNodeCfg,
			finderCfg: NodeFinderCfg{
				KeyType: NodeKey_CSAKey,
			},
			wantErr: false,
		},
		{
			name: "valid request with label key type",
			cfg:  validNodeCfg,
			finderCfg: NodeFinderCfg{
				KeyType:   NodeKey_Label,
				LabelName: fpointer.To("region"),
			},
			wantErr: false,
		},
		{
			name: "invalid finder config - label without label name",
			cfg:  validNodeCfg,
			finderCfg: NodeFinderCfg{
				KeyType:   NodeKey_Label,
				LabelName: nil,
			},
			wantErr:     true,
			errContains: "invalid node finder config",
		},
		{
			name: "invalid finder config - id key type",
			cfg:  validNodeCfg,
			finderCfg: NodeFinderCfg{
				KeyType: NodeKey_ID,
			},
			wantErr:     true,
			errContains: "id key type is not supported",
		},
		{
			name: "valid request with nonexistent label - gets empty value",
			cfg:  validNodeCfg,
			finderCfg: NodeFinderCfg{
				KeyType:   NodeKey_Label,
				LabelName: fpointer.To("nonexistent"),
			},
			wantErr: false, // This doesn't error, it just gets empty string
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req, err := NewUpdateNodeRequest(tt.cfg, tt.finderCfg)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, req)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, req)
				assert.Equal(t, tt.cfg, req.Cfg)
			}
		})
	}
}

func TestUpdateNodeRequest_Labels(t *testing.T) {
	t.Parallel()

	nodeCfg := fnode.NodeCfg{
		MinimalNodeCfg: fnode.MinimalNodeCfg{
			Name:                "test-node",
			CSAKey:              "test-csa-key",
			NOP:                 "Test NOP",
			EncryptionPublicKey: "test-encryption-key",
		},
		P2PID:        "test-p2p-id",
		AdminAddr:    "0x1234567890123456789012345678901234567890",
		MultiAddress: fpointer.To("127.0.0.1:8080"),
		Tags: map[string]string{
			"region": "us-east-1",
			"env":    "test",
		},
	}

	req, err := NewUpdateNodeRequest(nodeCfg, NodeFinderCfg{KeyType: NodeKey_Name})
	require.NoError(t, err)

	labels := req.Labels()

	// Convert to map for easier testing
	labelMap := make(map[string]string)
	for _, label := range labels {
		labelMap[label.Key] = *label.Value
	}

	expectedLabels := map[string]string{
		"p2p_id":        "test-p2p-id",
		"nop":           "Test_NOP", // spaces replaced with underscores
		"admin_addr":    "0x1234567890123456789012345678901234567890",
		"multi_address": "127.0.0.1:8080",
		"region":        "us-east-1",
		"env":           "test",
	}

	assert.Equal(t, expectedLabels, labelMap)
}

func TestUpdateNodeRequest_NodeKeyCriteria(t *testing.T) {
	t.Parallel()

	nodeCfg := fnode.NodeCfg{
		MinimalNodeCfg: fnode.MinimalNodeCfg{
			Name:   "test-node",
			CSAKey: "test-csa-key",
		},
		Tags: map[string]string{
			"region": "us-east-1",
		},
	}

	tests := []struct {
		name             string
		finderCfg        NodeFinderCfg
		expectedCriteria string
	}{
		{
			name: "name key type",
			finderCfg: NodeFinderCfg{
				KeyType: NodeKey_Name,
			},
			expectedCriteria: "key-type=name, value=test-node",
		},
		{
			name: "csa key type",
			finderCfg: NodeFinderCfg{
				KeyType: NodeKey_CSAKey,
			},
			expectedCriteria: "key-type=csa_key, value=test-csa-key",
		},
		{
			name: "label key type",
			finderCfg: NodeFinderCfg{
				KeyType:   NodeKey_Label,
				LabelName: fpointer.To("region"),
			},
			expectedCriteria: "key-type=label, value=region=us-east-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req, err := NewUpdateNodeRequest(nodeCfg, tt.finderCfg)
			require.NoError(t, err)

			criteria := req.NodeKeyCriteria()
			assert.Equal(t, tt.expectedCriteria, criteria)
		})
	}
}

func TestUpdateNodes_Success(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockClient := jdmocks.NewMockJDClient(t)

	// Create test nodes
	existingNodes := []*nodev1.Node{
		{
			Id:        "node-1",
			Name:      "old-name-1",
			PublicKey: "csa-key-1",
			Labels: plabels(map[string]string{
				"p2p_id": "p2p-1",
				"region": "us-east-1",
			}),
		},
		{
			Id:        "node-2",
			Name:      "old-name-2",
			PublicKey: "csa-key-2",
			Labels: plabels(map[string]string{
				"p2p_id": "p2p-2",
				"region": "us-west-1",
			}),
		},
	}

	listResponse := &nodev1.ListNodesResponse{
		Nodes: existingNodes,
	}

	// Create update requests
	req1, err := NewUpdateNodeRequest(fnode.NodeCfg{
		MinimalNodeCfg: fnode.MinimalNodeCfg{
			Name:   "new-name-1",
			CSAKey: "csa-key-1",
			NOP:    "nop-1",
		},
		P2PID:     "p2p-1",
		AdminAddr: "0x1111111111111111111111111111111111111111",
	}, NodeFinderCfg{KeyType: NodeKey_CSAKey})
	require.NoError(t, err)

	req2, err := NewUpdateNodeRequest(fnode.NodeCfg{
		MinimalNodeCfg: fnode.MinimalNodeCfg{
			Name:   "new-name-2",
			CSAKey: "csa-key-2",
			NOP:    "nop-2",
		},
		P2PID:     "p2p-2",
		AdminAddr: "0x2222222222222222222222222222222222222222",
	}, NodeFinderCfg{KeyType: NodeKey_CSAKey})
	require.NoError(t, err)

	updateReq := UpdateNodesRequest{
		Requests: []*UpdateNodeRequest{req1, req2},
	}

	// Mock ListNodes call
	mockClient.MockNodeServiceClient.On("ListNodes", ctx, &nodev1.ListNodesRequest{}).Return(listResponse, nil)

	// Mock UpdateNode calls
	mockClient.MockNodeServiceClient.On("UpdateNode", ctx, mock.MatchedBy(func(req *nodev1.UpdateNodeRequest) bool {
		return req.Id == "node-1" && req.Name == "new-name-1" && req.PublicKey == "csa-key-1"
	})).Return(&nodev1.UpdateNodeResponse{}, nil)

	mockClient.MockNodeServiceClient.On("UpdateNode", ctx, mock.MatchedBy(func(req *nodev1.UpdateNodeRequest) bool {
		return req.Id == "node-2" && req.Name == "new-name-2" && req.PublicKey == "csa-key-2"
	})).Return(&nodev1.UpdateNodeResponse{}, nil)

	err = UpdateNodes(ctx, mockClient, updateReq)
	require.NoError(t, err)

	mockClient.MockNodeServiceClient.AssertExpectations(t)
}

func TestUpdateNodes_EmptyRequest(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockClient := jdmocks.NewMockJDClient(t)

	updateReq := UpdateNodesRequest{
		Requests: []*UpdateNodeRequest{},
	}

	err := UpdateNodes(ctx, mockClient, updateReq)
	require.NoError(t, err)

	// No calls should be made
	mockClient.MockNodeServiceClient.AssertExpectations(t)
}

func TestUpdateNodes_ListNodesError(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockClient := jdmocks.NewMockJDClient(t)

	req, err := NewUpdateNodeRequest(fnode.NodeCfg{
		MinimalNodeCfg: fnode.MinimalNodeCfg{
			Name:   "test-node",
			CSAKey: "test-csa-key",
			NOP:    "test-nop",
		},
		P2PID:     "test-p2p",
		AdminAddr: "0x1111111111111111111111111111111111111111",
	}, NodeFinderCfg{KeyType: NodeKey_Name})
	require.NoError(t, err)

	updateReq := UpdateNodesRequest{
		Requests: []*UpdateNodeRequest{req},
	}

	// Mock ListNodes error
	expectedError := errors.New("failed to list nodes")
	mockClient.MockNodeServiceClient.On("ListNodes", ctx, &nodev1.ListNodesRequest{}).Return(nil, expectedError)

	err = UpdateNodes(ctx, mockClient, updateReq)
	require.Error(t, err)
	assert.Equal(t, expectedError, err)

	mockClient.MockNodeServiceClient.AssertExpectations(t)
}

func TestUpdateNodes_NodeNotFound(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockClient := jdmocks.NewMockJDClient(t)

	listResponse := &nodev1.ListNodesResponse{
		Nodes: []*nodev1.Node{
			{
				Id:        "node-1",
				Name:      "existing-node",
				PublicKey: "existing-csa-key",
			},
		},
	}

	req, err := NewUpdateNodeRequest(fnode.NodeCfg{
		MinimalNodeCfg: fnode.MinimalNodeCfg{
			Name:   "nonexistent-node",
			CSAKey: "nonexistent-csa-key",
			NOP:    "test-nop",
		},
		P2PID:     "test-p2p",
		AdminAddr: "0x1111111111111111111111111111111111111111",
	}, NodeFinderCfg{KeyType: NodeKey_Name})
	require.NoError(t, err)

	updateReq := UpdateNodesRequest{
		Requests: []*UpdateNodeRequest{req},
	}

	// Mock ListNodes call
	mockClient.MockNodeServiceClient.On("ListNodes", ctx, &nodev1.ListNodesRequest{}).Return(listResponse, nil)

	err = UpdateNodes(ctx, mockClient, updateReq)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get node")
	assert.Contains(t, err.Error(), "nonexistent-node")

	mockClient.MockNodeServiceClient.AssertExpectations(t)
}

func TestUpdateNodes_UpdateNodeError(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockClient := jdmocks.NewMockJDClient(t)

	existingNodes := []*nodev1.Node{
		{
			Id:        "node-1",
			Name:      "test-node",
			PublicKey: "test-csa-key",
		},
	}

	listResponse := &nodev1.ListNodesResponse{
		Nodes: existingNodes,
	}

	req, err := NewUpdateNodeRequest(fnode.NodeCfg{
		MinimalNodeCfg: fnode.MinimalNodeCfg{
			Name:   "test-node",
			CSAKey: "test-csa-key",
			NOP:    "test-nop",
		},
		P2PID:     "test-p2p",
		AdminAddr: "0x1111111111111111111111111111111111111111",
	}, NodeFinderCfg{KeyType: NodeKey_Name})
	require.NoError(t, err)

	updateReq := UpdateNodesRequest{
		Requests: []*UpdateNodeRequest{req},
	}

	// Mock ListNodes call
	mockClient.MockNodeServiceClient.On("ListNodes", ctx, &nodev1.ListNodesRequest{}).Return(listResponse, nil)

	// Mock UpdateNode error
	updateError := errors.New("failed to update node")
	mockClient.MockNodeServiceClient.On("UpdateNode", ctx, mock.AnythingOfType("*fnode.UpdateNodeRequest")).Return(nil, updateError)

	err = UpdateNodes(ctx, mockClient, updateReq)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update node")
	assert.Contains(t, err.Error(), "node-1")

	mockClient.MockNodeServiceClient.AssertExpectations(t)
}

func TestNewNodeKeyCfg(t *testing.T) {
	t.Parallel()

	nodeCfg := fnode.NodeCfg{
		MinimalNodeCfg: fnode.MinimalNodeCfg{
			Name:   "test-node",
			CSAKey: "test-csa-key",
		},
		Tags: map[string]string{
			"region": "us-east-1",
			"env":    "test",
		},
	}

	tests := []struct {
		name        string
		finderCfg   NodeFinderCfg
		want        nodeKeyCfg
		wantErr     bool
		errContains string
	}{
		{
			name: "name key type",
			finderCfg: NodeFinderCfg{
				KeyType: NodeKey_Name,
			},
			want: nodeKeyCfg{
				keyType: NodeKey_Name,
				value:   "test-node",
			},
			wantErr: false,
		},
		{
			name: "csa key type",
			finderCfg: NodeFinderCfg{
				KeyType: NodeKey_CSAKey,
			},
			want: nodeKeyCfg{
				keyType: NodeKey_CSAKey,
				value:   "test-csa-key",
			},
			wantErr: false,
		},
		{
			name: "label key type",
			finderCfg: NodeFinderCfg{
				KeyType:   NodeKey_Label,
				LabelName: fpointer.To("region"),
			},
			want: nodeKeyCfg{
				keyType:  NodeKey_Label,
				value:    "us-east-1",
				labelKey: fpointer.To("region"),
			},
			wantErr: false,
		},
		{
			name: "label key type with nonexistent label",
			finderCfg: NodeFinderCfg{
				KeyType:   NodeKey_Label,
				LabelName: fpointer.To("nonexistent"),
			},
			want: nodeKeyCfg{
				keyType:  NodeKey_Label,
				value:    "", // empty string for nonexistent label
				labelKey: fpointer.To("nonexistent"),
			},
			wantErr: false,
		},
		{
			name: "id key type - not supported",
			finderCfg: NodeFinderCfg{
				KeyType: NodeKey_ID,
			},
			wantErr:     true,
			errContains: "id key type is not supported",
		},
		{
			name: "unknown key type",
			finderCfg: NodeFinderCfg{
				KeyType: "unknown",
			},
			wantErr:     true,
			errContains: "unknown key type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := newNodeKeyCfg(nodeCfg, tt.finderCfg)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestNodeKeyCfg_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  nodeKeyCfg
		want string
	}{
		{
			name: "name key type",
			cfg: nodeKeyCfg{
				keyType: NodeKey_Name,
				value:   "test-node",
			},
			want: "key-type=name, value=test-node",
		},
		{
			name: "csa key type",
			cfg: nodeKeyCfg{
				keyType: NodeKey_CSAKey,
				value:   "test-csa-key",
			},
			want: "key-type=csa_key, value=test-csa-key",
		},
		{
			name: "label key type",
			cfg: nodeKeyCfg{
				keyType:  NodeKey_Label,
				value:    "us-east-1",
				labelKey: fpointer.To("region"),
			},
			want: "key-type=label, value=region=us-east-1",
		},
		{
			name: "label key type with empty value",
			cfg: nodeKeyCfg{
				keyType:  NodeKey_Label,
				value:    "",
				labelKey: fpointer.To("nonexistent"),
			},
			want: "key-type=label, value=nonexistent=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.cfg.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

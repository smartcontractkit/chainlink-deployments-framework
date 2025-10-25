package memory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	csav1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/csa"
	jobv1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/job"
	nodev1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/node"
	"github.com/smartcontractkit/chainlink-protos/job-distributor/v1/shared/ptypes"

	"github.com/smartcontractkit/chainlink-deployments-framework/internal/pointer"
)

func TestMemoryJobDistributor_ProposeJob(t *testing.T) {
	t.Parallel()

	t.Run("successfully propose a job", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		req := &jobv1.ProposeJobRequest{
			NodeId: "test-node-1",
			Spec:   "test job spec",
		}

		resp, err := client.ProposeJob(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Proposal)

		assert.NotEmpty(t, resp.Proposal.Id)
		assert.NotEmpty(t, resp.Proposal.JobId)
		assert.Equal(t, "test job spec", resp.Proposal.Spec)
		assert.Equal(t, jobv1.ProposalStatus_PROPOSAL_STATUS_APPROVED, resp.Proposal.Status)
	})

	t.Run("nil request returns error", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		resp, err := client.ProposeJob(ctx, nil)
		require.ErrorContains(t, err, "request cannot be nil")
		assert.Nil(t, resp)
	})
}

func TestMemoryJobDistributor_GetJob(t *testing.T) {
	t.Parallel()

	t.Run("get existing job by id", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		// First create a job via proposal
		proposeResp, err := client.ProposeJob(ctx, &jobv1.ProposeJobRequest{
			NodeId: "test-node-1",
			Spec:   "test job spec",
		})
		require.NoError(t, err)
		jobID := proposeResp.Proposal.JobId

		// Now get the job
		getResp, err := client.GetJob(ctx, &jobv1.GetJobRequest{
			IdOneof: &jobv1.GetJobRequest_Id{Id: jobID},
		})
		require.NoError(t, err)
		require.NotNil(t, getResp)
		require.NotNil(t, getResp.Job)

		assert.Equal(t, jobID, getResp.Job.Id)
	})

	t.Run("get non-existent job returns error", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		resp, err := client.GetJob(ctx, &jobv1.GetJobRequest{
			IdOneof: &jobv1.GetJobRequest_Id{Id: "non-existent"},
		})
		require.ErrorContains(t, err, "not found")
		assert.Nil(t, resp)
	})

	t.Run("nil request returns error", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		resp, err := client.GetJob(ctx, nil)
		require.ErrorContains(t, err, "request cannot be nil")
		assert.Nil(t, resp)
	})

	t.Run("missing id returns error", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		resp, err := client.GetJob(ctx, &jobv1.GetJobRequest{})
		require.ErrorContains(t, err, "job id must be provided")
		assert.Nil(t, resp)
	})
}

func TestMemoryJobDistributor_ListJobs(t *testing.T) {
	t.Parallel()

	t.Run("list jobs", func(t *testing.T) {
		t.Parallel()

		client := NewMemoryJobDistributor()
		ctx := t.Context()

		// Create multiple jobs
		_, err := client.ProposeJob(ctx, &jobv1.ProposeJobRequest{
			NodeId: "test-node-1",
			Spec:   "job spec 1",
			Labels: []*ptypes.Label{
				{Key: "environment", Value: pointer.To("prod")},
			},
		})
		require.NoError(t, err)

		_, err = client.ProposeJob(ctx, &jobv1.ProposeJobRequest{
			NodeId: "test-node-2",
			Spec:   "job spec 2",
		})
		require.NoError(t, err)

		t.Run("list all jobs", func(t *testing.T) {
			t.Parallel()

			listResp, err := client.ListJobs(ctx, &jobv1.ListJobsRequest{})
			require.NoError(t, err)
			require.NotNil(t, listResp)
			assert.Len(t, listResp.Jobs, 2)
		})

		t.Run("filter jobs by label", func(t *testing.T) {
			t.Parallel()
			listResp, err := client.ListJobs(ctx, &jobv1.ListJobsRequest{
				Filter: &jobv1.ListJobsRequest_Filter{
					Selectors: []*ptypes.Selector{
						{Key: "environment", Op: ptypes.SelectorOp_EQ, Value: pointer.To("prod")},
					},
				},
			})
			require.NoError(t, err)
			require.NotNil(t, listResp)
			assert.Len(t, listResp.Jobs, 1)
		})
	})

	t.Run("list jobs excludes soft-deleted jobs by default", func(t *testing.T) {
		t.Parallel()

		client := NewMemoryJobDistributor()
		ctx := t.Context()

		// Create a job
		proposeResp, err := client.ProposeJob(ctx, &jobv1.ProposeJobRequest{
			NodeId: "test-node-1",
			Spec:   "job spec 1",
		})
		require.NoError(t, err)
		jobID := proposeResp.Proposal.JobId

		// Soft delete the job
		_, err = client.DeleteJob(ctx, &jobv1.DeleteJobRequest{
			IdOneof: &jobv1.DeleteJobRequest_Id{Id: jobID},
		})
		require.NoError(t, err)

		// List jobs should exclude the soft-deleted job
		listResp, err := client.ListJobs(ctx, &jobv1.ListJobsRequest{})
		require.NoError(t, err)
		require.NotNil(t, listResp)
		assert.Empty(t, listResp.Jobs)

		// But when IncludeDeleted is true, it should include the soft-deleted job
		listRespWithDeleted, err := client.ListJobs(ctx, &jobv1.ListJobsRequest{
			Filter: &jobv1.ListJobsRequest_Filter{
				IncludeDeleted: true,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, listRespWithDeleted)
		assert.Len(t, listRespWithDeleted.Jobs, 1)
		assert.Equal(t, jobID, listRespWithDeleted.Jobs[0].Id)
	})

	t.Run("list jobs on empty store returns empty list", func(t *testing.T) {
		t.Parallel()

		emptyClient := NewMemoryJobDistributor()
		ctx := t.Context()
		listResp, err := emptyClient.ListJobs(ctx, &jobv1.ListJobsRequest{})
		require.NoError(t, err)
		require.NotNil(t, listResp)
		assert.Empty(t, listResp.Jobs)
	})
}

func TestMemoryJobDistributor_GetProposal(t *testing.T) {
	t.Parallel()

	t.Run("get existing proposal", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		// Create a proposal
		proposeResp, err := client.ProposeJob(ctx, &jobv1.ProposeJobRequest{
			NodeId: "test-node-1",
			Spec:   "test job spec",
		})
		require.NoError(t, err)
		proposalID := proposeResp.Proposal.Id

		// Get the proposal
		getResp, err := client.GetProposal(ctx, &jobv1.GetProposalRequest{Id: proposalID})
		require.NoError(t, err)
		require.NotNil(t, getResp)
		require.NotNil(t, getResp.Proposal)

		assert.Equal(t, proposalID, getResp.Proposal.Id)
		assert.Equal(t, "test job spec", getResp.Proposal.Spec)
	})

	t.Run("get non-existent proposal returns error", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		resp, err := client.GetProposal(ctx, &jobv1.GetProposalRequest{Id: "non-existent"})
		require.ErrorContains(t, err, "not found")
		assert.Nil(t, resp)
	})
}

func TestMemoryJobDistributor_ListProposals(t *testing.T) {
	t.Parallel()

	t.Run("list proposals returns all proposals", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		// Create multiple proposals
		_, err := client.ProposeJob(ctx, &jobv1.ProposeJobRequest{
			NodeId: "test-node-1",
			Spec:   "job spec 1",
		})
		require.NoError(t, err)

		_, err = client.ProposeJob(ctx, &jobv1.ProposeJobRequest{
			NodeId: "test-node-2",
			Spec:   "job spec 2",
		})
		require.NoError(t, err)

		// List all proposals
		listResp, err := client.ListProposals(ctx, &jobv1.ListProposalsRequest{})
		require.NoError(t, err)
		require.NotNil(t, listResp)

		assert.Len(t, listResp.Proposals, 2)
	})
}

func TestMemoryJobDistributor_BatchProposeJob(t *testing.T) {
	t.Parallel()

	t.Run("batch propose multiple jobs", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		req := &jobv1.BatchProposeJobRequest{
			NodeIds: []string{"node-1", "node-2", "node-3"},
			Spec:    "shared spec",
		}

		resp, err := client.BatchProposeJob(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.Len(t, resp.SuccessResponses, 3)
		for _, nodeID := range req.NodeIds {
			propResp, exists := resp.SuccessResponses[nodeID]
			assert.True(t, exists, "missing response for node %s", nodeID)
			assert.NotNil(t, propResp.Proposal)
			assert.NotEmpty(t, propResp.Proposal.Id)
			assert.NotEmpty(t, propResp.Proposal.JobId)
			assert.Equal(t, "shared spec", propResp.Proposal.Spec)
		}

		// Verify jobs were created
		listResp, err := client.ListJobs(ctx, &jobv1.ListJobsRequest{})
		require.NoError(t, err)
		assert.Len(t, listResp.Jobs, 3)
	})

	t.Run("nil request returns error", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		resp, err := client.BatchProposeJob(ctx, nil)
		require.ErrorContains(t, err, "request cannot be nil")
		assert.Nil(t, resp)
	})
}

func TestMemoryJobDistributor_RevokeJob(t *testing.T) {
	t.Parallel()

	t.Run("revoke existing job", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		// Create a job
		proposeResp, err := client.ProposeJob(ctx, &jobv1.ProposeJobRequest{
			NodeId: "test-node-1",
			Spec:   "test job spec",
		})
		require.NoError(t, err)
		jobID := proposeResp.Proposal.JobId
		proposalID := proposeResp.Proposal.Id

		// Revoke the job
		revokeResp, err := client.RevokeJob(ctx, &jobv1.RevokeJobRequest{
			IdOneof: &jobv1.RevokeJobRequest_Id{Id: jobID},
		})
		require.NoError(t, err)
		require.NotNil(t, revokeResp)

		// Verify proposal status changed
		getResp, err := client.GetProposal(ctx, &jobv1.GetProposalRequest{Id: proposalID})
		require.NoError(t, err)
		assert.Equal(t, jobv1.ProposalStatus_PROPOSAL_STATUS_REVOKED, getResp.Proposal.Status)
	})

	t.Run("nil request returns error", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		resp, err := client.RevokeJob(ctx, nil)
		require.ErrorContains(t, err, "request cannot be nil")
		assert.Nil(t, resp)
	})
}

func TestMemoryJobDistributor_DeleteJob(t *testing.T) {
	t.Parallel()

	t.Run("delete existing job", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		// Create a job
		proposeResp, err := client.ProposeJob(ctx, &jobv1.ProposeJobRequest{
			NodeId: "test-node-1",
			Spec:   "test job spec",
		})
		require.NoError(t, err)
		jobID := proposeResp.Proposal.JobId

		// Delete the job
		deleteResp, err := client.DeleteJob(ctx, &jobv1.DeleteJobRequest{
			IdOneof: &jobv1.DeleteJobRequest_Id{Id: jobID},
		})
		require.NoError(t, err)
		require.NotNil(t, deleteResp)

		// Verify job is deleted
		getResp, err := client.GetJob(ctx, &jobv1.GetJobRequest{
			IdOneof: &jobv1.GetJobRequest_Id{Id: jobID},
		})
		require.NoError(t, err)

		assert.NotNil(t, getResp.Job.DeletedAt)
	})

	t.Run("delete non-existent job does errors", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		_, err := client.DeleteJob(ctx, &jobv1.DeleteJobRequest{
			IdOneof: &jobv1.DeleteJobRequest_Id{Id: "non-existent"},
		})
		require.Error(t, err)
	})
}

func TestMemoryJobDistributor_UpdateJob(t *testing.T) {
	t.Parallel()

	t.Run("update existing job labels", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		// Create a job
		value := "testvalue"
		proposeResp, err := client.ProposeJob(ctx, &jobv1.ProposeJobRequest{
			NodeId: "test-node-1",
			Spec:   "original spec",
			Labels: []*ptypes.Label{
				{Key: "env", Value: &value},
			},
		})
		require.NoError(t, err)
		jobID := proposeResp.Proposal.JobId

		// Update the job
		updatedValue := "production"
		updateResp, err := client.UpdateJob(ctx, &jobv1.UpdateJobRequest{
			IdOneof: &jobv1.UpdateJobRequest_Id{Id: jobID},
			Labels: []*ptypes.Label{
				{Key: "env", Value: &updatedValue},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, updateResp)
		require.Len(t, updateResp.Job.Labels, 1)
		assert.Equal(t, "env", updateResp.Job.Labels[0].Key)
		assert.Equal(t, "production", *updateResp.Job.Labels[0].Value)

		// Verify job was updated
		getResp, err := client.GetJob(ctx, &jobv1.GetJobRequest{
			IdOneof: &jobv1.GetJobRequest_Id{Id: jobID},
		})
		require.NoError(t, err)
		require.Len(t, getResp.Job.Labels, 1)
		assert.Equal(t, "production", *getResp.Job.Labels[0].Value)
	})

	t.Run("update non-existent job returns error", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		resp, err := client.UpdateJob(ctx, &jobv1.UpdateJobRequest{
			IdOneof: &jobv1.UpdateJobRequest_Id{Id: "non-existent"},
		})
		require.ErrorContains(t, err, "not found")
		assert.Nil(t, resp)
	})
}

func TestMemoryJobDistributor_RegisterNode(t *testing.T) {
	t.Parallel()

	t.Run("register node", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		value := "test"
		req := &nodev1.RegisterNodeRequest{
			Name:      "Test Node",
			PublicKey: "test-public-key",
			Labels: []*ptypes.Label{
				{Key: "env", Value: &value},
			},
		}

		resp, err := client.RegisterNode(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Node)

		assert.NotEmpty(t, resp.Node.Id)
		assert.Equal(t, "Test Node", resp.Node.Name)
		assert.Equal(t, "test-public-key", resp.Node.PublicKey)
		assert.True(t, resp.Node.IsEnabled)
		assert.False(t, resp.Node.IsConnected)
		require.Len(t, resp.Node.Labels, 1)
		assert.Equal(t, "env", resp.Node.Labels[0].Key)
		assert.Equal(t, "test", *resp.Node.Labels[0].Value)
	})

	t.Run("nil request returns error", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		resp, err := client.RegisterNode(ctx, nil)
		require.ErrorContains(t, err, "request cannot be nil")
		assert.Nil(t, resp)
	})
}

func TestMemoryJobDistributor_GetNode(t *testing.T) {
	t.Parallel()

	t.Run("get existing node", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		// Register a node
		registerResp, err := client.RegisterNode(ctx, &nodev1.RegisterNodeRequest{
			Name:      "Test Node",
			PublicKey: "test-public-key",
		})
		require.NoError(t, err)
		nodeID := registerResp.Node.Id

		// Get the node
		getResp, err := client.GetNode(ctx, &nodev1.GetNodeRequest{Id: nodeID})
		require.NoError(t, err)
		require.NotNil(t, getResp)
		require.NotNil(t, getResp.Node)

		assert.Equal(t, nodeID, getResp.Node.Id)
		assert.Equal(t, "Test Node", getResp.Node.Name)
	})

	t.Run("get non-existent node returns error", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		resp, err := client.GetNode(ctx, &nodev1.GetNodeRequest{Id: "non-existent"})
		require.ErrorContains(t, err, "not found")
		assert.Nil(t, resp)
	})
}

func TestMemoryJobDistributor_ListNodes(t *testing.T) {
	t.Parallel()

	client := NewMemoryJobDistributor()
	ctx := t.Context()
	// Register multiple nodes
	_, err := client.RegisterNode(ctx, &nodev1.RegisterNodeRequest{
		Name:      "Node 1",
		PublicKey: "key-1",
		Labels: []*ptypes.Label{
			{Key: "environment", Value: pointer.To("prod")},
		},
	})
	require.NoError(t, err)

	_, err = client.RegisterNode(ctx, &nodev1.RegisterNodeRequest{
		Name:      "Node 2",
		PublicKey: "key-2",
	})
	require.NoError(t, err)

	t.Run("list nodes returns all nodes", func(t *testing.T) {
		t.Parallel()

		// List all nodes
		listResp, err := client.ListNodes(ctx, &nodev1.ListNodesRequest{})
		require.NoError(t, err)
		require.NotNil(t, listResp)

		assert.Len(t, listResp.Nodes, 2)
	})

	t.Run("filter nodes by label", func(t *testing.T) {
		t.Parallel()

		listResp, err := client.ListNodes(ctx, &nodev1.ListNodesRequest{
			Filter: &nodev1.ListNodesRequest_Filter{
				Selectors: []*ptypes.Selector{
					{
						Key:   "environment",
						Op:    ptypes.SelectorOp_EQ,
						Value: pointer.To("prod"),
					},
				},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, listResp)
		assert.Len(t, listResp.Nodes, 1)
		assert.Equal(t, "Node 1", listResp.Nodes[0].Name)
	})
}

func TestMemoryJobDistributor_UpdateNode(t *testing.T) {
	t.Parallel()

	t.Run("update existing node", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		// Register a node
		registerResp, err := client.RegisterNode(ctx, &nodev1.RegisterNodeRequest{
			Name:      "Original Name",
			PublicKey: "original-key",
		})
		require.NoError(t, err)
		nodeID := registerResp.Node.Id

		// Update the node
		updatedValue := "true"
		updateResp, err := client.UpdateNode(ctx, &nodev1.UpdateNodeRequest{
			Id:        nodeID,
			Name:      "Updated Name",
			PublicKey: "updated-key",
			Labels: []*ptypes.Label{
				{Key: "updated", Value: &updatedValue},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, updateResp)

		assert.Equal(t, "Updated Name", updateResp.Node.Name)
		assert.Equal(t, "updated-key", updateResp.Node.PublicKey)
		require.Len(t, updateResp.Node.Labels, 1)
		assert.Equal(t, "updated", updateResp.Node.Labels[0].Key)
		assert.Equal(t, "true", *updateResp.Node.Labels[0].Value)
	})

	t.Run("update non-existent node returns error", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		resp, err := client.UpdateNode(ctx, &nodev1.UpdateNodeRequest{
			Id:   "non-existent",
			Name: "Updated Name",
		})
		require.ErrorContains(t, err, "not found")
		assert.Nil(t, resp)
	})
}

func TestMemoryJobDistributor_EnableDisableNode(t *testing.T) {
	t.Parallel()

	t.Run("disable and enable node", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		// Register a node
		registerResp, err := client.RegisterNode(ctx, &nodev1.RegisterNodeRequest{
			Name:      "Test Node",
			PublicKey: "test-key",
		})
		require.NoError(t, err)
		assert.True(t, registerResp.Node.IsEnabled)
		nodeID := registerResp.Node.Id

		// Disable the node
		disableResp, err := client.DisableNode(ctx, &nodev1.DisableNodeRequest{Id: nodeID})
		require.NoError(t, err)
		require.NotNil(t, disableResp)
		assert.False(t, disableResp.Node.IsEnabled)

		// Verify node is disabled
		getResp, err := client.GetNode(ctx, &nodev1.GetNodeRequest{Id: nodeID})
		require.NoError(t, err)
		assert.False(t, getResp.Node.IsEnabled)

		// Enable the node
		enableResp, err := client.EnableNode(ctx, &nodev1.EnableNodeRequest{Id: nodeID})
		require.NoError(t, err)
		require.NotNil(t, enableResp)
		assert.True(t, enableResp.Node.IsEnabled)

		// Verify node is enabled
		getResp, err = client.GetNode(ctx, &nodev1.GetNodeRequest{Id: nodeID})
		require.NoError(t, err)
		assert.True(t, getResp.Node.IsEnabled)
	})
}

func TestMemoryJobDistributor_ListNodeChainConfigs(t *testing.T) {
	t.Parallel()

	t.Run("list chain configs for specific node", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		// Register a node
		registerResp, err := client.RegisterNode(ctx, &nodev1.RegisterNodeRequest{
			Name:      "Test Node",
			PublicKey: "test-key",
		})
		require.NoError(t, err)
		nodeID := registerResp.Node.Id

		// Add chain configs
		config1 := &nodev1.ChainConfig{
			Chain: &nodev1.Chain{Id: "chain-1", Type: nodev1.ChainType_CHAIN_TYPE_EVM},
		}
		config2 := &nodev1.ChainConfig{
			Chain: &nodev1.Chain{Id: "chain-2", Type: nodev1.ChainType_CHAIN_TYPE_SOLANA},
		}
		err = client.AddChainConfig(nodeID, config1)
		require.NoError(t, err)
		err = client.AddChainConfig(nodeID, config2)
		require.NoError(t, err)

		// List chain configs for the node
		listResp, err := client.ListNodeChainConfigs(ctx, &nodev1.ListNodeChainConfigsRequest{
			Filter: &nodev1.ListNodeChainConfigsRequest_Filter{
				NodeIds: []string{nodeID},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, listResp)

		assert.Len(t, listResp.ChainConfigs, 2)
	})

	t.Run("list all chain configs", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()

		// Register nodes and add configs
		registerResp1, err := client.RegisterNode(ctx, &nodev1.RegisterNodeRequest{
			Name:      "Node 1",
			PublicKey: "key-1",
		})
		require.NoError(t, err)
		nodeID1 := registerResp1.Node.Id

		registerResp2, err := client.RegisterNode(ctx, &nodev1.RegisterNodeRequest{
			Name:      "Node 2",
			PublicKey: "key-2",
		})
		require.NoError(t, err)
		nodeID2 := registerResp2.Node.Id

		err = client.AddChainConfig(nodeID1, &nodev1.ChainConfig{
			Chain: &nodev1.Chain{Id: "chain-1", Type: nodev1.ChainType_CHAIN_TYPE_EVM},
		})
		require.NoError(t, err)

		err = client.AddChainConfig(nodeID2, &nodev1.ChainConfig{
			Chain: &nodev1.Chain{Id: "chain-2", Type: nodev1.ChainType_CHAIN_TYPE_SOLANA},
		})
		require.NoError(t, err)

		// List all chain configs
		listResp, err := client.ListNodeChainConfigs(ctx, &nodev1.ListNodeChainConfigsRequest{})
		require.NoError(t, err)
		require.NotNil(t, listResp)

		assert.Len(t, listResp.ChainConfigs, 2)
	})

	t.Run("add chain config to non-existent node returns error", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		err := client.AddChainConfig("non-existent", &nodev1.ChainConfig{
			Chain: &nodev1.Chain{Id: "chain-1", Type: nodev1.ChainType_CHAIN_TYPE_EVM},
		})
		require.ErrorContains(t, err, "not found")
	})
}

func TestMemoryJobDistributor_GetKeypair(t *testing.T) {
	t.Parallel()

	t.Run("get keypair returns first one", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		// Add keypairs
		client.AddKeypair(&csav1.Keypair{PublicKey: "test-public-key-1"})
		client.AddKeypair(&csav1.Keypair{PublicKey: "test-public-key-2"})

		// Get a keypair (should return first one found)
		getResp, err := client.GetKeypair(ctx, &csav1.GetKeypairRequest{})
		require.NoError(t, err)
		require.NotNil(t, getResp)
		require.NotNil(t, getResp.Keypair)

		// Should be one of the added keypairs
		assert.Contains(t, []string{"test-public-key-1", "test-public-key-2"}, getResp.Keypair.PublicKey)
	})

	t.Run("get keypair with no keypairs returns error", func(t *testing.T) {
		t.Parallel()
		emptyClient := NewMemoryJobDistributor()
		ctx := t.Context()
		resp, err := emptyClient.GetKeypair(ctx, &csav1.GetKeypairRequest{})
		require.ErrorContains(t, err, "no keypairs found")
		assert.Nil(t, resp)
	})
}

func TestMemoryJobDistributor_ListKeypairs(t *testing.T) {
	t.Parallel()

	t.Run("list keypairs returns all keypairs", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()
		ctx := t.Context()
		// Add multiple keypairs
		client.AddKeypair(&csav1.Keypair{PublicKey: "key-1"})
		client.AddKeypair(&csav1.Keypair{PublicKey: "key-2"})
		client.AddKeypair(&csav1.Keypair{PublicKey: "key-3"})

		// List all keypairs
		listResp, err := client.ListKeypairs(ctx, &csav1.ListKeypairsRequest{})
		require.NoError(t, err)
		require.NotNil(t, listResp)

		assert.Len(t, listResp.Keypairs, 3)
	})

	t.Run("list keypairs on empty store returns empty list", func(t *testing.T) {
		t.Parallel()
		emptyClient := NewMemoryJobDistributor()
		ctx := t.Context()
		listResp, err := emptyClient.ListKeypairs(ctx, &csav1.ListKeypairsRequest{})
		require.NoError(t, err)
		require.NotNil(t, listResp)
		assert.Empty(t, listResp.Keypairs)
	})
}

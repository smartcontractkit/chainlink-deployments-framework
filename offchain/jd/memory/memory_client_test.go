package memory

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	csav1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/csa"
	jobv1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/job"
	nodev1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/node"
	"github.com/smartcontractkit/chainlink-protos/job-distributor/v1/shared/ptypes"

	"github.com/smartcontractkit/chainlink-deployments-framework/internal/pointer"
)

var (
	// A unique spec external job id for testing
	spec1UUID = uuid.New().String()
	// A minimally viable spec1 for testing with an external job id
	spec1 = fmt.Sprintf("externalJobID = '%s'", spec1UUID)

	// An additional spec with a different external job id for testing
	spec2UUID = uuid.New().String()
	// A minimally viable spec2 for testing with an external job id
	spec2 = fmt.Sprintf("externalJobID = '%s'", spec2UUID)
)

func TestMemoryJobDistributor_ProposeJob(t *testing.T) {
	t.Parallel()

	var (
		labels = []*ptypes.Label{
			{Key: "environment", Value: pointer.To("prod")},
		}
	)

	t.Run("successfully propose a job", func(t *testing.T) {
		t.Parallel()

		client := NewMemoryJobDistributor()
		req := &jobv1.ProposeJobRequest{
			NodeId: "test-node-1",
			Spec:   spec1,
			Labels: labels,
		}

		resp, err := client.ProposeJob(t.Context(), req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Proposal)
		assertProposalEquals(t, &jobv1.Proposal{
			Revision:       1,
			Status:         jobv1.ProposalStatus_PROPOSAL_STATUS_PROPOSED,
			DeliveryStatus: jobv1.ProposalDeliveryStatus_PROPOSAL_DELIVERY_STATUS_DELIVERED,
			JobId:          resp.Proposal.JobId,
			Spec:           spec1,
		}, resp.Proposal)

		// Check the job
		getResp, err := client.GetJob(t.Context(), &jobv1.GetJobRequest{
			IdOneof: &jobv1.GetJobRequest_Id{Id: resp.Proposal.JobId},
		})
		require.NoError(t, err)
		require.NotNil(t, getResp)
		require.NotNil(t, getResp.Job)
		assertJobEquals(t, &jobv1.Job{
			Id:          resp.Proposal.JobId,
			Uuid:        spec1UUID,
			NodeId:      req.NodeId,
			ProposalIds: []string{resp.Proposal.Id},
			Labels:      labels,
		}, getResp.Job)
	})

	t.Run("propose job with same external job id returns same job id", func(t *testing.T) {
		t.Parallel()

		client := NewMemoryJobDistributor()
		ctx := t.Context()
		req := &jobv1.ProposeJobRequest{
			NodeId: "test-node-1",
			Spec:   spec1,
			Labels: labels,
		}

		resp1, err := client.ProposeJob(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp1)
		assertProposalEquals(t, &jobv1.Proposal{
			Revision:       1,
			Status:         jobv1.ProposalStatus_PROPOSAL_STATUS_PROPOSED,
			DeliveryStatus: jobv1.ProposalDeliveryStatus_PROPOSAL_DELIVERY_STATUS_DELIVERED,
			JobId:          resp1.Proposal.JobId,
			Spec:           spec1,
		}, resp1.Proposal)

		resp2, err := client.ProposeJob(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp2)
		assertProposalEquals(t, &jobv1.Proposal{
			Revision:       2,
			Status:         jobv1.ProposalStatus_PROPOSAL_STATUS_PROPOSED,
			DeliveryStatus: jobv1.ProposalDeliveryStatus_PROPOSAL_DELIVERY_STATUS_DELIVERED,
			JobId:          resp1.Proposal.JobId,
			Spec:           spec1,
		}, resp2.Proposal)

		// Check the job
		getResp, err := client.GetJob(t.Context(), &jobv1.GetJobRequest{
			IdOneof: &jobv1.GetJobRequest_Id{Id: resp1.Proposal.JobId},
		})
		require.NoError(t, err)
		require.NotNil(t, getResp)
		require.NotNil(t, getResp.Job)
		assertJobEquals(t, &jobv1.Job{
			Id:          resp1.Proposal.JobId,
			Uuid:        spec1UUID,
			NodeId:      req.NodeId,
			ProposalIds: []string{resp1.Proposal.Id, resp2.Proposal.Id},
			Labels:      labels,
		}, getResp.Job)
	})

	t.Run("spec is invalid", func(t *testing.T) {
		t.Parallel()

		client := NewMemoryJobDistributor()
		ctx := t.Context()
		_, err := client.ProposeJob(ctx, &jobv1.ProposeJobRequest{
			NodeId: "test-node-1",
			Spec:   "invalid spec",
		})

		require.ErrorContains(t, err, "invalid spec")
	})

	t.Run("nil request returns error", func(t *testing.T) {
		t.Parallel()

		client := NewMemoryJobDistributor()
		resp, err := client.ProposeJob(t.Context(), nil)
		require.ErrorContains(t, err, "request cannot be nil")
		assert.Nil(t, resp)
	})
}

func TestMemoryJobDistributor_GetJob(t *testing.T) {
	t.Parallel()

	t.Run("get job by id", func(t *testing.T) {
		t.Parallel()

		client := NewMemoryJobDistributor()
		ctx := t.Context()

		// Create the job
		proposeResp, err := client.ProposeJob(ctx, &jobv1.ProposeJobRequest{
			NodeId: "test-node-1",
			Spec:   spec1,
		})
		require.NoError(t, err)
		jobID := proposeResp.Proposal.JobId

		t.Run("get job by id", func(t *testing.T) {
			t.Parallel()

			getResp, err := client.GetJob(ctx, &jobv1.GetJobRequest{
				IdOneof: &jobv1.GetJobRequest_Id{Id: jobID},
			})
			require.NoError(t, err)
			require.NotNil(t, getResp)
		})

		t.Run("job not found", func(t *testing.T) {
			t.Parallel()

			_, err := client.GetJob(ctx, &jobv1.GetJobRequest{
				IdOneof: &jobv1.GetJobRequest_Id{Id: "non-existent"},
			})
			require.ErrorContains(t, err, "not found")
		})
	})

	t.Run("invalid request", func(t *testing.T) {
		t.Parallel()

		client := NewMemoryJobDistributor()

		t.Run("nil request", func(t *testing.T) {
			t.Parallel()

			_, err := client.GetJob(t.Context(), nil)
			require.ErrorContains(t, err, "request cannot be nil")
		})

		t.Run("invalid id oneof", func(t *testing.T) {
			t.Parallel()

			_, err := client.GetJob(t.Context(), &jobv1.GetJobRequest{})
			require.ErrorContains(t, err, "must provide id")
		})

		t.Run("uuid lookup is not supported", func(t *testing.T) {
			t.Parallel()

			_, err := client.GetJob(t.Context(), &jobv1.GetJobRequest{
				IdOneof: &jobv1.GetJobRequest_Uuid{Uuid: spec1UUID},
			})
			require.ErrorContains(t, err, "uuid lookup is deprecated and not supported by this implementation")
		})
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
			Spec:   spec1,
			Labels: []*ptypes.Label{
				{Key: "environment", Value: pointer.To("prod")},
			},
		})
		require.NoError(t, err)

		_, err = client.ProposeJob(ctx, &jobv1.ProposeJobRequest{
			NodeId: "test-node-2",
			Spec:   spec2,
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
			Spec:   spec1,
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
			Spec:   spec1,
		})
		require.NoError(t, err)
		proposalID := proposeResp.Proposal.Id

		// Get the proposal
		getResp, err := client.GetProposal(ctx, &jobv1.GetProposalRequest{Id: proposalID})
		require.NoError(t, err)
		require.NotNil(t, getResp)
		assertProposalEquals(t, &jobv1.Proposal{
			Id:             proposalID,
			Revision:       1,
			Status:         jobv1.ProposalStatus_PROPOSAL_STATUS_PROPOSED,
			DeliveryStatus: jobv1.ProposalDeliveryStatus_PROPOSAL_DELIVERY_STATUS_DELIVERED,
			JobId:          proposeResp.Proposal.JobId,
			Spec:           spec1,
		}, getResp.Proposal)
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
			Spec:   spec1,
		})
		require.NoError(t, err)

		_, err = client.ProposeJob(ctx, &jobv1.ProposeJobRequest{
			NodeId: "test-node-2",
			Spec:   spec1,
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
			Spec:    spec1,
		}

		resp, err := client.BatchProposeJob(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)

		require.Len(t, resp.SuccessResponses, 3)
		for _, nodeID := range req.NodeIds {
			propResp, exists := resp.SuccessResponses[nodeID]
			assert.True(t, exists, "missing response for node %s", nodeID)

			assertProposalEquals(t, &jobv1.Proposal{
				Id:             propResp.Proposal.Id,
				Revision:       1,
				Status:         jobv1.ProposalStatus_PROPOSAL_STATUS_PROPOSED,
				DeliveryStatus: jobv1.ProposalDeliveryStatus_PROPOSAL_DELIVERY_STATUS_DELIVERED,
				JobId:          propResp.Proposal.JobId,
				Spec:           spec1,
			}, propResp.Proposal)
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

	// Setup a job and proposal and return the client, proposal id, and job id
	setup := func(t *testing.T) (*MemoryJobDistributor, string, string) {
		t.Helper()

		client := NewMemoryJobDistributor()

		proposeResp, err := client.ProposeJob(t.Context(), &jobv1.ProposeJobRequest{
			NodeId: "test-node-1",
			Spec:   spec1,
		})
		require.NoError(t, err)

		jobID := proposeResp.Proposal.JobId
		propID := proposeResp.Proposal.Id

		return client, propID, jobID
	}

	t.Run("revoke existing job", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		client, propID, jobID := setup(t)

		// Revoke the job
		revokeResp, err := client.RevokeJob(ctx, &jobv1.RevokeJobRequest{
			IdOneof: &jobv1.RevokeJobRequest_Id{Id: jobID},
		})
		require.NoError(t, err)
		require.NotNil(t, revokeResp)

		// Verify proposal status changed
		getResp, err := client.GetProposal(ctx, &jobv1.GetProposalRequest{Id: propID})
		require.NoError(t, err)
		assert.Equal(t, jobv1.ProposalStatus_PROPOSAL_STATUS_REVOKED, getResp.Proposal.Status)
	})

	t.Run("revoke with UUID is not supported", func(t *testing.T) {
		t.Parallel()

		client, _, _ := setup(t)
		ctx := t.Context()
		_, err := client.RevokeJob(ctx, &jobv1.RevokeJobRequest{
			IdOneof: &jobv1.RevokeJobRequest_Uuid{Uuid: uuid.New().String()},
		})
		require.ErrorContains(t, err, "uuid lookup is deprecated and not supported by this implementation")
	})

	t.Run("revoke non-existent job returns error", func(t *testing.T) {
		t.Parallel()

		client, _, _ := setup(t)
		ctx := t.Context()
		_, err := client.RevokeJob(ctx, &jobv1.RevokeJobRequest{
			IdOneof: &jobv1.RevokeJobRequest_Id{Id: "non-existent"},
		})
		require.ErrorContains(t, err, "not found")
	})

	t.Run("revoke job with invalid id returns error", func(t *testing.T) {
		t.Parallel()

		client, _, _ := setup(t)
		ctx := t.Context()
		_, err := client.RevokeJob(ctx, &jobv1.RevokeJobRequest{
			IdOneof: &jobv1.RevokeJobRequest_Id{Id: ""},
		})
		require.ErrorContains(t, err, "job id must be provided")
	})

	t.Run("highest revision proposal is revoked", func(t *testing.T) {
		t.Parallel()

		client, prop1ID, jobID := setup(t)
		ctx := t.Context()

		// Propose another job with the same job id to get a new proposal with a higher revision
		// number
		proposeResp, err := client.ProposeJob(ctx, &jobv1.ProposeJobRequest{
			NodeId: "test-node-1",
			Spec:   spec1,
		})
		require.NoError(t, err)

		// Revoke the job
		revokeResp, err := client.RevokeJob(ctx, &jobv1.RevokeJobRequest{
			IdOneof: &jobv1.RevokeJobRequest_Id{Id: jobID},
		})
		require.NoError(t, err)
		require.NotNil(t, revokeResp)

		// Verify original proposal status is not revoked
		getResp, err := client.GetProposal(ctx, &jobv1.GetProposalRequest{Id: prop1ID})
		require.NoError(t, err)
		assert.Equal(t, jobv1.ProposalStatus_PROPOSAL_STATUS_PROPOSED, getResp.Proposal.Status)

		// Verify proposal status changed
		getResp2, err := client.GetProposal(ctx, &jobv1.GetProposalRequest{Id: proposeResp.Proposal.Id})
		require.NoError(t, err)
		assert.Equal(t, jobv1.ProposalStatus_PROPOSAL_STATUS_REVOKED, getResp2.Proposal.Status)
	})

	t.Run("job has no proposals (should never happen) returns error", func(t *testing.T) {
		t.Parallel()

		client, propID, jobID := setup(t)
		ctx := t.Context()

		// Delete the proposal by accessing the map directly
		delete(client.proposals, propID)

		_, err := client.RevokeJob(ctx, &jobv1.RevokeJobRequest{
			IdOneof: &jobv1.RevokeJobRequest_Id{Id: jobID},
		})
		require.ErrorContains(t, err, "not found")
	})

	t.Run("job is not in the correct state to be revoked returns error", func(t *testing.T) {
		t.Parallel()

		client, propID, jobID := setup(t)
		ctx := t.Context()

		// Update the proposal status to a state that is not allowed to be revoked
		client.proposals[propID].Status = jobv1.ProposalStatus_PROPOSAL_STATUS_APPROVED

		_, err := client.RevokeJob(ctx, &jobv1.RevokeJobRequest{
			IdOneof: &jobv1.RevokeJobRequest_Id{Id: jobID},
		})
		require.ErrorContains(t, err, "job cannot be revoked")
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

		t.Run("delete job by id", func(t *testing.T) {
			t.Parallel()

			// Create a job
			proposeResp, err := client.ProposeJob(ctx, &jobv1.ProposeJobRequest{
				NodeId: "test-node-1",
				Spec:   spec1,
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

		t.Run("job not found", func(t *testing.T) {
			t.Parallel()

			_, err := client.DeleteJob(t.Context(), &jobv1.DeleteJobRequest{
				IdOneof: &jobv1.DeleteJobRequest_Id{Id: "non-existent"},
			})
			require.ErrorContains(t, err, "not found")
		})
	})

	t.Run("invalid requests", func(t *testing.T) {
		t.Parallel()

		client := NewMemoryJobDistributor()

		t.Run("nil request", func(t *testing.T) {
			t.Parallel()

			_, err := client.DeleteJob(t.Context(), nil)
			require.ErrorContains(t, err, "request cannot be nil")
		})

		t.Run("invalid id oneof", func(t *testing.T) {
			t.Parallel()

			_, err := client.DeleteJob(t.Context(), &jobv1.DeleteJobRequest{})
			require.ErrorContains(t, err, "must provide id")
		})

		t.Run("empty id", func(t *testing.T) {
			t.Parallel()

			_, err := client.DeleteJob(t.Context(), &jobv1.DeleteJobRequest{
				IdOneof: &jobv1.DeleteJobRequest_Id{Id: ""},
			})
			require.ErrorContains(t, err, "job id must be provided")
		})

		t.Run("delete job by uuid is not supported", func(t *testing.T) {
			t.Parallel()

			_, err := client.DeleteJob(t.Context(), &jobv1.DeleteJobRequest{
				IdOneof: &jobv1.DeleteJobRequest_Uuid{Uuid: uuid.New().String()},
			})
			require.ErrorContains(t, err, "uuid lookup is deprecated and not supported by this implementation")
		})
	})
}

func TestMemoryJobDistributor_UpdateJob(t *testing.T) {
	t.Parallel()

	client := NewMemoryJobDistributor()

	t.Run("update by id", func(t *testing.T) {
		t.Run("update existing job", func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			// Create a job
			value := "testvalue"
			proposeResp, err := client.ProposeJob(ctx, &jobv1.ProposeJobRequest{
				NodeId: "test-node-1",
				Spec:   spec1,
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
				RollbackPolicyId: pointer.To(int64(100)),
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
			assert.EqualValues(t, 100, *getResp.Job.RollbackPolicyId)
		})

		t.Run("job not found", func(t *testing.T) {
			t.Parallel()

			_, err := client.UpdateJob(t.Context(), &jobv1.UpdateJobRequest{
				IdOneof: &jobv1.UpdateJobRequest_Id{Id: "non-existent"},
			})
			require.ErrorContains(t, err, "not found")
		})
	})

	t.Run("invalid requests", func(t *testing.T) {
		t.Parallel()

		client := NewMemoryJobDistributor()
		ctx := t.Context()

		t.Run("nil request", func(t *testing.T) {
			t.Parallel()
			_, err := client.UpdateJob(ctx, nil)
			require.ErrorContains(t, err, "request cannot be nil")
		})

		t.Run("invalid id oneof", func(t *testing.T) {
			t.Parallel()
			_, err := client.UpdateJob(ctx, &jobv1.UpdateJobRequest{})
			require.ErrorContains(t, err, "must provide id")
		})

		t.Run("empty id", func(t *testing.T) {
			t.Parallel()
			_, err := client.UpdateJob(ctx, &jobv1.UpdateJobRequest{
				IdOneof: &jobv1.UpdateJobRequest_Id{Id: ""},
			})
			require.ErrorContains(t, err, "job id must be provided")
		})

		t.Run("update job by uuid is not supported", func(t *testing.T) {
			t.Parallel()
			_, err := client.UpdateJob(ctx, &jobv1.UpdateJobRequest{
				IdOneof: &jobv1.UpdateJobRequest_Uuid{Uuid: uuid.New().String()},
			})
			require.ErrorContains(t, err, "uuid lookup is deprecated and not supported by this implementation")
		})
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
		assert.Equal(t, NodeVersion, resp.Node.Version)
		assert.NotNil(t, resp.Node.CreatedAt)
		assert.NotNil(t, resp.Node.UpdatedAt)
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

		t.Run("get node by id", func(t *testing.T) {
			t.Parallel()

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

		t.Run("node not found", func(t *testing.T) {
			t.Parallel()
			_, err := client.GetNode(t.Context(), &nodev1.GetNodeRequest{Id: "non-existent"})
			require.ErrorContains(t, err, "not found")
		})
	})

	t.Run("invalid requests", func(t *testing.T) {
		t.Parallel()
		client := NewMemoryJobDistributor()

		t.Run("get non-existent node returns error", func(t *testing.T) {
			t.Parallel()

			resp, err := client.GetNode(t.Context(), &nodev1.GetNodeRequest{Id: "non-existent"})
			require.ErrorContains(t, err, "not found")
			assert.Nil(t, resp)
		})
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

		t.Run("update node by id", func(t *testing.T) {
			t.Parallel()

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

		t.Run("node not found", func(t *testing.T) {
			t.Parallel()
			_, err := client.UpdateNode(t.Context(), &nodev1.UpdateNodeRequest{
				Id: "non-existent",
			})
			require.ErrorContains(t, err, "not found")
		})
	})

	t.Run("invalid requests", func(t *testing.T) {
		t.Parallel()

		client := NewMemoryJobDistributor()

		t.Run("nil request", func(t *testing.T) {
			t.Parallel()

			_, err := client.UpdateNode(t.Context(), nil)
			require.ErrorContains(t, err, "request cannot be nil")
		})
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

		t.Run("node not found: enable", func(t *testing.T) {
			t.Parallel()
			_, err := client.EnableNode(t.Context(), &nodev1.EnableNodeRequest{Id: "non-existent"})
			require.ErrorContains(t, err, "not found")
		})

		t.Run("node not found: disable", func(t *testing.T) {
			t.Parallel()
			_, err := client.DisableNode(t.Context(), &nodev1.DisableNodeRequest{Id: "non-existent"})
			require.ErrorContains(t, err, "not found")
		})
	})

	t.Run("invalid requests", func(t *testing.T) {
		t.Parallel()

		client := NewMemoryJobDistributor()

		t.Run("nil request: enable", func(t *testing.T) {
			t.Parallel()
			_, err := client.EnableNode(t.Context(), nil)
			require.ErrorContains(t, err, "request cannot be nil")
		})

		t.Run("nil request: disable", func(t *testing.T) {
			t.Parallel()
			_, err := client.DisableNode(t.Context(), nil)
			require.ErrorContains(t, err, "request cannot be nil")
		})
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

func TestMemoryJobDistributor_AddChainConfig(t *testing.T) {
	t.Parallel()

	client := NewMemoryJobDistributor()
	ctx := t.Context()

	t.Run("add chain config to node", func(t *testing.T) {
		t.Parallel()

		registerResp, err := client.RegisterNode(ctx, &nodev1.RegisterNodeRequest{
			Name:      "Test Node",
			PublicKey: "test-key",
		})
		require.NoError(t, err)
		nodeID := registerResp.Node.Id

		err = client.AddChainConfig(nodeID, &nodev1.ChainConfig{
			Chain: &nodev1.Chain{Id: "chain-1", Type: nodev1.ChainType_CHAIN_TYPE_EVM},
		})
		require.NoError(t, err)

		listResp, err := client.ListNodeChainConfigs(ctx, &nodev1.ListNodeChainConfigsRequest{
			Filter: &nodev1.ListNodeChainConfigsRequest_Filter{
				NodeIds: []string{nodeID},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, listResp)
		assert.Len(t, listResp.ChainConfigs, 1)
		assert.Equal(t, "chain-1", listResp.ChainConfigs[0].Chain.Id)
		assert.Equal(t, nodev1.ChainType_CHAIN_TYPE_EVM, listResp.ChainConfigs[0].Chain.Type)
	})

	t.Run("add chain config to non-existent node returns error", func(t *testing.T) {
		t.Parallel()
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

// assertJobEquals asserts that two jobv1.Job objects are functionally equal.
func assertJobEquals(t *testing.T, want *jobv1.Job, actual *jobv1.Job) {
	t.Helper()

	assert.True(t, strings.HasPrefix(actual.Id, "job_"))
	assert.ElementsMatch(t, want.ProposalIds, actual.ProposalIds)
	assert.Equal(t, want.NodeId, actual.NodeId)
	assert.Equal(t, want.Uuid, actual.Uuid)
	assert.ElementsMatch(t, want.Labels, actual.Labels)
	assert.NotNil(t, actual.CreatedAt)
	assert.NotNil(t, actual.UpdatedAt)
}

// assertProposalEquals asserts that two jobv1.Proposal objects are functionally equal.
func assertProposalEquals(t *testing.T, want *jobv1.Proposal, actual *jobv1.Proposal) {
	t.Helper()

	assert.True(t, strings.HasPrefix(actual.Id, "prop_"))
	assert.Equal(t, want.Revision, actual.Revision)
	assert.Equal(t, want.Status, actual.Status)
	assert.Equal(t, want.DeliveryStatus, actual.DeliveryStatus)
	assert.Equal(t, want.JobId, actual.JobId)
	assert.Equal(t, want.Spec, actual.Spec)
	assert.NotNil(t, actual.CreatedAt)
	assert.NotNil(t, actual.UpdatedAt)
}

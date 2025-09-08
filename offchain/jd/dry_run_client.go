package jd

import (
	"context"

	"google.golang.org/grpc"

	"github.com/smartcontractkit/chainlink-deployments-framework/offchain"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	csav1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/csa"
	jobv1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/job"
	nodev1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/node"
)

// DryRunJobDistributor is a readonly JD client.
// Read operations are forwarded to the real backend, while write operations are ignored.
type DryRunJobDistributor struct {
	// Used for read-only commands
	realBackend offchain.Client
	lggr        logger.Logger
}

var _ offchain.Client = (*DryRunJobDistributor)(nil)

// NewDryRunJobDistributor creates a new DryRunJobDistributor.
func NewDryRunJobDistributor(realBackend offchain.Client, lggr logger.Logger) *DryRunJobDistributor {
	return &DryRunJobDistributor{
		realBackend: realBackend,
		lggr:        lggr,
	}
}

// GetJob retrieves a specific job by its ID from the Job Distributor.
// This operation is forwarded to the real backend since it's a read-only operation.
func (d *DryRunJobDistributor) GetJob(ctx context.Context, in *jobv1.GetJobRequest, opts ...grpc.CallOption) (*jobv1.GetJobResponse, error) {
	d.lggr.Infow("DryRunJobDistributor.GetJob", "in", in)
	return d.realBackend.GetJob(ctx, in)
}

// GetProposal retrieves a specific job proposal by its ID from the Job Distributor.
// This operation is forwarded to the real backend since it's a read-only operation.
func (d *DryRunJobDistributor) GetProposal(ctx context.Context, in *jobv1.GetProposalRequest, opts ...grpc.CallOption) (*jobv1.GetProposalResponse, error) {
	d.lggr.Infow("DryRunJobDistributor.GetProposal", "in", in)
	return d.realBackend.GetProposal(ctx, in)
}

// ListJobs retrieves a list of all jobs from the Job Distributor.
// This operation is forwarded to the real backend since it's a read-only operation.
func (d *DryRunJobDistributor) ListJobs(ctx context.Context, in *jobv1.ListJobsRequest, opts ...grpc.CallOption) (*jobv1.ListJobsResponse, error) {
	d.lggr.Infow("DryRunJobDistributor.ListJobs", "in", in)
	return d.realBackend.ListJobs(ctx, in)
}

// ListProposals retrieves a list of all job proposals from the Job Distributor.
// This operation is forwarded to the real backend since it's a read-only operation.
func (d *DryRunJobDistributor) ListProposals(ctx context.Context, in *jobv1.ListProposalsRequest, opts ...grpc.CallOption) (*jobv1.ListProposalsResponse, error) {
	d.lggr.Infow("DryRunJobDistributor.ListProposals", "in", in)
	return d.realBackend.ListProposals(ctx, in)
}

// ProposeJob simulates proposing a new job to the Job Distributor without actually submitting it.
// In dry run mode, this returns a mock proposal response with a dummy job ID indicating
// the job was not actually proposed to the node.
func (d *DryRunJobDistributor) ProposeJob(ctx context.Context, in *jobv1.ProposeJobRequest, opts ...grpc.CallOption) (*jobv1.ProposeJobResponse, error) {
	d.lggr.Infow("DryRunJobDistributor.ProposeJob", "in", in)
	return &jobv1.ProposeJobResponse{
		Proposal: &jobv1.Proposal{
			JobId:  "dryRunJobId_NOT_PROPOSED_on_node_" + in.NodeId,
			Spec:   in.Spec,
			Status: jobv1.ProposalStatus_PROPOSAL_STATUS_UNSPECIFIED,
		},
	}, nil
}

// BatchProposeJob simulates proposing multiple jobs in a batch to the Job Distributor without actually submitting them.
// In dry run mode, this returns an empty response indicating the batch operation was logged but not executed.
func (d *DryRunJobDistributor) BatchProposeJob(ctx context.Context, in *jobv1.BatchProposeJobRequest, opts ...grpc.CallOption) (*jobv1.BatchProposeJobResponse, error) {
	d.lggr.Infow("DryRunJobDistributor.BatchProposeJob", "in", in)
	return &jobv1.BatchProposeJobResponse{}, nil
}

// RevokeJob simulates revoking a job from the Job Distributor without actually executing the revocation.
// In dry run mode, this returns an empty response indicating the revocation was logged but not executed.
func (d *DryRunJobDistributor) RevokeJob(ctx context.Context, in *jobv1.RevokeJobRequest, opts ...grpc.CallOption) (*jobv1.RevokeJobResponse, error) {
	d.lggr.Infow("DryRunJobDistributor.RevokeJob", "in", in)
	return &jobv1.RevokeJobResponse{}, nil
}

// DeleteJob simulates deleting a job from the Job Distributor without actually executing the deletion.
// In dry run mode, this returns an empty response indicating the deletion was logged but not executed.
func (d *DryRunJobDistributor) DeleteJob(ctx context.Context, in *jobv1.DeleteJobRequest, opts ...grpc.CallOption) (*jobv1.DeleteJobResponse, error) {
	d.lggr.Infow("DryRunJobDistributor.DeleteJob", "in", in)
	return &jobv1.DeleteJobResponse{}, nil
}

// UpdateJob simulates updating an existing job in the Job Distributor without actually executing the update.
// In dry run mode, this returns an empty response indicating the update was logged but not executed.
func (d *DryRunJobDistributor) UpdateJob(ctx context.Context, in *jobv1.UpdateJobRequest, opts ...grpc.CallOption) (*jobv1.UpdateJobResponse, error) {
	d.lggr.Infow("DryRunJobDistributor.UpdateJob", "in", in)
	return &jobv1.UpdateJobResponse{}, nil
}

// DisableNode simulates disabling a node in the Job Distributor without actually executing the operation.
// In dry run mode, this returns an empty response indicating the node disable operation was logged but not executed.
func (d *DryRunJobDistributor) DisableNode(ctx context.Context, in *nodev1.DisableNodeRequest, opts ...grpc.CallOption) (*nodev1.DisableNodeResponse, error) {
	d.lggr.Infow("DryRunJobDistributor.DisableNode", "in", in)
	return &nodev1.DisableNodeResponse{}, nil
}

// EnableNode simulates enabling a node in the Job Distributor without actually executing the operation.
// In dry run mode, this returns an empty response indicating the node enable operation was logged but not executed.
func (d *DryRunJobDistributor) EnableNode(ctx context.Context, in *nodev1.EnableNodeRequest, opts ...grpc.CallOption) (*nodev1.EnableNodeResponse, error) {
	d.lggr.Infow("DryRunJobDistributor.EnableNode", "in", in)
	return &nodev1.EnableNodeResponse{}, nil
}

// GetNode retrieves information about a specific node from the Job Distributor.
// This operation is forwarded to the real backend since it's a read-only operation.
func (d *DryRunJobDistributor) GetNode(ctx context.Context, in *nodev1.GetNodeRequest, opts ...grpc.CallOption) (*nodev1.GetNodeResponse, error) {
	d.lggr.Infow("DryRunJobDistributor.GetNode", "in", in)
	return d.realBackend.GetNode(ctx, in)
}

// ListNodes retrieves a list of all nodes registered with the Job Distributor.
// This operation is forwarded to the real backend since it's a read-only operation.
func (d *DryRunJobDistributor) ListNodes(ctx context.Context, in *nodev1.ListNodesRequest, opts ...grpc.CallOption) (*nodev1.ListNodesResponse, error) {
	d.lggr.Infow("DryRunJobDistributor.ListNodes", "in", in)
	return d.realBackend.ListNodes(ctx, in)
}

// ListNodeChainConfigs retrieves chain configuration information for nodes from the Job Distributor.
// This operation is forwarded to the real backend since it's a read-only operation.
func (d *DryRunJobDistributor) ListNodeChainConfigs(ctx context.Context, in *nodev1.ListNodeChainConfigsRequest, opts ...grpc.CallOption) (*nodev1.ListNodeChainConfigsResponse, error) {
	d.lggr.Infow("DryRunJobDistributor.ListNodeChainConfigs", "in", in)
	return d.realBackend.ListNodeChainConfigs(ctx, in)
}

// RegisterNode simulates registering a new node with the Job Distributor without actually executing the registration.
// In dry run mode, this returns an empty response indicating the node registration was logged but not executed.
func (d *DryRunJobDistributor) RegisterNode(ctx context.Context, in *nodev1.RegisterNodeRequest, opts ...grpc.CallOption) (*nodev1.RegisterNodeResponse, error) {
	d.lggr.Infow("DryRunJobDistributor.RegisterNode", "in", in)
	return &nodev1.RegisterNodeResponse{}, nil
}

// UpdateNode simulates updating an existing node in the Job Distributor without actually executing the update.
// In dry run mode, this returns an empty response indicating the node update was logged but not executed.
func (d *DryRunJobDistributor) UpdateNode(ctx context.Context, in *nodev1.UpdateNodeRequest, opts ...grpc.CallOption) (*nodev1.UpdateNodeResponse, error) {
	d.lggr.Infow("DryRunJobDistributor.UpdateNode", "in", in)
	return &nodev1.UpdateNodeResponse{}, nil
}

// GetKeypair retrieves a specific CSA keypair from the Job Distributor.
// This operation is forwarded to the real backend since it's a read-only operation.
func (d *DryRunJobDistributor) GetKeypair(ctx context.Context, in *csav1.GetKeypairRequest, opts ...grpc.CallOption) (*csav1.GetKeypairResponse, error) {
	d.lggr.Infow("DryRunJobDistributor.GetKeypair", "in", in)
	return d.realBackend.GetKeypair(ctx, in)
}

// ListKeypairs retrieves a list of all CSA keypairs from the Job Distributor.
// This operation is forwarded to the real backend since it's a read-only operation.
func (d *DryRunJobDistributor) ListKeypairs(ctx context.Context, in *csav1.ListKeypairsRequest, opts ...grpc.CallOption) (*csav1.ListKeypairsResponse, error) {
	d.lggr.Infow("DryRunJobDistributor.ListKeypairs", "in", in)
	return d.realBackend.ListKeypairs(ctx, in)
}

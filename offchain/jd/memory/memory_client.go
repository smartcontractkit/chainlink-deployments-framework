package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	csav1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/csa"
	jobv1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/job"
	nodev1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/node"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smartcontractkit/chainlink-deployments-framework/offchain"
)

var _ offchain.Client = (*MemoryJobDistributor)(nil)

// MemoryJobDistributor is an in-memory implementation of the Job Distributor client.
// It stores jobs, proposals, nodes, and keypairs in memory without persisting to any backend.
// This implementation is thread-safe and can be used concurrently from multiple goroutines.
type MemoryJobDistributor struct {
	mu sync.RWMutex // protects all fields below

	jobs      map[string]*jobv1.Job
	proposals map[string]*jobv1.Proposal
	nodes     map[string]*nodev1.Node
	keypairs  map[string]*csav1.Keypair
	// chainConfigs stores chain configurations per node
	chainConfigs map[string][]*nodev1.ChainConfig
}

// NewMemoryJobDistributor creates a new in-memory Job Distributor client.
func NewMemoryJobDistributor() *MemoryJobDistributor {
	return &MemoryJobDistributor{
		jobs:         make(map[string]*jobv1.Job),
		proposals:    make(map[string]*jobv1.Proposal),
		nodes:        make(map[string]*nodev1.Node),
		keypairs:     make(map[string]*csav1.Keypair),
		chainConfigs: make(map[string][]*nodev1.ChainConfig),
	}
}

// Job Service Methods

// ProposeJob creates a new job proposal and stores it in memory.
func (m *MemoryJobDistributor) ProposeJob(ctx context.Context, in *jobv1.ProposeJobRequest, opts ...grpc.CallOption) (*jobv1.ProposeJobResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	// Generate unique IDs
	proposalID := uuid.New().String()
	jobID := uuid.New().String()

	// Create the proposal
	proposal := &jobv1.Proposal{
		Id:     proposalID,
		JobId:  jobID,
		Spec:   in.Spec,
		Status: jobv1.ProposalStatus_PROPOSAL_STATUS_APPROVED,
	}

	// Also create a job based on the proposal
	job := &jobv1.Job{
		Id:     jobID,
		Uuid:   uuid.New().String(),
		NodeId: in.NodeId,
		Labels: in.Labels,
	}

	m.mu.Lock()
	m.proposals[proposalID] = proposal
	m.jobs[jobID] = job
	m.mu.Unlock()

	return &jobv1.ProposeJobResponse{
		Proposal: proposal,
	}, nil
}

// GetJob retrieves a job by its ID.
func (m *MemoryJobDistributor) GetJob(ctx context.Context, in *jobv1.GetJobRequest, opts ...grpc.CallOption) (*jobv1.GetJobResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	// Use GetId() method to access the oneof field
	jobID := in.GetId()

	if jobID == "" {
		return nil, status.Error(codes.InvalidArgument, "job id must be provided")
	}

	m.mu.RLock()
	job, exists := m.jobs[jobID]
	m.mu.RUnlock()

	if !exists {
		return nil, status.Errorf(codes.NotFound, "job with id %s not found", jobID)
	}

	return &jobv1.GetJobResponse{
		Job: job,
	}, nil
}

// ListJobs returns all jobs stored in memory.
func (m *MemoryJobDistributor) ListJobs(ctx context.Context, in *jobv1.ListJobsRequest, opts ...grpc.CallOption) (*jobv1.ListJobsResponse, error) {
	m.mu.RLock()
	allJobs := make([]*jobv1.Job, 0, len(m.jobs))
	for _, job := range m.jobs {
		allJobs = append(allJobs, job)
	}
	m.mu.RUnlock()

	// Apply filtering - always filter out soft-deleted jobs by default
	var filteredJobs []*jobv1.Job
	if in.Filter != nil {
		filteredJobs = applyJobFilter(allJobs, in.Filter)
	} else {
		// Create a default filter that excludes soft-deleted jobs
		defaultFilter := &jobv1.ListJobsRequest_Filter{
			IncludeDeleted: false,
		}
		filteredJobs = applyJobFilter(allJobs, defaultFilter)
	}

	return &jobv1.ListJobsResponse{
		Jobs: filteredJobs,
	}, nil
}

// GetProposal retrieves a proposal by its ID.
func (m *MemoryJobDistributor) GetProposal(ctx context.Context, in *jobv1.GetProposalRequest, opts ...grpc.CallOption) (*jobv1.GetProposalResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	m.mu.RLock()
	proposal, exists := m.proposals[in.GetId()]
	m.mu.RUnlock()

	if !exists {
		return nil, status.Errorf(codes.NotFound, "proposal with id %s not found", in.GetId())
	}

	return &jobv1.GetProposalResponse{
		Proposal: proposal,
	}, nil
}

// ListProposals returns all proposals stored in memory.
func (m *MemoryJobDistributor) ListProposals(ctx context.Context, in *jobv1.ListProposalsRequest, opts ...grpc.CallOption) (*jobv1.ListProposalsResponse, error) {
	m.mu.RLock()
	proposals := make([]*jobv1.Proposal, 0, len(m.proposals))
	for _, proposal := range m.proposals {
		proposals = append(proposals, proposal)
	}
	m.mu.RUnlock()

	return &jobv1.ListProposalsResponse{
		Proposals: proposals,
	}, nil
}

// BatchProposeJob creates multiple job proposals in a batch.
func (m *MemoryJobDistributor) BatchProposeJob(ctx context.Context, in *jobv1.BatchProposeJobRequest, opts ...grpc.CallOption) (*jobv1.BatchProposeJobResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	successResponses := make(map[string]*jobv1.ProposeJobResponse)

	m.mu.Lock()
	// Create a proposal for each node
	for _, nodeID := range in.GetNodeIds() {
		// Generate unique IDs
		proposalID := uuid.New().String()
		jobID := uuid.New().String()

		// Create the proposal
		proposal := &jobv1.Proposal{
			Id:     proposalID,
			JobId:  jobID,
			Spec:   in.GetSpec(),
			Status: jobv1.ProposalStatus_PROPOSAL_STATUS_APPROVED,
		}

		// Store the proposal
		m.proposals[proposalID] = proposal

		// Also create and store a job based on the proposal
		job := &jobv1.Job{
			Id:     jobID,
			NodeId: nodeID,
			Labels: in.GetLabels(),
		}
		m.jobs[jobID] = job

		// Add to success responses
		successResponses[nodeID] = &jobv1.ProposeJobResponse{
			Proposal: proposal,
		}
	}
	m.mu.Unlock()

	return &jobv1.BatchProposeJobResponse{
		SuccessResponses: successResponses,
	}, nil
}

// RevokeJob revokes an existing job proposal.
func (m *MemoryJobDistributor) RevokeJob(ctx context.Context, in *jobv1.RevokeJobRequest, opts ...grpc.CallOption) (*jobv1.RevokeJobResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	// Use GetId() method to access the oneof field
	jobID := in.GetId()

	if jobID == "" {
		return nil, status.Error(codes.InvalidArgument, "job id must be provided")
	}

	m.mu.Lock()
	// Find the proposal associated with this job
	var foundProposal *jobv1.Proposal
	for _, proposal := range m.proposals {
		if proposal.JobId == jobID {
			foundProposal = proposal
			break
		}
	}

	if foundProposal != nil {
		foundProposal.Status = jobv1.ProposalStatus_PROPOSAL_STATUS_REVOKED
	}
	m.mu.Unlock()

	if foundProposal != nil {
		return &jobv1.RevokeJobResponse{
			Proposal: foundProposal,
		}, nil
	}

	return &jobv1.RevokeJobResponse{}, nil
}

// DeleteJob removes a job from memory.
func (m *MemoryJobDistributor) DeleteJob(ctx context.Context, in *jobv1.DeleteJobRequest, opts ...grpc.CallOption) (*jobv1.DeleteJobResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	// Use GetId() method to access the oneof field
	jobID := in.GetId()

	if jobID == "" {
		return nil, status.Error(codes.InvalidArgument, "job id must be provided")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "job with id %s not found", jobID)
	}
	job.DeletedAt = &timestamppb.Timestamp{Seconds: time.Now().Unix()}

	return &jobv1.DeleteJobResponse{
		Job: job,
	}, nil
}

// UpdateJob updates an existing job in memory.
func (m *MemoryJobDistributor) UpdateJob(ctx context.Context, in *jobv1.UpdateJobRequest, opts ...grpc.CallOption) (*jobv1.UpdateJobResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	// Use GetId() method to access the oneof field
	jobID := in.GetId()

	if jobID == "" {
		return nil, status.Error(codes.InvalidArgument, "job id must be provided")
	}

	m.mu.Lock()
	job, exists := m.jobs[jobID]
	if !exists {
		m.mu.Unlock()
		return nil, status.Errorf(codes.NotFound, "job with id %s not found", jobID)
	}

	// Update the job labels if provided
	if in.GetLabels() != nil {
		job.Labels = in.GetLabels()
	}
	m.mu.Unlock()

	return &jobv1.UpdateJobResponse{
		Job: job,
	}, nil
}

// Node Service Methods

// RegisterNode registers a new node in memory.
func (m *MemoryJobDistributor) RegisterNode(ctx context.Context, in *nodev1.RegisterNodeRequest, opts ...grpc.CallOption) (*nodev1.RegisterNodeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	// Generate a new ID for the node
	nodeID := uuid.New().String()

	// Create the node
	node := &nodev1.Node{
		Id:          nodeID,
		Name:        in.Name,
		PublicKey:   in.PublicKey,
		IsEnabled:   true,
		IsConnected: false,
		Labels:      in.Labels,
	}

	m.mu.Lock()
	m.nodes[nodeID] = node
	m.mu.Unlock()

	return &nodev1.RegisterNodeResponse{
		Node: node,
	}, nil
}

// GetNode retrieves a node by its ID.
func (m *MemoryJobDistributor) GetNode(ctx context.Context, in *nodev1.GetNodeRequest, opts ...grpc.CallOption) (*nodev1.GetNodeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	m.mu.RLock()
	node, exists := m.nodes[in.Id]
	m.mu.RUnlock()

	if !exists {
		return nil, status.Errorf(codes.NotFound, "node with id %s not found", in.Id)
	}

	return &nodev1.GetNodeResponse{
		Node: node,
	}, nil
}

// ListNodes returns all nodes stored in memory.
func (m *MemoryJobDistributor) ListNodes(ctx context.Context, in *nodev1.ListNodesRequest, opts ...grpc.CallOption) (*nodev1.ListNodesResponse, error) {
	m.mu.RLock()
	allNodes := make([]*nodev1.Node, 0, len(m.nodes))
	for _, node := range m.nodes {
		allNodes = append(allNodes, node)
	}
	m.mu.RUnlock()

	// Apply filtering if filter is provided
	var filteredNodes []*nodev1.Node
	if in.Filter != nil {
		filteredNodes = applyNodeFilter(allNodes, in.Filter)
	} else {
		filteredNodes = allNodes
	}

	return &nodev1.ListNodesResponse{
		Nodes: filteredNodes,
	}, nil
}

// UpdateNode updates an existing node in memory.
func (m *MemoryJobDistributor) UpdateNode(ctx context.Context, in *nodev1.UpdateNodeRequest, opts ...grpc.CallOption) (*nodev1.UpdateNodeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	m.mu.Lock()
	node, exists := m.nodes[in.Id]
	if !exists {
		m.mu.Unlock()
		return nil, status.Errorf(codes.NotFound, "node with id %s not found", in.Id)
	}

	// Update fields if provided
	if in.Name != "" {
		node.Name = in.Name
	}
	if in.PublicKey != "" {
		node.PublicKey = in.PublicKey
	}
	if in.Labels != nil {
		node.Labels = in.Labels
	}
	m.mu.Unlock()

	return &nodev1.UpdateNodeResponse{
		Node: node,
	}, nil
}

// EnableNode enables a disabled node.
func (m *MemoryJobDistributor) EnableNode(ctx context.Context, in *nodev1.EnableNodeRequest, opts ...grpc.CallOption) (*nodev1.EnableNodeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	m.mu.Lock()
	node, exists := m.nodes[in.Id]
	if !exists {
		m.mu.Unlock()
		return nil, status.Errorf(codes.NotFound, "node with id %s not found", in.Id)
	}

	node.IsEnabled = true
	m.mu.Unlock()

	return &nodev1.EnableNodeResponse{
		Node: node,
	}, nil
}

// DisableNode disables a node.
func (m *MemoryJobDistributor) DisableNode(ctx context.Context, in *nodev1.DisableNodeRequest, opts ...grpc.CallOption) (*nodev1.DisableNodeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	m.mu.Lock()
	node, exists := m.nodes[in.Id]
	if !exists {
		m.mu.Unlock()
		return nil, status.Errorf(codes.NotFound, "node with id %s not found", in.Id)
	}

	node.IsEnabled = false
	m.mu.Unlock()

	return &nodev1.DisableNodeResponse{
		Node: node,
	}, nil
}

// ListNodeChainConfigs returns chain configurations for nodes.
func (m *MemoryJobDistributor) ListNodeChainConfigs(ctx context.Context, in *nodev1.ListNodeChainConfigsRequest, opts ...grpc.CallOption) (*nodev1.ListNodeChainConfigsResponse, error) {
	m.mu.RLock()
	chainConfigs := make([]*nodev1.ChainConfig, 0)

	// If a specific node is requested, return its configs
	if in.GetFilter() != nil && in.GetFilter().NodeIds != nil && len(in.GetFilter().NodeIds) > 0 {
		for _, nodeID := range in.GetFilter().NodeIds {
			if configs, exists := m.chainConfigs[nodeID]; exists {
				chainConfigs = append(chainConfigs, configs...)
			}
		}
	} else {
		// Return all chain configs
		for _, configs := range m.chainConfigs {
			chainConfigs = append(chainConfigs, configs...)
		}
	}
	m.mu.RUnlock()

	return &nodev1.ListNodeChainConfigsResponse{
		ChainConfigs: chainConfigs,
	}, nil
}

// AddChainConfig is a helper method to add chain configurations for testing purposes.
// This is not part of the standard interface but is useful for setting up test data.
func (m *MemoryJobDistributor) AddChainConfig(nodeID string, config *nodev1.ChainConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.nodes[nodeID]; !exists {
		return fmt.Errorf("node with id %s not found", nodeID)
	}

	m.chainConfigs[nodeID] = append(m.chainConfigs[nodeID], config)

	return nil
}

// CSA Service Methods

// GetKeypair retrieves the first CSA keypair (simulates getting the active keypair).
func (m *MemoryJobDistributor) GetKeypair(ctx context.Context, in *csav1.GetKeypairRequest, opts ...grpc.CallOption) (*csav1.GetKeypairResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return the first keypair if any exist
	for _, keypair := range m.keypairs {
		return &csav1.GetKeypairResponse{
			Keypair: keypair,
		}, nil
	}

	return nil, status.Error(codes.NotFound, "no keypairs found")
}

// ListKeypairs returns all CSA keypairs stored in memory.
func (m *MemoryJobDistributor) ListKeypairs(ctx context.Context, in *csav1.ListKeypairsRequest, opts ...grpc.CallOption) (*csav1.ListKeypairsResponse, error) {
	m.mu.RLock()
	keypairs := make([]*csav1.Keypair, 0, len(m.keypairs))
	for _, keypair := range m.keypairs {
		keypairs = append(keypairs, keypair)
	}
	m.mu.RUnlock()

	return &csav1.ListKeypairsResponse{
		Keypairs: keypairs,
	}, nil
}

// AddKeypair is a helper method to add a CSA keypair for testing purposes.
// This is not part of the standard interface but is useful for setting up test data.
func (m *MemoryJobDistributor) AddKeypair(keypair *csav1.Keypair) {
	if keypair != nil && keypair.PublicKey != "" {
		m.mu.Lock()
		m.keypairs[keypair.PublicKey] = keypair
		m.mu.Unlock()
	}
}

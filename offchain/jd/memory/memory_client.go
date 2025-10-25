package memory

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pelletier/go-toml/v2"
	csav1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/csa"
	jobv1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/job"
	nodev1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/node"
	"github.com/smartcontractkit/chainlink-protos/job-distributor/v1/shared/ptypes"
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
func (m *MemoryJobDistributor) ProposeJob(
	ctx context.Context, in *jobv1.ProposeJobRequest, opts ...grpc.CallOption,
) (*jobv1.ProposeJobResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	proposal, err := m.upsertProposal(in.NodeId, in.Spec, in.Labels)
	if err != nil {
		return nil, err
	}

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
		proposal, err := m.upsertProposal(nodeID, in.GetSpec(), in.GetLabels())
		if err != nil {
			return nil, err
		}

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

	var jobID string
	switch idOneof := in.GetIdOneof().(type) {
	case *jobv1.RevokeJobRequest_Id:
		if idOneof.Id == "" {
			return nil, status.Error(codes.InvalidArgument, "job id must be provided")
		}

		jobID = idOneof.Id
	case *jobv1.RevokeJobRequest_Uuid:
		return nil, status.Error(codes.InvalidArgument, "uuid is not supported")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Find the proposal with the highest revision number associated with this job
	prop := m.getHighestRevisionProposalByJobID(jobID)
	if prop == nil {
		return nil, status.Errorf(codes.NotFound, "proposal with job id %s not found", jobID)
	}

	if !slices.Contains([]jobv1.ProposalStatus{jobv1.ProposalStatus_PROPOSAL_STATUS_PROPOSED, jobv1.ProposalStatus_PROPOSAL_STATUS_CANCELLED}, prop.Status) {
		return nil, errors.New("job cannot be revoked")
	}

	prop.Status = jobv1.ProposalStatus_PROPOSAL_STATUS_REVOKED

	return &jobv1.RevokeJobResponse{
		Proposal: prop,
	}, nil
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
	nodeID := newNodeID()

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

// getJobByUUID retrieves a job by its UUID and node ID.
func (m *MemoryJobDistributor) getJobByUUIDAndNodeID(uuid string, nodeID string) (*jobv1.Job, error) {
	for _, job := range m.jobs {
		if job.Uuid == uuid && job.NodeId == nodeID {
			return job, nil
		}
	}

	return nil, fmt.Errorf("job with uuid %s not found", uuid)
}

// getNextRevisionNum returns the next revision number for a given job ID.
func (m *MemoryJobDistributor) getNextRevisionNum(jobID string) int64 {
	return m.proposalsByJobCount(jobID) + 1
}

// getHighestRevisionProposalByJobID returns the proposal with the highest revision number for a
// given job ID.
func (m *MemoryJobDistributor) getHighestRevisionProposalByJobID(jobID string) *jobv1.Proposal {
	var prop *jobv1.Proposal
	var maxRevision int64 = -1

	for _, p := range m.proposals {
		if p.JobId == jobID && p.Revision > maxRevision {
			prop = p
			maxRevision = prop.Revision
		}
	}

	return prop
}

// proposalsByJobCount returns the number of proposals for a given job ID.
func (m *MemoryJobDistributor) proposalsByJobCount(jobID string) int64 {
	count := 0
	for _, proposal := range m.proposals {
		if proposal.JobId == jobID {
			count++
		}
	}

	return int64(count)
}

// upsertProposal upserts a proposal for a given spec
func (m *MemoryJobDistributor) upsertProposal(
	nodeID string, spec string, labels []*ptypes.Label,
) (*jobv1.Proposal, error) {
	specUUID, err := getSpecUUID(spec)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid spec: %v", err)
	}

	// Generate unique IDs
	proposalID := newProposalID()
	jobID := newJobID()
	now := timestamppb.Now()

	// If the job already exists, we use the existing job ID, otherwise we create a new job
	if job, _ := m.getJobByUUIDAndNodeID(specUUID.String(), nodeID); job != nil {
		jobID = job.Id
	} else {
		m.jobs[jobID] = &jobv1.Job{
			Id:     jobID,
			Uuid:   specUUID.String(),
			NodeId: nodeID,
			Labels: labels,
		}
	}

	// Insert the proposal
	m.proposals[proposalID] = &jobv1.Proposal{
		Id:             proposalID,
		Revision:       m.getNextRevisionNum(jobID),
		Status:         jobv1.ProposalStatus_PROPOSAL_STATUS_PROPOSED,
		DeliveryStatus: jobv1.ProposalDeliveryStatus_PROPOSAL_DELIVERY_STATUS_DELIVERED,
		JobId:          jobID,
		Spec:           spec,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	return m.proposals[proposalID], nil
}

// getSpecUUID extracts the UUID from a spec
func getSpecUUID(spec string) (uuid.UUID, error) {
	s := struct {
		ExternalJobID *uuid.UUID `toml:"externalJobID,omitempty"`
	}{}

	d := toml.NewDecoder(strings.NewReader(spec))

	if err := d.Decode(&s); err != nil {
		return uuid.Nil, err
	}

	if s.ExternalJobID == nil {
		return uuid.Nil, errors.New("externalJobID field not found in spec")
	}

	return *s.ExternalJobID, nil
}

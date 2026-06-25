package deployment

import (
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"

	"github.com/smartcontractkit/mcms"
	mcms_types "github.com/smartcontractkit/mcms/types"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
)

// MCMSTimelockProposalSpec is the input for building one MCMS timelock proposal.
type MCMSTimelockProposalSpec struct {
	Input    MCMSTimelockProposalInput
	BatchOps []mcms_types.BatchOperation
}

// OutputBuilder builds a ChangesetOutput, including MCMS timelock proposals from MCMSTimelockProposalSpec entries.
//
// Multiple proposals: call WithTimelockProposal once per proposal. Each spec is built independently.
//
// StartingOpCount: GetChainMetadata is invoked per proposal from current environment state. This builder does
// not advance op counts across proposals. When multiple proposals target the same chain in one changeset,
// coordinate StartingOpCount outside the builder before Build.
type OutputBuilder struct {
	registry        *MCMSReaderRegistry
	environment     Environment
	proposalSpecs   []MCMSTimelockProposalSpec
	changesetOutput ChangesetOutput
}

// NewOutputBuilder creates a new OutputBuilder with the given environment and data store.
func NewOutputBuilder(e Environment, newDS datastore.MutableDataStore) *OutputBuilder {
	return &OutputBuilder{
		environment:   e,
		proposalSpecs: []MCMSTimelockProposalSpec{},
		changesetOutput: ChangesetOutput{
			DataStore: newDS,
		},
	}
}

// WithMCMSReaderRegistry overrides the MCMS reader registry used during Build (default: GetMCMSReaderRegistry).
// In most cases, you should not need to use this method.
func (b *OutputBuilder) WithMCMSReaderRegistry(registry *MCMSReaderRegistry) *OutputBuilder {
	b.registry = registry
	return b
}

type batchOpsConfig struct {
	mergePerChain bool
}

// BatchOpsOption configures how batch operations are processed for a timelock proposal.
type BatchOpsOption func(*batchOpsConfig)

// WithoutMergeBatchOpsPerChain keeps batch operations as provided, only filtering out empty operations.
func WithoutMergeBatchOpsPerChain() BatchOpsOption {
	return func(c *batchOpsConfig) {
		c.mergePerChain = false
	}
}

// WithTimelockProposal appends an MCMS timelock proposal to build at the end of Build.
// Empty batch operations (no transactions) are filtered out. By default, multiple batch operations for the same
// chain are merged into one, preserving transaction order. Use WithoutMergeBatchOpsPerChain to keep operations
// separate per chain.
func (b *OutputBuilder) WithTimelockProposal(
	input MCMSTimelockProposalInput,
	ops []mcms_types.BatchOperation,
	opts ...BatchOpsOption,
) *OutputBuilder {
	b.proposalSpecs = append(b.proposalSpecs, MCMSTimelockProposalSpec{
		Input:    input,
		BatchOps: processBatchOps(ops, opts...),
	})

	return b
}

// Build constructs the final ChangesetOutput, including one MCMS timelock proposal per non-empty spec.
// On error, returns the accumulated ChangesetOutput (data store and any proposals built so far)
// together with an error. The error index refers to the position in the appended proposal spec list (including
// specs skipped because they have no batch operations after processing).
func (b *OutputBuilder) Build() (ChangesetOutput, error) {
	proposals := make([]mcms.TimelockProposal, 0, len(b.proposalSpecs))
	for i, spec := range b.proposalSpecs {
		if len(spec.BatchOps) == 0 {
			continue
		}
		proposal, err := b.buildTimelockProposal(spec)
		if err != nil {
			b.changesetOutput.MCMSTimelockProposals = proposals
			return b.changesetOutput, fmt.Errorf("timelock proposal spec at index %d: %w", i, err)
		}
		proposals = append(proposals, proposal)
	}

	b.changesetOutput.MCMSTimelockProposals = proposals

	return b.changesetOutput, nil
}

func (b *OutputBuilder) buildTimelockProposal(spec MCMSTimelockProposalSpec) (mcms.TimelockProposal, error) {
	if err := spec.Input.Validate(); err != nil {
		return mcms.TimelockProposal{}, fmt.Errorf("failed to validate MCMS timelock proposal input: %w", err)
	}

	timelockAddresses, chainMetadata, err := b.resolveMCMSPerChain(spec.Input, spec.BatchOps)
	if err != nil {
		return mcms.TimelockProposal{}, err
	}

	proposal, err := mcms.NewTimelockProposalBuilder().
		SetVersion("v1").
		SetDescription(spec.Input.Description).
		SetOverridePreviousRoot(spec.Input.OverridePreviousRoot).
		SetValidUntil(spec.Input.ValidUntil).
		SetDelay(spec.Input.TimelockDelay).
		SetAction(spec.Input.TimelockAction).
		SetOperations(spec.BatchOps).
		SetTimelockAddresses(timelockAddresses).
		SetChainMetadata(chainMetadata).
		Build()
	if err != nil {
		return mcms.TimelockProposal{}, fmt.Errorf("failed to build MCMS proposal: %w", err)
	}

	return *proposal, nil
}

func (b *OutputBuilder) mcmsRegistry() *MCMSReaderRegistry {
	if b.registry != nil {
		return b.registry
	}

	return GetMCMSReaderRegistry()
}

func processBatchOps(ops []mcms_types.BatchOperation, opts ...BatchOpsOption) []mcms_types.BatchOperation {
	cfg := batchOpsConfig{mergePerChain: true}
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.mergePerChain {
		return mergeBatchOpsPerChain(ops)
	}

	return filterNonEmptyBatchOps(ops)
}

func filterNonEmptyBatchOps(ops []mcms_types.BatchOperation) []mcms_types.BatchOperation {
	filtered := make([]mcms_types.BatchOperation, 0, len(ops))
	for _, op := range ops {
		if len(op.Transactions) > 0 {
			filtered = append(filtered, op)
		}
	}

	return filtered
}

func mergeBatchOpsPerChain(ops []mcms_types.BatchOperation) []mcms_types.BatchOperation {
	txPerChain := make(map[mcms_types.ChainSelector][]mcms_types.Transaction)
	chainOrder := make([]mcms_types.ChainSelector, 0)
	for _, op := range ops {
		if len(op.Transactions) == 0 {
			continue
		}
		if _, seen := txPerChain[op.ChainSelector]; !seen {
			chainOrder = append(chainOrder, op.ChainSelector)
		}
		txPerChain[op.ChainSelector] = append(txPerChain[op.ChainSelector], op.Transactions...)
	}

	merged := make([]mcms_types.BatchOperation, 0, len(chainOrder))
	for _, chainSelector := range chainOrder {
		merged = append(merged, mcms_types.BatchOperation{
			ChainSelector: chainSelector,
			Transactions:  txPerChain[chainSelector],
		})
	}

	return merged
}

func (b *OutputBuilder) resolveMCMSPerChain(
	input MCMSTimelockProposalInput,
	ops []mcms_types.BatchOperation,
) (map[mcms_types.ChainSelector]string, map[mcms_types.ChainSelector]mcms_types.ChainMetadata, error) {
	timelocks := make(map[mcms_types.ChainSelector]string)
	metadata := make(map[mcms_types.ChainSelector]mcms_types.ChainMetadata)
	seen := make(map[mcms_types.ChainSelector]struct{})

	for _, op := range ops {
		if _, ok := seen[op.ChainSelector]; ok {
			continue
		}
		seen[op.ChainSelector] = struct{}{}
		chainSelector := op.ChainSelector

		family, err := chain_selectors.GetSelectorFamily(uint64(chainSelector))
		if err != nil {
			return nil, nil, fmt.Errorf("chain family for selector %d: %w", chainSelector, err)
		}
		reader, ok := b.mcmsRegistry().Get(family)
		if !ok {
			return nil, nil, fmt.Errorf("no MCMS reader registered for chain family '%s'", family)
		}

		timelockRef, err := reader.GetTimelockRef(b.environment, uint64(chainSelector), input)
		if err != nil {
			return nil, nil, fmt.Errorf("get timelock ref for chain %d: %w", chainSelector, err)
		}
		timelocks[chainSelector] = timelockRef.Address

		chainMetadata, err := reader.GetChainMetadata(b.environment, uint64(chainSelector), input)
		if err != nil {
			return nil, nil, fmt.Errorf("get chain metadata for chain %d: %w", chainSelector, err)
		}
		metadata[chainSelector] = chainMetadata
	}

	return timelocks, metadata, nil
}

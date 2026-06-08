// Package sequenceutils provides helpers for building deployment changesets from operations sequences.
//
// Use NewOnChainChangesetFromSequence to wrap a sequence that returns OnChainOutput into a
// ChangeSetV2 with datastore and MCMS timelock proposal integration.
package sequenceutils

import (
	"errors"
	"fmt"

	mcms_types "github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"
)

var (
	// ErrBatchOpsWithoutMCMSInput indicates the sequence returned batch operations without MCMS timelock proposal input.
	ErrBatchOpsWithoutMCMSInput = errors.New("batch operations require MCMS timelock proposal input")
)

// OnChainOutput is a standard output type for sequences that deploy contracts on-chain and perform write operations.
type OnChainOutput struct {
	// Metadata is written to the changeset datastore (see datastore.MetadataBundle).
	//   - Addresses: deployed contract addresses (Added)
	//   - Contracts: contract metadata keyed by address + chain selector (Upserted)
	//   - Chains: chain metadata keyed by chain selector (Upserted)
	//   - Env: one env metadata record per environment (Set)
	// If writing to a particular key, ensure that you populate all required fields.
	Metadata datastore.MetadataBundle
	// BatchOps are MCMS batch operations for a single timelock proposal. MCMS input comes from the changeset config.
	BatchOps []mcms_types.BatchOperation
}

type WithMCMS[CFG any] struct {
	// MCMS, when non-nil, is validated at verification time and used to build the timelock proposal when the
	// sequence returns batch operations with transactions. Apply fails if the sequence returns non-empty batch
	// operations without MCMS input.
	MCMS *deployment.MCMSTimelockProposalInput
	Cfg  CFG
}

// NewOnChainChangesetFromSequenceParams configures NewOnChainChangesetFromSequence.
type NewOnChainChangesetFromSequenceParams[IN any, DEP any, CFG any] struct {
	// Sequence is the operations.Sequence to execute.
	Sequence *operations.Sequence[IN, OnChainOutput, DEP]
	// ResolveInput resolves the input for the sequence based on the environment and changeset config.
	ResolveInput func(e deployment.Environment, cfg CFG) (IN, error)
	// ResolveDep resolves the dependencies for the sequence based on the environment and changeset config.
	ResolveDep func(e deployment.Environment, cfg CFG) (DEP, error)
	// Verify, if non-nil, performs additional validation beyond built-in MCMS checks, params, environment, and resolve.
	Verify func(e deployment.Environment, wrapped WithMCMS[CFG]) error
}

type onChainChangesetConfig struct {
	mcmsRegistry *deployment.MCMSReaderRegistry
}

// OnChainChangesetFromSequenceOption configures NewOnChainChangesetFromSequence.
type OnChainChangesetFromSequenceOption func(*onChainChangesetConfig)

// WithMCMSRegistry overrides the default MCMS reader registry (deployment.GetMCMSReaderRegistry).
// In most cases, you should not need to use this method.
func WithMCMSRegistry(registry *deployment.MCMSReaderRegistry) OnChainChangesetFromSequenceOption {
	return func(c *onChainChangesetConfig) {
		c.mcmsRegistry = registry
	}
}

// NewOnChainChangesetFromSequence creates a ChangeSetV2 from an operations.Sequence that
// deploys contracts on-chain and performs write operations. It executes the sequence,
// writes OnChainOutput.Metadata to a datastore, and optionally builds an MCMS timelock
// proposal from OnChainOutput.BatchOps.
//
// Config is passed as WithMCMS[CFG], which wraps the deployment-specific config (Cfg) and
// an optional MCMS timelock proposal input (MCMS). When the sequence returns batch
// operations with transactions, MCMS must be set or Apply returns
// ErrBatchOpsWithoutMCMSInput.
//
// Usage:
//
//	deploySeq := operations.NewSequence(
//	    "deploy-timelock",
//	    semver.MustParse("1.0.0"),
//	    "Deploy timelock contracts",
//	    func(b operations.Bundle, deps DeployDeps, in DeployInput) (OnChainOutput, error) {
//	        // Run operations and collect addresses and/or MCMS batch operations.
//	        return OnChainOutput{
//	            Metadata: datastore.MetadataBundle{
//	                Addresses: []datastore.AddressRef{timelockRef},
//	            },
//	            BatchOps: batchOps, // omit when no MCMS proposal is needed
//	        }, nil
//	    },
//	)
//
//	cs := NewOnChainChangesetFromSequence(
//	    NewOnChainChangesetFromSequenceParams[DeployInput, DeployDeps, DeployConfig]{
//	        Sequence: deploySeq,
//	        ResolveInput: func(e deployment.Environment, cfg DeployConfig) (DeployInput, error) {
//	            return DeployInput{ChainSelector: cfg.ChainSelector}, nil
//	        },
//	        ResolveDep: func(e deployment.Environment, cfg DeployConfig) (DeployDeps, error) {
//	            return DeployDeps{Chain: e.BlockChains.EVMChains()[cfg.ChainSelector]}, nil
//	        },
//	    },
//	)
//
//	wrapped := WithMCMS[DeployConfig]{
//	    Cfg: DeployConfig{ChainSelector: ethMainnetSelector},
//	    MCMS: &deployment.MCMSTimelockProposalInput{
//	        TimelockAction: mcms_types.TimelockActionSchedule,
//	        ValidUntil:     validUntil,
//	        TimelockDelay:  mcms_types.NewDuration(time.Hour),
//	        Description:    "schedule timelock ops",
//	    },
//	}
//	if err := cs.VerifyPreconditions(env, wrapped); err != nil {
//	    return err
//	}
//	out, err := cs.Apply(env, wrapped)
//	// out.DataStore holds deployed addresses; out.MCMSTimelockProposals holds proposals.
func NewOnChainChangesetFromSequence[IN any, DEP any, CFG any](
	params NewOnChainChangesetFromSequenceParams[IN, DEP, CFG],
	opts ...OnChainChangesetFromSequenceOption,
) deployment.ChangeSetV2[WithMCMS[CFG]] {
	cfg := onChainChangesetConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	sequenceID := func() string {
		if params.Sequence != nil {
			return params.Sequence.ID()
		}

		return "<nil>"
	}

	resolveSeqInputAndDeps := func(e deployment.Environment, wrapped WithMCMS[CFG]) (IN, DEP, error) {
		in, err := params.ResolveInput(e, wrapped.Cfg)
		if err != nil {
			var dep DEP
			return in, dep, fmt.Errorf("%w: failed to resolve input for sequence %s: %w",
				deployment.ErrInvalidConfig, sequenceID(), err)
		}
		dep, err := params.ResolveDep(e, wrapped.Cfg)
		if err != nil {
			return in, dep, fmt.Errorf("%w: failed to resolve dependencies for sequence %s: %w",
				deployment.ErrInvalidConfig, sequenceID(), err)
		}

		return in, dep, nil
	}

	verifyMCMS := func(wrapped WithMCMS[CFG]) error {
		if wrapped.MCMS == nil {
			return nil
		}
		if err := wrapped.MCMS.Validate(); err != nil {
			return fmt.Errorf("%w: invalid MCMS timelock proposal input for sequence %s: %w",
				deployment.ErrInvalidConfig, sequenceID(), err)
		}

		return nil
	}

	verify := func(e deployment.Environment, wrapped WithMCMS[CFG]) error {
		if err := verifyMCMS(wrapped); err != nil {
			return err
		}
		if params.Verify != nil {
			return params.Verify(e, wrapped)
		}

		return nil
	}

	validate := func(e deployment.Environment, wrapped WithMCMS[CFG]) error {
		if err := validateOnChainParams(params); err != nil {
			return err
		}
		if err := validateEnvironment(e); err != nil {
			return err
		}
		if err := verify(e, wrapped); err != nil {
			return err
		}
		_, _, err := resolveSeqInputAndDeps(e, wrapped)

		return err
	}

	changesetApply := func(e deployment.Environment, wrapped WithMCMS[CFG]) (deployment.ChangesetOutput, error) {
		input, deps, err := resolveSeqInputAndDeps(e, wrapped)
		if err != nil {
			return deployment.ChangesetOutput{}, err
		}
		report, err := operations.ExecuteSequence(e.OperationsBundle, params.Sequence, deps, input)
		if err != nil {
			return deployment.ChangesetOutput{Reports: report.ExecutionReports},
				fmt.Errorf("failed to execute sequence with ID %s: %w", params.Sequence.ID(), err)
		}

		return buildOnChainChangesetOutput(e, cfg, wrapped.MCMS, params.Sequence.ID(), report)
	}

	return deployment.CreateChangeSet(changesetApply, validate)
}

func buildOnChainChangesetOutput[IN any](
	e deployment.Environment,
	cfg onChainChangesetConfig,
	mcmsInput *deployment.MCMSTimelockProposalInput,
	sequenceID string,
	report operations.SequenceReport[IN, OnChainOutput],
) (deployment.ChangesetOutput, error) {
	ds := datastore.NewMemoryDataStore()
	if metaErr := ds.WriteMetadata(report.Output.Metadata); metaErr != nil {
		return deployment.ChangesetOutput{Reports: report.ExecutionReports},
			fmt.Errorf("failed to write metadata to datastore: %w", metaErr)
	}

	partialOutput := deployment.ChangesetOutput{
		Reports:   report.ExecutionReports,
		DataStore: ds,
	}

	builder := deployment.NewOutputBuilder(e, ds).
		WithOperationsReports(report.ExecutionReports)

	if cfg.mcmsRegistry != nil {
		builder = builder.WithMCMSReaderRegistry(cfg.mcmsRegistry)
	}

	if hasNonEmptyBatchOps(report.Output.BatchOps) {
		if mcmsInput == nil {
			return partialOutput,
				fmt.Errorf("%w: sequence %s returned batch operations: %w",
					deployment.ErrInvalidConfig, sequenceID, ErrBatchOpsWithoutMCMSInput)
		}
		builder = builder.WithTimelockProposal(*mcmsInput, report.Output.BatchOps)
	}

	out, err := builder.Build()
	if err != nil {
		return out, err
	}

	return out, nil
}

func hasNonEmptyBatchOps(ops []mcms_types.BatchOperation) bool {
	for _, op := range ops {
		if len(op.Transactions) > 0 {
			return true
		}
	}

	return false
}

func validateOnChainParams[IN any, DEP any, CFG any](params NewOnChainChangesetFromSequenceParams[IN, DEP, CFG]) error {
	if params.Sequence == nil {
		return fmt.Errorf("%w: sequence is required", deployment.ErrInvalidConfig)
	}
	if params.ResolveInput == nil {
		return fmt.Errorf("%w: ResolveInput is required", deployment.ErrInvalidConfig)
	}
	if params.ResolveDep == nil {
		return fmt.Errorf("%w: ResolveDep is required", deployment.ErrInvalidConfig)
	}

	return nil
}

func validateEnvironment(e deployment.Environment) error {
	if e.Logger == nil {
		return fmt.Errorf("%w: logger is required", deployment.ErrInvalidEnvironment)
	}
	if e.GetContext == nil {
		return fmt.Errorf("%w: GetContext is required", deployment.ErrInvalidEnvironment)
	}
	if e.OperationsBundle.Logger == nil || e.OperationsBundle.GetContext == nil {
		return fmt.Errorf("%w: OperationsBundle is not configured", deployment.ErrInvalidEnvironment)
	}

	return nil
}

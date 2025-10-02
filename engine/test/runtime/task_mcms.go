package runtime

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	mcmslib "github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/test/internal/mcmsutils"
)

var (
	_ Executable = &signProposalTask{}
	_ Executable = &executeProposalTask{}
	_ Executable = &signAndExecuteProposalTask{}
)

// SignProposalTask creates a new task for signing MCMS or Timelock proposals.
// The task will automatically detect the proposal type and apply the appropriate
// signing method when executed.
//
// Parameters:
//   - proposalID: The unique identifier of the proposal to sign from the runtime state
//   - privateKey: The ECDSA private key to use for signing the proposal
//
// Returns a signProposalTask that can be executed within the runtime framework.
func SignProposalTask(proposalID string, signingKeys ...*ecdsa.PrivateKey) signProposalTask {
	return signProposalTask{
		baseTask:    newBaseTask(),
		proposalID:  proposalID,
		signingKeys: signingKeys,
	}
}

// signProposalTask represents a task that signs an MCMS or Timelock proposal
// using a provided private key. The task automatically detects the proposal
// type and applies the appropriate signing logic.
type signProposalTask struct {
	*baseTask

	proposalID  string              // ID of the proposal to sign
	signingKeys []*ecdsa.PrivateKey // Private keys for signing
}

// Run executes the proposal signing task within the provided environment and state.
// This method retrieves the proposal from the state, determines its type, signs it
// with the provided private keys, and updates the proposal state with the signed version.
//
// The signing process supports both MCMS proposals and Timelock proposals, automatically
// detecting the proposal type and applying the appropriate signing logic. Multiple
// signatures may be provided to sign the proposal, or they can be accumulated on the same proposal
// by running this task multiple times with different keys.
//
// Returns an error if:
//   - The proposal cannot be found in the state
//   - The proposal type cannot be determined
//   - The signer cannot be created
//   - The signing process fails
//   - The proposal state cannot be updated
func (t signProposalTask) Run(e fdeployment.Environment, state *State) error {
	ctx := e.GetContext()

	propState, err := state.GetProposal(t.proposalID)
	if err != nil {
		return err
	}

	kind, err := propState.Kind()
	if err != nil {
		return err
	}

	// Create a new signer
	signer, err := mcmsutils.NewSigner()
	if err != nil {
		return fmt.Errorf("failed to create new signer: %w", err)
	}

	propJSON := propState.JSON
	for _, signingKey := range t.signingKeys {
		switch kind {
		case mcmstypes.KindProposal:
			propJSON, err = signProposal(ctx, propJSON, signingKey, signer)
		case mcmstypes.KindTimelockProposal:
			propJSON, err = signTimelockProposal(ctx, propJSON, signingKey, signer)
		default:
			return fmt.Errorf("unsupported proposal kind: %s", kind)
		}
		if err != nil {
			return err
		}
	}

	// Update the proposal state with the signed proposal
	if err := state.UpdateProposalJSON(t.proposalID, propJSON); err != nil {
		return fmt.Errorf("failed to update proposal state: %w", err)
	}

	return nil
}

// ExecuteProposalTask creates a new task for executing signed MCMS or Timelock proposals.
// The task will automatically detect the proposal type and apply the appropriate
// execution method when run.
//
// Parameters:
//   - proposalID: The unique identifier of the signed proposal to execute from the runtime state
//
// Returns an executeProposalTask that can be executed within the runtime framework.
//
// Note: The proposal must be properly signed before execution. Unsigned proposals
// will fail during the execution process.
func ExecuteProposalTask(proposalID string) executeProposalTask {
	return executeProposalTask{
		baseTask:    newBaseTask(),
		proposalID:  proposalID,
		newExecutor: newDefaultExecutor,
	}
}

// executeProposalTask represents a task that executes a signed MCMS or Timelock proposal
// on the target blockchain networks. The task automatically detects the proposal type
// and applies the appropriate execution logic.
type executeProposalTask struct {
	*baseTask

	proposalID  string // ID of the proposal to execute
	newExecutor func(e fdeployment.Environment) proposalExecutor
}

// Run executes the proposal execution task within the provided environment and state.
// This method retrieves the signed proposal from the state, determines its type,
// and executes it on the appropriate blockchain networks.
//
// For Timelock proposals, a random salt override is applied to ensure unique operation
// IDs in test environments where multiple proposals may have identical timestamps.
//
// Returns an error if:
//   - The proposal cannot be found in the state
//   - The proposal type cannot be determined
//   - The proposal decoding fails
//   - The execution process fails on any target chain
func (t executeProposalTask) Run(e fdeployment.Environment, state *State) error {
	ctx := e.GetContext()

	propState, err := state.GetProposal(t.proposalID)
	if err != nil {
		return err
	}

	if propState.IsExecuted {
		return fmt.Errorf("proposal already executed: %s", t.proposalID)
	}

	kind, err := propState.Kind()
	if err != nil {
		return fmt.Errorf("failed to get proposal kind: %w", err)
	}

	// Create a new executor
	executor := t.newExecutor(e)

	switch kind {
	case mcmstypes.KindProposal:
		prop, err := mcmsutils.DecodeProposal(propState.JSON)
		if err != nil {
			return fmt.Errorf("failed to decode MCMS proposal (id: %s): %w", t.proposalID, err)
		}

		if err = executor.ExecuteMCMS(ctx, prop); err != nil {
			return fmt.Errorf("failed to execute MCMS proposal (id: %s): %w", t.proposalID, err)
		}
	case mcmstypes.KindTimelockProposal:
		prop, err := mcmsutils.DecodeTimelockProposal(propState.JSON)
		if err != nil {
			return fmt.Errorf("failed to decode Timelock proposal (id: %s): %w", t.proposalID, err)
		}

		prop.SaltOverride = randomHash()

		if err = executor.ExecuteTimelock(ctx, prop); err != nil {
			return fmt.Errorf("failed to execute Timelock proposal (id: %s): %w", t.proposalID, err)
		}
	}

	// Now we mark the proposal as executed
	if err := state.MarkProposalExecuted(t.proposalID); err != nil {
		return fmt.Errorf("failed to mark proposal as executed: %w", err)
	}

	return nil
}

// SignAndExecuteProposalsTask creates a new task for signing and executing all pending proposals.
func SignAndExecuteProposalsTask(signingKeys []*ecdsa.PrivateKey) signAndExecuteProposalTask {
	return signAndExecuteProposalTask{
		baseTask:    newBaseTask(),
		signingKeys: signingKeys,
		newExecutor: newDefaultExecutor,
	}
}

// signAndExecuteProposalTask represents a task that signs and executes all pending proposals.
type signAndExecuteProposalTask struct {
	*baseTask
	signingKeys []*ecdsa.PrivateKey // Private keys for signing
	newExecutor func(e fdeployment.Environment) proposalExecutor
}

// Run executes the sign and execute proposal task within the provided environment and state.
// This method retrieves all pending proposals from the state, signs them with the provided private keys,
// and executes them on the appropriate blockchain networks.
//
// Returns an error if:
//   - The proposal cannot be signed
//   - The proposal cannot be executed
func (t signAndExecuteProposalTask) Run(e fdeployment.Environment, state *State) error {
	for _, p := range state.GetPendingProposals() {
		signTask := SignProposalTask(p.ID, t.signingKeys...)
		if err := signTask.Run(e, state); err != nil {
			return fmt.Errorf("failed to sign proposal: %w", err)
		}

		execTask := ExecuteProposalTask(p.ID)
		if t.newExecutor != nil { // Override the default executor if provided. Used for testing only.
			execTask.newExecutor = t.newExecutor
		}

		if err := execTask.Run(e, state); err != nil {
			return fmt.Errorf("failed to execute proposal: %w", err)
		}
	}

	return nil
}

// signProposal is a helper function that signs an MCMS proposal and returns the signed proposal
// as JSON.
//
// This function handles the complete signing workflow: decoding the proposal from JSON,
// applying the cryptographic signature, and encoding the result back to JSON.
//
// The signing process preserves all existing signatures on the proposal, allowing
// multiple signers to incrementally add their signatures to the same proposal.
func signProposal(
	ctx context.Context,
	propJSON string,
	privateKey *ecdsa.PrivateKey,
	signer *mcmsutils.Signer,
) (string, error) {
	p, err := mcmsutils.DecodeProposal(propJSON)
	if err != nil {
		return "", fmt.Errorf("failed to decode proposal: %w", err)
	}

	if err = signer.SignMCMS(ctx, p, privateKey); err != nil {
		return "", fmt.Errorf("failed to sign proposal: %w", err)
	}

	propJSON, err = mcmsutils.EncodeProposal(p)
	if err != nil {
		return "", fmt.Errorf("failed to encode proposal: %w", err)
	}

	return propJSON, nil
}

// signTimelockProposal is a helper function that signs a Timelock proposal and returns the signed
// proposal as JSON.
//
// This function handles the complete signing workflow for timelock proposals: decoding
// the proposal from JSON, applying the cryptographic signature, and encoding the result
// back to JSON.
//
// The signing process preserves all existing signatures on the proposal, allowing
// multiple signers to incrementally add their signatures to the same timelock proposal.
func signTimelockProposal(
	ctx context.Context,
	propJSON string,
	privateKey *ecdsa.PrivateKey,
	signer *mcmsutils.Signer,
) (string, error) {
	p, err := mcmsutils.DecodeTimelockProposal(propJSON)
	if err != nil {
		return "", fmt.Errorf("failed to decode timelock proposal: %w", err)
	}

	if err = signer.SignTimelock(ctx, p, privateKey); err != nil {
		return "", fmt.Errorf("failed to sign timelock proposal: %w", err)
	}

	propJSON, err = mcmsutils.EncodeTimelockProposal(p)
	if err != nil {
		return "", fmt.Errorf("failed to encode timelock proposal: %w", err)
	}

	return propJSON, nil
}

// randomHash generates a random 32-byte hash to use as a salt override for timelock proposals.
// This function is specifically used to prevent scheduling conflicts in test environments
// where multiple timelock proposals might be created with identical timestamps.
//
// In production, timelock proposals use their validUntil timestamp to generate operation IDs.
// However, in tests where proposals are often created within the same second, this can lead
// to "AlreadyScheduled" errors when multiple proposals have the same generated ID.
//
// Note: This function is designed for test use only. The error from rand.Read is intentionally
// ignored as cryptographic randomness is not critical for test salt generation.
func randomHash() *common.Hash {
	b := make([]byte, 32)
	_, _ = rand.Read(b) // Assignment for errcheck. Only used in tests so we can ignore.
	h := common.BytesToHash(b)

	return &h
}

// proposalExecutor is an interface that defines the methods for executing MCMS and timelock
// proposals.
type proposalExecutor interface {
	ExecuteMCMS(ctx context.Context, proposal *mcmslib.Proposal) error
	ExecuteTimelock(ctx context.Context, proposal *mcmslib.TimelockProposal) error
}

// newDefaultExecutor creates a new default proposal executor.`
func newDefaultExecutor(e fdeployment.Environment) proposalExecutor {
	return mcmsutils.NewExecutor(e)
}

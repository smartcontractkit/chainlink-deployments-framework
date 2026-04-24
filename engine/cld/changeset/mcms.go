package changeset

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	cldfenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
)

// ----- mcms timelock execution report types -----

type MCMSReportStatus string

const (
	StatusSuccess MCMSReportStatus = "SUCCESS"
	StatusNoOp    MCMSReportStatus = "NOOP"
	StatusFailed  MCMSReportStatus = "FAILED"
)

type mcmsReport[IN, OUT any] struct {
	ID        string           `json:"id"`
	Type      string           `json:"type"`
	Status    MCMSReportStatus `json:"status,omitempty"`
	Error     string           `json:"error,omitempty"`
	Timestamp time.Time        `json:"timestamp,omitzero"`
	Input     IN               `json:"input,omitempty"`
	Output    OUT              `json:"output,omitempty"`
}

type MCMSTimelockExecuteReportInput struct {
	Index            int                 `json:"index"`
	OperationID      gethcommon.Hash     `json:"operationID,omitzero"`
	ChainSelector    uint64              `json:"chainSelector"`
	TimelockAddress  string              `json:"timelockAddress"`
	MCMAddress       string              `json:"mcmAddress"`
	AdditionalFields json.RawMessage     `json:"additionalFields,omitempty,omitzero"`
	Changeset        MCMSReportChangeset `json:"changeset,omitzero"`
}

type MCMSTimelockExecuteReportOutput struct {
	TransactionResult mcmstypes.TransactionResult `json:"transactionResult"`
}

type MCMSReportChangeset struct {
	ID    string `json:"id"`
	Index int    `json:"index"`
	Name  string `json:"name,omitzero"`
}

type MCMSTimelockExecuteReport mcmsReport[MCMSTimelockExecuteReportInput, MCMSTimelockExecuteReportOutput]

const MCMSTimelockExecuteReportType = "timelock-execution"

// ----- fork context -----

// ForkContext provides information about the forked state of the environment, if applicable. It is
// exposed to hooks to allow them to adjust their behavior accordingly.
type ForkContext interface {
	ChainFamily() string
}

type EVMForkContext struct {
	ChainConfig cldfenv.ChainConfig
	Client      cldfenv.ForkedOnchainClient
}

func (*EVMForkContext) ChainFamily() string {
	return chainsel.FamilyEVM
}

// RunProposalHooks executes all post-proposal hooks for the given proposal and reports. It returns
// an error if any of the hooks fail.
// Execution order is:
//  1. Per-changeset post-proposal-hooks
//  2. Global post-proposal-hooks
func (r *ChangesetsRegistry) RunProposalHooks(
	key string, e fdeployment.Environment, proposal *mcms.TimelockProposal, input, config any,
	reports []MCMSTimelockExecuteReport, forkCtx ForkContext,
) error {
	applySnapshot, err := r.getApplySnapshot(key)
	if err != nil {
		return err
	}

	blockChains := e.BlockChains
	if forkCtx == nil {
		blockChains = blockChains.ReadOnly()
	}

	params := PostProposalHookParams{
		Env: ProposalHookEnv{
			Name:        e.Name,
			Logger:      e.Logger,
			BlockChains: blockChains,
			DataStore:   e.DataStore,
			Offchain:    e.Offchain,
			ForkContext: forkCtx,
			// TODO: CRE client
		},
		ChangesetKey: key,
		Proposal:     proposal,
		Input:        input,
		Config:       config,
		Reports:      reports,
	}

	for _, h := range applySnapshot.registryEntry.postProposalHooks {
		err := ExecuteHook(e, h.HookDefinition, func(ctx context.Context) error {
			return h.Func(ctx, params)
		})
		if err != nil {
			return fmt.Errorf("changeset post-proposal-hook %q failed: %w", h.Name, err)
		}
	}

	for _, h := range applySnapshot.globalPostProposalHooks {
		if err := ExecuteHook(e, h.HookDefinition, func(ctx context.Context) error {
			return h.Func(ctx, params)
		}); err != nil {
			return fmt.Errorf("global post-proposal-hook %q failed: %w", h.Name, err)
		}
	}

	return nil
}

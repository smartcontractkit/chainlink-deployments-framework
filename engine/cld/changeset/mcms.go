package changeset

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

type mcmsReport[IN, OUT any] struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Status    string    `json:"status,omitempty"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp,omitzero"`
	Input     IN        `json:"input,omitempty"`
	Output    OUT       `json:"output,omitempty"`
}

type MCMSTimelockExecuteReportInput struct {
	Index            int                 `json:"index"`
	OperationID      gethcommon.Hash     `json:"operationID,omitzero"`
	ChainSelector    uint64              `json:"chainSelector"`
	TimelockAddress  string              `json:"timelockAddress"`
	MCMAddress       string              `json:"mcmAddress"`
	AdditionalFields json.RawMessage     `json:"additionalFields,omitempty,omitzero"`
	Changeset        mcmsReportChangeset `json:"changeset,omitzero"`
}

type MCMSTimelockExecuteReportOutput struct {
	TransactionResult mcmstypes.TransactionResult `json:"transactionResult"`
}

type mcmsReportChangeset struct {
	Index int    `json:"index"`
	Name  string `json:"name,omitzero"`
}

type MCMSTimelockExecuteReport mcmsReport[MCMSTimelockExecuteReportInput, MCMSTimelockExecuteReportOutput]

const MCMSTimelockExecuteReportType = "timelock-execution"

func (r *ChangesetsRegistry) RunProposalHooks(
	key string, e fdeployment.Environment, proposal *mcms.TimelockProposal, input any, reports []MCMSTimelockExecuteReport,
) error {
	applySnapshot, err := r.getApplySnapshot(key)
	if err != nil {
		return err
	}

	params := PostProposalHookParams{
		Env: ProposalHookEnv{
			Name:        e.Name,
			Logger:      e.Logger,
			BlockChains: e.BlockChains.ReadOnly(),
		},
		ChangesetKey: key,
		Proposal:     proposal,
		Input:        input,
		Reports:      reports,
	}

	for _, h := range applySnapshot.registryEntry.postProposalHooks {
		err := ExecuteHook(e, h.HookDefinition, func(ctx context.Context) error {
			return h.Func(ctx, params)
		})
		if err != nil {
			return fmt.Errorf("post-hook %q failed: %w", h.Name, err)
		}
	}

	for _, h := range applySnapshot.globalPostProposalHooks {
		if err := ExecuteHook(e, h.HookDefinition, func(ctx context.Context) error {
			return h.Func(ctx, params)
		}); err != nil {
			return fmt.Errorf("global post-hook %q failed: %w", h.Name, err)
		}
	}

	return nil
}

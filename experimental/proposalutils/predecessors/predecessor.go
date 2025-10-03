package predecessors

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	"github.com/smartcontractkit/mcms"

	mcmstypes "github.com/smartcontractkit/mcms/types"
)

// ComputeHighestOpCountsFromPredecessors looks at the given predecessors,
// and for each chain present in 'newProposalData', computes the new starting
// op count as: current StartingOpCount (from the new proposal) + SUM of the
// number of ops across ALL predecessors that share the same chain selector
// AND the same MCM address. We assume that the predecessors are sorted
// from oldest to newest, so we can log them in that order.
// Returns:
//   - newStartPerChain: map[chain] = baseline + sum(ops)
func ComputeHighestOpCountsFromPredecessors(
	lggr logger.Logger,
	newProposalData ProposalsOpData,
	predecessors []PRView,
) map[mcmstypes.ChainSelector]uint64 {
	out := make(map[mcmstypes.ChainSelector]uint64, len(newProposalData))

	lggr.Infof("Computing new starting op counts from %d predecessors...", len(predecessors))

	for sel, cur := range newProposalData {
		lggr.Infof("New proposal data - chain %d: MCM=%s start=%d ops=%d", uint64(sel), cur.MCMAddress, cur.StartingOpCount, cur.OpsCount)
		baseline := cur.StartingOpCount
		var extra uint64

		for _, pred := range predecessors {
			other, ok := pred.ProposalData[sel]
			if !ok || !sameMCM(cur.MCMAddress, other.MCMAddress) {
				continue
			}

			start := other.StartingOpCount
			if start < baseline {
				// Stale predecessor: its proposed start is below current on-chain baseline.
				// Ignore to avoid double-counting or lowering.
				lggr.Warnf("Skipping stale predecessor PR#%d on chain %d: pred.start=%d < baseline (ie onchain-value)=%d",
					pred.Number, uint64(sel), start, baseline)

				continue
			}

			extra += other.OpsCount
			lggr.Infof("Counting predecessor PR#%d on chain %d: baseline=%d start=%d +ops=%d",
				pred.Number, uint64(sel), baseline, start, other.OpsCount)
		}

		out[sel] = baseline + extra
	}

	// Safety check for race conditions: get the latest predecessor and add their ops, it should be the same as out[sel]
	if len(predecessors) > 0 {
		for sel, cur := range newProposalData {
			// get most recent predecessor with same MCM address
			latestPredecessor := predecessors[len(predecessors)-1]
			other, ok := latestPredecessor.ProposalData[sel]
			if !ok || !sameMCM(cur.MCMAddress, other.MCMAddress) {
				continue
			}
			newStart := other.StartingOpCount + other.OpsCount
			if newStart > out[sel] {
				lggr.Warnf("sum of predecessors' ops on chain %d is %d, but latest predecessor PR#%d has ops=%d, adjusting new start from %d to %d. This can happen when a predecessor gets merged while this action is running, but is not executed on-chain", uint64(sel), out[sel]-cur.StartingOpCount, latestPredecessor.Number, other.OpsCount, out[sel], newStart)
				out[sel] = newStart
			}
		}
	}

	lggr.Infof("Computed new starting op counts: %v", out)

	return out
}

// ApplyHighestOpCountsToProposal updates the proposal file with the new starting op counts.
func ApplyHighestOpCountsToProposal(
	lggr logger.Logger,
	proposalPath string,
	newOpCounts map[mcmstypes.ChainSelector]uint64,
) error {
	// 1) load via mcms
	prop, err := mcms.LoadProposal(mcmstypes.KindTimelockProposal, proposalPath)
	if err != nil {
		return fmt.Errorf("load proposal: %w", err)
	}

	// 2) cast to concrete type to mutate ChainMetadata
	tp, ok := prop.(*mcms.TimelockProposal)
	if !ok {
		return fmt.Errorf("expected *mcms.TimelockProposal, got %T", prop)
	}

	// 3) bump starting op counts
	changed := false
	for sel, end := range newOpCounts {
		meta, ok := tp.ChainMetadata[sel]
		if !ok {
			continue
		}
		newStart := end
		if meta.StartingOpCount < newStart {
			lggr.Infof("Updated startingOpCount: chain %d → %d", uint64(sel), end)
			meta.StartingOpCount = newStart
			tp.ChainMetadata[sel] = meta
			changed = true
		} else {
			lggr.Warnf("Not updating startingOpCount for chain %d: current=%d, proposed=%d",
				uint64(sel), meta.StartingOpCount, newStart)
		}
	}

	if !changed {
		lggr.Infof("No startingOpCount changes needed for %s", proposalPath)
		return nil
	}

	// 4) write back using mcms helper
	f, err := os.OpenFile(proposalPath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0o600)
	if err != nil {
		return fmt.Errorf("open for write: %w", err)
	}
	defer f.Close()

	if err := mcms.WriteTimelockProposal(f, tp); err != nil {
		return fmt.Errorf("write proposal: %w", err)
	}

	return nil
}

// ParseProposalOpsData uses mcms for a local file path (current proposal) to get op counts.
func ParseProposalOpsData(ctx context.Context, filePath string) (ProposalsOpData, error) {
	proposal, err := mcms.LoadProposal(mcmstypes.KindTimelockProposal, filePath)
	if err != nil {
		return nil, fmt.Errorf("load proposal from %s: %w", filePath, err)
	}
	// convert to timelock proposal
	tp, ok := proposal.(*mcms.TimelockProposal)
	if !ok {
		return nil, fmt.Errorf("expected *mcms.TimelockProposal, got %T", proposal)
	}

	// Use conversion-aware counts
	counts, err := tp.OperationCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("converted operation counts: %w", err)
	}

	data := make(ProposalsOpData, len(proposal.ChainMetadatas()))
	for chain, meta := range proposal.ChainMetadatas() {
		data[chain] = McmOpData{
			MCMAddress:      strings.TrimSpace(meta.MCMAddress),
			StartingOpCount: meta.StartingOpCount,
			OpsCount:        counts[chain],
		}
	}

	return data, nil
}

// sameMCM is a tiny helper to check mcm addresses are equal
func sameMCM(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

package lanedetector

import (
	"context"
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/smartcontractkit/chainlink-ccip/chains/evm/gobindings/generated/latest/token_pool"

	chainutils "github.com/smartcontractkit/chainlink-deployments-framework/chain/utils"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotation"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/decoder"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/examples/ccip"
)

const (
	AnalyzerID     = "ccip.lane_detector"
	AnnotationLane = "ccip.lane"
)

// LaneDetectorAnalyzer is a proposal-level analyzer that identifies
// cross-chain lanes from symmetric applyChainUpdates calls between two chains.
type LaneDetectorAnalyzer struct{}

var _ analyzer.ProposalAnalyzer = (*LaneDetectorAnalyzer)(nil)

func (a *LaneDetectorAnalyzer) ID() string             { return AnalyzerID }
func (a *LaneDetectorAnalyzer) Dependencies() []string { return nil }

func (a *LaneDetectorAnalyzer) CanAnalyze(
	_ context.Context,
	_ analyzer.ProposalAnalyzeRequest,
	_ decoder.DecodedTimelockProposal,
) bool {
	return true
}

func (a *LaneDetectorAnalyzer) Analyze(
	_ context.Context,
	_ analyzer.ProposalAnalyzeRequest,
	proposal decoder.DecodedTimelockProposal,
) (annotation.Annotations, error) {
	type edge struct{ from, to uint64 }

	edgeSet := make(map[edge]struct{})

	for _, batch := range proposal.BatchOperations() {
		src := batch.ChainSelector()

		for _, call := range batch.Calls() {
			if !ccip.IsTokenPoolContract(call.ContractType()) {
				continue
			}
			if call.Name() != "applyChainUpdates" {
				continue
			}

			for _, remote := range extractRemoteSelectors(call.Inputs()) {
				if remote != src {
					edgeSet[edge{src, remote}] = struct{}{}
				}
			}
		}
	}

	// Find bidirectional pairs: A>B and B>A both exist.
	seen := make(map[edge]struct{})
	var lanes []string

	for e := range edgeSet {
		if _, ok := edgeSet[edge{e.to, e.from}]; !ok {
			continue
		}

		canonical := e
		if e.from > e.to {
			canonical = edge{e.to, e.from}
		}
		if _, ok := seen[canonical]; ok {
			continue
		}
		seen[canonical] = struct{}{}

		lanes = append(lanes, fmt.Sprintf("%s <-> %s",
			resolveChainName(canonical.from),
			resolveChainName(canonical.to)))
	}

	sort.Strings(lanes)

	anns := make(annotation.Annotations, len(lanes))
	for i, lane := range lanes {
		anns[i] = annotation.New(AnnotationLane, "string", lane)
	}

	return anns, nil
}

func extractRemoteSelectors(params decoder.DecodedParameters) []uint64 {
	for _, param := range params {
		if param.Name() != "chainsToAdd" && param.Name() != "chains" {
			continue
		}

		raw := param.RawValue()
		if raw == nil {
			continue
		}

		converted, ok := abi.ConvertType(raw, new([]token_pool.TokenPoolChainUpdate)).(*[]token_pool.TokenPoolChainUpdate)
		if !ok {
			continue
		}

		selectors := make([]uint64, len(*converted))
		for i, u := range *converted {
			selectors[i] = u.RemoteChainSelector
		}

		return selectors
	}

	return nil
}

func resolveChainName(sel uint64) string {
	info, err := chainutils.ChainInfo(sel)
	if err != nil {
		return fmt.Sprintf("chain-%d", sel)
	}

	return info.ChainName
}

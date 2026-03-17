package lanedetector

import (
	"context"
	"fmt"
	"sort"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/examples/ccip"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/format"
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
	_ analyzer.DecodedTimelockProposal,
) bool {
	return true
}

func (a *LaneDetectorAnalyzer) Analyze(
	_ context.Context,
	_ analyzer.ProposalAnalyzeRequest,
	proposal analyzer.DecodedTimelockProposal,
) (analyzer.Annotations, error) {
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

			updates, err := ccip.ParseChainUpdates(call.Inputs())
			if err != nil {
				return nil, fmt.Errorf("parse chain updates for %s on chain %d: %w", call.To(), src, err)
			}

			for _, u := range updates {
				if u.RemoteChainSelector != src {
					edgeSet[edge{src, u.RemoteChainSelector}] = struct{}{}
				}
			}
		}
	}

	// Find bidirectional pairs: A>B and B>A both exist.
	seen := make(map[edge]struct{})
	lanes := make([]string, 0, len(edgeSet))

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
			format.ResolveChainName(canonical.from),
			format.ResolveChainName(canonical.to)))
	}

	sort.Strings(lanes)

	anns := make(analyzer.Annotations, len(lanes))
	for i, lane := range lanes {
		anns[i] = analyzer.NewAnnotation(AnnotationLane, "string", lane)
	}

	return anns, nil
}

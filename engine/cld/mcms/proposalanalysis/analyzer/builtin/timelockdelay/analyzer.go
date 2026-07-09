package timelockdelay

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/smartcontractkit/mcms"
	mcmschainwrappers "github.com/smartcontractkit/mcms/chainwrappers"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	cldfmcmsadapters "github.com/smartcontractkit/chainlink-deployments-framework/chain/mcms/adapters"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/format"
)

type chainDelay struct {
	selector uint64
	minDelay mcmstypes.Duration
}

type timelockChainEntry struct {
	selector uint64
	address  string
}

// Validator checks that schedule proposals use a delay >= each executing
// timelock's on-chain minDelay.
type Validator struct{}

var _ analyzer.ProposalAnalyzer = Validator{}

func (Validator) ID() string             { return ValidatorID }
func (Validator) Dependencies() []string { return nil }

func (Validator) CanAnalyze(
	_ context.Context,
	req analyzer.ProposalAnalyzeRequest,
	_ analyzer.DecodedTimelockProposal,
) bool {
	proposal := req.TimelockProposal
	if proposal == nil {
		return false
	}

	return proposal.Action == mcmstypes.TimelockActionSchedule
}

func (Validator) Analyze(
	ctx context.Context,
	req analyzer.ProposalAnalyzeRequest,
	_ analyzer.DecodedTimelockProposal,
) (analyzer.Annotations, error) {
	proposal := req.TimelockProposal
	if proposal == nil {
		return nil, nil
	}

	proposalDelay := proposal.Delay
	execCtx := req.ExecutionContext

	chainDelays := make([]chainDelay, 0)
	verifyErrors := make([]string, 0)

	for _, entry := range sortedTimelockEntries(proposal) {
		minDelay, err := readTimelockMinDelay(ctx, execCtx, proposal, entry.selector, entry.address)
		if err != nil {
			verifyErrors = append(
				verifyErrors,
				fmt.Sprintf(
					"%s (%d): %v",
					format.ResolveChainName(entry.selector),
					entry.selector,
					err,
				),
			)

			continue
		}
		chainDelays = append(chainDelays, chainDelay{selector: entry.selector, minDelay: minDelay})
	}

	report := buildReport(proposalDelay, chainDelays, verifyErrors, proposal)

	return reportAnnotations(report), nil
}

func buildReport(
	proposalDelay mcmstypes.Duration,
	chainDelays []chainDelay,
	verifyErrors []string,
	proposal *mcms.TimelockProposal,
) Report {
	report := Report{
		ProposalDelay: proposalDelay.String(),
	}

	if len(verifyErrors) > 0 {
		sort.Strings(verifyErrors)
		report.Validation = "unable to verify on-chain minDelay: " + strings.Join(verifyErrors, "; ")
		report.Severity = analyzer.SeverityWarning

		return report
	}

	if len(chainDelays) == 0 {
		report.Validation = "no timelock addresses in proposal"
		report.Severity = analyzer.SeverityWarning

		return report
	}

	var maxMinDelay mcmstypes.Duration
	sort.Slice(chainDelays, func(i, j int) bool { return chainDelays[i].selector < chainDelays[j].selector })
	report.ChainMinDelays = make([]ChainMinDelay, 0, len(chainDelays))
	for _, entry := range chainDelays {
		report.ChainMinDelays = append(report.ChainMinDelays, ChainMinDelay{
			ChainSelector: entry.selector,
			ChainName:     format.ResolveChainName(entry.selector),
			MinDelay:      entry.minDelay.String(),
			Address:       timelockAddressForChain(proposal, entry.selector),
		})
		if entry.minDelay.Duration > maxMinDelay.Duration {
			maxMinDelay = entry.minDelay
		}
	}

	if maxMinDelay.Duration > 0 && proposalDelay.Duration < maxMinDelay.Duration {
		report.Validation = fmt.Sprintf(
			"proposal delay %s is less than on-chain minDelay %s",
			proposalDelay,
			maxMinDelay,
		)
		report.Severity = analyzer.SeverityError

		return report
	}

	report.Validation = fmt.Sprintf(
		"proposal delay %s satisfies on-chain minDelay %s",
		proposalDelay,
		maxMinDelay,
	)
	report.Severity = analyzer.SeverityInfo

	return report
}

func sortedTimelockEntries(proposal *mcms.TimelockProposal) []timelockChainEntry {
	if proposal == nil || len(proposal.TimelockAddresses) == 0 {
		return nil
	}

	selectors := make([]uint64, 0, len(proposal.TimelockAddresses))
	for selector := range proposal.TimelockAddresses {
		selectors = append(selectors, uint64(selector))
	}
	sort.Slice(selectors, func(i, j int) bool { return selectors[i] < selectors[j] })

	entries := make([]timelockChainEntry, len(selectors))
	for i, selector := range selectors {
		entries[i] = timelockChainEntry{
			selector: selector,
			address:  proposal.TimelockAddresses[mcmstypes.ChainSelector(selector)],
		}
	}

	return entries
}

func timelockAddressForChain(proposal *mcms.TimelockProposal, chainSelector uint64) string {
	if proposal == nil {
		return ""
	}

	return proposal.TimelockAddresses[mcmstypes.ChainSelector(chainSelector)]
}

func readTimelockMinDelay(
	ctx context.Context,
	execCtx analyzer.ExecutionContext,
	proposal *mcms.TimelockProposal,
	chainSelector uint64,
	timelockAddress string,
) (mcmstypes.Duration, error) {
	chainAccessor := cldfmcmsadapters.Wrap(execCtx.BlockChains())
	metadata := chainMetadataForInspector(proposal, chainSelector)

	inspector, err := mcmschainwrappers.BuildTimelockInspector(
		&chainAccessor,
		mcmstypes.ChainSelector(chainSelector),
		metadata,
	)
	if err != nil {
		return mcmstypes.Duration{}, err
	}

	minDelaySec, err := inspector.GetMinDelay(ctx, timelockAddress)
	if err != nil {
		return mcmstypes.Duration{}, err
	}

	return durationFromSeconds(minDelaySec)
}

func chainMetadataForInspector(proposal *mcms.TimelockProposal, chainSelector uint64) mcmstypes.ChainMetadata {
	if proposal == nil || len(proposal.ChainMetadata) == 0 {
		return mcmstypes.ChainMetadata{}
	}

	metadata, ok := proposal.ChainMetadata[mcmstypes.ChainSelector(chainSelector)]
	if !ok {
		return mcmstypes.ChainMetadata{}
	}

	return metadata
}

func durationFromSeconds(seconds uint64) (mcmstypes.Duration, error) {
	maxSeconds := uint64(math.MaxInt64 / int64(time.Second))
	if seconds > maxSeconds {
		return mcmstypes.Duration{}, fmt.Errorf("min delay %d seconds exceeds representable duration", seconds)
	}

	return mcmstypes.NewDuration(time.Duration(seconds) * time.Second), nil
}

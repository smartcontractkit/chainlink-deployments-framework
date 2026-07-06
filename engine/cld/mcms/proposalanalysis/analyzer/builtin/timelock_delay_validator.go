package builtin

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	mcmstypes "github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/format"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalutils"
)

const (
	TimelockDelayValidatorID = "cld.proposal.timelock_delay"

	annotProposalTimelockDelay = "proposal_timelock_delay"
	annotTimelockMinDelay      = "timelock_min_delay"
	annotTimelockDelayCheck    = "timelock_delay_validation"
)

type chainDelay struct {
	selector uint64
	minDelay mcmstypes.Duration
}

// TimelockDelayValidator checks that schedule proposals use a delay >= each
// executing timelock's on-chain minDelay.
type TimelockDelayValidator struct{}

var _ analyzer.ProposalAnalyzer = TimelockDelayValidator{}

func (TimelockDelayValidator) ID() string             { return TimelockDelayValidatorID }
func (TimelockDelayValidator) Dependencies() []string { return nil }

func (TimelockDelayValidator) CanAnalyze(
	_ context.Context,
	req analyzer.ProposalAnalyzeRequest,
	_ analyzer.DecodedTimelockProposal,
) bool {
	return req.ExecutionContext.ProposalAction() == mcmstypes.TimelockActionSchedule
}

func (TimelockDelayValidator) Analyze(
	ctx context.Context,
	req analyzer.ProposalAnalyzeRequest,
	_ analyzer.DecodedTimelockProposal,
) (analyzer.Annotations, error) {
	execCtx := req.ExecutionContext
	proposalDelay := execCtx.ProposalDelay()

	anns := make(analyzer.Annotations, 0, 4)
	anns = append(anns, analyzer.NewAnnotation(annotProposalTimelockDelay, "string", proposalDelay.String()))

	chainDelays := make([]chainDelay, 0)
	verifyErrors := make([]string, 0)

	for chainSelector, timelockAddress := range sortedTimelockAddresses(execCtx) {
		minDelay, err := readTimelockMinDelay(ctx, execCtx, chainSelector, timelockAddress)
		if err != nil {
			verifyErrors = append(
				verifyErrors,
				fmt.Sprintf(
					"%s (%d): %v",
					format.ResolveChainName(chainSelector),
					chainSelector,
					err,
				),
			)

			continue
		}
		chainDelays = append(chainDelays, chainDelay{selector: chainSelector, minDelay: minDelay})
	}

	anns = append(anns, appendTimelockDelayValidation(proposalDelay, chainDelays, verifyErrors, execCtx)...)

	return anns, nil
}

func appendTimelockDelayValidation(
	proposalDelay mcmstypes.Duration,
	chainDelays []chainDelay,
	verifyErrors []string,
	execCtx analyzer.ExecutionContext,
) analyzer.Annotations {
	var anns analyzer.Annotations

	if len(verifyErrors) > 0 {
		return append(anns,
			analyzer.NewAnnotation(
				annotTimelockDelayCheck,
				"string",
				"unable to verify on-chain minDelay: "+strings.Join(verifyErrors, "; "),
			),
			analyzer.SeverityAnnotation(analyzer.SeverityWarning),
		)
	}

	if len(chainDelays) == 0 {
		return append(anns,
			analyzer.NewAnnotation(annotTimelockDelayCheck, "string", "no timelock addresses in proposal"),
			analyzer.SeverityAnnotation(analyzer.SeverityWarning),
		)
	}

	minDelayLines := make([]string, 0, len(chainDelays))
	var maxMinDelay mcmstypes.Duration
	for _, entry := range chainDelays {
		minDelayLines = append(
			minDelayLines,
			fmt.Sprintf(
				"%s: %s (`%s`)",
				format.ResolveChainName(entry.selector),
				entry.minDelay,
				timelockAddressForChain(execCtx, entry.selector),
			),
		)
		if entry.minDelay.Duration > maxMinDelay.Duration {
			maxMinDelay = entry.minDelay
		}
	}
	anns = append(anns, analyzer.NewAnnotation(annotTimelockMinDelay, "string", strings.Join(minDelayLines, "; ")))

	if maxMinDelay.Duration > 0 && proposalDelay.Duration < maxMinDelay.Duration {
		return append(anns,
			analyzer.NewAnnotation(
				annotTimelockDelayCheck,
				"string",
				fmt.Sprintf(
					"proposal delay %s is less than on-chain minDelay %s",
					proposalDelay,
					maxMinDelay,
				),
			),
			analyzer.SeverityAnnotation(analyzer.SeverityError),
		)
	}

	return append(anns,
		analyzer.NewAnnotation(
			annotTimelockDelayCheck,
			"string",
			fmt.Sprintf("proposal delay %s satisfies on-chain minDelay %s", proposalDelay, maxMinDelay),
		),
		analyzer.SeverityAnnotation(analyzer.SeverityInfo),
	)
}

func sortedTimelockAddresses(execCtx analyzer.ExecutionContext) map[uint64]string {
	addrs := execCtx.TimelockAddresses()
	if len(addrs) == 0 {
		return nil
	}

	selectors := make([]uint64, 0, len(addrs))
	for selector := range addrs {
		selectors = append(selectors, selector)
	}
	sort.Slice(selectors, func(i, j int) bool { return selectors[i] < selectors[j] })

	out := make(map[uint64]string, len(addrs))
	for _, selector := range selectors {
		out[selector] = addrs[selector]
	}

	return out
}

func timelockAddressForChain(execCtx analyzer.ExecutionContext, chainSelector uint64) string {
	addr, _ := execCtx.TimelockAddress(chainSelector)

	return addr
}

func readTimelockMinDelay(
	ctx context.Context,
	execCtx analyzer.ExecutionContext,
	chainSelector uint64,
	timelockAddress string,
) (mcmstypes.Duration, error) {
	inspector, err := proposalutils.McmsTimelockInspectorForChain(
		execCtx.BlockChains(),
		chainSelector,
		mcmstypes.ChainMetadata{},
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

func durationFromSeconds(seconds uint64) (mcmstypes.Duration, error) {
	maxSeconds := uint64(math.MaxInt64 / int64(time.Second))
	if seconds > maxSeconds {
		return mcmstypes.Duration{}, fmt.Errorf("min delay %d seconds exceeds representable duration", seconds)
	}

	return mcmstypes.NewDuration(time.Duration(seconds) * time.Second), nil
}

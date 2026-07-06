package builtin

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	chainsel "github.com/smartcontractkit/chain-selectors"
	mcmsevmsdk "github.com/smartcontractkit/mcms/sdk/evm"
	mcmssolana "github.com/smartcontractkit/mcms/sdk/solana"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	cldfevm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	cldfsol "github.com/smartcontractkit/chainlink-deployments-framework/chain/solana"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/format"
)

const (
	TimelockDelayValidatorID = "cld.proposal.timelock_delay"

	annotProposalTimelockDelay = "proposal_timelock_delay"
	annotTimelockMinDelay      = "timelock_min_delay"
	annotTimelockDelayCheck    = "timelock_delay_validation"
)

// TimelockDelayValidator checks that schedule proposals use a delay >= each
// executing timelock's on-chain minDelay. A proposal delay of 0 does not mean
// immediate execution when minDelay is non-zero.
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

	anns := analyzer.Annotations{
		analyzer.NewAnnotation(annotProposalTimelockDelay, "string", proposalDelay.String()),
	}

	type chainDelay struct {
		selector uint64
		minDelay mcmstypes.Duration
	}

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

	if len(verifyErrors) > 0 {
		anns = append(anns,
			analyzer.NewAnnotation(
				annotTimelockDelayCheck,
				"string",
				"unable to verify on-chain minDelay: "+strings.Join(verifyErrors, "; "),
			),
			analyzer.SeverityAnnotation(analyzer.SeverityWarning),
		)

		return anns, nil
	}

	if len(chainDelays) == 0 {
		anns = append(anns,
			analyzer.NewAnnotation(annotTimelockDelayCheck, "string", "no timelock addresses in proposal"),
			analyzer.SeverityAnnotation(analyzer.SeverityWarning),
		)

		return anns, nil
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
		anns = append(anns,
			analyzer.NewAnnotation(
				annotTimelockDelayCheck,
				"string",
				fmt.Sprintf(
					"proposal delay %s is less than on-chain minDelay %s; delay 0 does not mean immediate execution",
					proposalDelay,
					maxMinDelay,
				),
			),
			analyzer.SeverityAnnotation(analyzer.SeverityError),
		)

		return anns, nil
	}

	anns = append(anns,
		analyzer.NewAnnotation(
			annotTimelockDelayCheck,
			"string",
			fmt.Sprintf("proposal delay %s satisfies on-chain minDelay %s", proposalDelay, maxMinDelay),
		),
		analyzer.SeverityAnnotation(analyzer.SeverityInfo),
	)

	return anns, nil
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
	family, err := chainsel.GetSelectorFamily(chainSelector)
	if err != nil {
		return mcmstypes.Duration{}, fmt.Errorf("chain family: %w", err)
	}

	switch family {
	case chainsel.FamilyEVM:
		chain, ok := execCtx.BlockChains().EVMChains()[chainSelector]
		if !ok {
			return mcmstypes.Duration{}, fmt.Errorf("evm chain %d not loaded in environment", chainSelector)
		}

		return readEVMTimelockMinDelay(ctx, chain, timelockAddress)
	case chainsel.FamilySolana:
		chain, ok := execCtx.BlockChains().SolanaChains()[chainSelector]
		if !ok {
			return mcmstypes.Duration{}, fmt.Errorf("solana chain %d not loaded in environment", chainSelector)
		}

		return readSolanaTimelockMinDelay(ctx, chain, timelockAddress)
	default:
		return mcmstypes.Duration{}, fmt.Errorf("unsupported chain family %q", family)
	}
}

func readEVMTimelockMinDelay(
	ctx context.Context,
	chain cldfevm.Chain,
	timelockAddress string,
) (mcmstypes.Duration, error) {
	inspector := mcmsevmsdk.NewTimelockInspector(chain.Client)
	minDelaySec, err := inspector.GetMinDelay(ctx, timelockAddress)
	if err != nil {
		return mcmstypes.Duration{}, err
	}

	return durationFromSeconds(minDelaySec)
}

func readSolanaTimelockMinDelay(
	ctx context.Context,
	chain cldfsol.Chain,
	timelockAddress string,
) (mcmstypes.Duration, error) {
	inspector := mcmssolana.NewTimelockInspector(chain.Client)
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

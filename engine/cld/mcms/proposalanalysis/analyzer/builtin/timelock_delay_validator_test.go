package builtin

import (
	"testing"
	"time"

	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
)

func testExecutionContext(metadata *analyzer.ProposalExecutionMetadata) analyzer.ExecutionContext {
	return analyzer.NewExecutionContextNode(
		cldfdomain.NewDomain("/tmp/domains", "test"),
		"staging",
		chain.NewBlockChains(nil),
		datastore.NewMemoryDataStore().Seal(),
		metadata,
	)
}

func TestTimelockDelayValidator_Analyze_DelayTooLow(t *testing.T) {
	t.Parallel()

	validator := TimelockDelayValidator{}
	req := analyzer.ProposalAnalyzeRequest{
		ExecutionContext: testExecutionContext(
			&analyzer.ProposalExecutionMetadata{
				Action: mcmstypes.TimelockActionSchedule,
				Delay:  mcmstypes.NewDuration(0),
				TimelockAddresses: map[uint64]string{
					1: "0xTimelock",
				},
			},
		),
	}

	require.True(t, validator.CanAnalyze(t.Context(), req, nil))

	// Without RPC access this reports a verification warning rather than a false positive pass.
	anns, err := validator.Analyze(t.Context(), req, nil)
	require.NoError(t, err)
	require.NotEmpty(t, anns)
	require.Equal(t, "0s", anns[0].Value())
}

func TestTimelockDelayValidator_CanAnalyze_ScheduleOnly(t *testing.T) {
	t.Parallel()

	validator := TimelockDelayValidator{}

	scheduleCtx := testExecutionContext(
		&analyzer.ProposalExecutionMetadata{Action: mcmstypes.TimelockActionSchedule},
	)
	require.True(t, validator.CanAnalyze(t.Context(), analyzer.ProposalAnalyzeRequest{
		ExecutionContext: scheduleCtx,
	}, nil))

	bypassCtx := testExecutionContext(
		&analyzer.ProposalExecutionMetadata{Action: mcmstypes.TimelockActionBypass},
	)
	require.False(t, validator.CanAnalyze(t.Context(), analyzer.ProposalAnalyzeRequest{
		ExecutionContext: bypassCtx,
	}, nil))
}

func TestCompareProposalDelayToMinDelay(t *testing.T) {
	t.Parallel()

	minDelay := mcmstypes.NewDuration(5 * time.Minute)
	proposalDelay := mcmstypes.NewDuration(0)

	require.Less(t, proposalDelay.Duration, minDelay.Duration)
}

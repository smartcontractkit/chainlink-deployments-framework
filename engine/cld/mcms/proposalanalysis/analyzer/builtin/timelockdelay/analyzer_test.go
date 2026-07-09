package timelockdelay

import (
	"math"
	"testing"
	"time"

	"github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/format"
)

func testExecutionContext() analyzer.ExecutionContext {
	return analyzer.NewExecutionContextNode(
		cldfdomain.NewDomain("/tmp/domains", "test"),
		"staging",
		chain.NewBlockChains(nil),
		datastore.NewMemoryDataStore().Seal(),
	)
}

func testAnalyzeRequest(proposal *mcms.TimelockProposal) analyzer.ProposalAnalyzeRequest {
	return analyzer.ProposalAnalyzeRequest{
		AnalyzeEnvelope: analyzer.AnalyzeEnvelope{
			ExecutionContext: testExecutionContext(),
			TimelockProposal: proposal,
		},
	}
}

func reportFromAnnotations(anns analyzer.Annotations) (Report, bool) {
	for _, ann := range anns {
		if ann.Name() == ReportName {
			report, ok := ann.Value().(Report)

			return report, ok
		}
	}

	return Report{}, false
}

func TestValidator_Analyze_VerificationWarningWhenChainMissing(t *testing.T) {
	t.Parallel()

	validator := Validator{}
	req := testAnalyzeRequest(&mcms.TimelockProposal{
		Action: mcmstypes.TimelockActionSchedule,
		Delay:  mcmstypes.NewDuration(0),
		TimelockAddresses: map[mcmstypes.ChainSelector]string{
			1: "0xTimelock",
		},
	})

	require.True(t, validator.CanAnalyze(t.Context(), req, nil))

	anns, err := validator.Analyze(t.Context(), req, nil)
	require.NoError(t, err)
	require.NotEmpty(t, anns)

	report, ok := reportFromAnnotations(anns)
	require.True(t, ok)
	assert.Contains(t, report.Validation, "unable to verify on-chain minDelay")
	assert.Empty(t, report.ChainMinDelays)
	assert.Equal(t, analyzer.SeverityWarning, report.Severity)
}

func TestValidator_Analyze_NoTimelockAddresses(t *testing.T) {
	t.Parallel()

	validator := Validator{}
	anns, err := validator.Analyze(t.Context(), testAnalyzeRequest(&mcms.TimelockProposal{
		Action: mcmstypes.TimelockActionSchedule,
		Delay:  mcmstypes.NewDuration(5 * time.Minute),
	}), nil)
	require.NoError(t, err)

	report, ok := reportFromAnnotations(anns)
	require.True(t, ok)
	assert.Equal(t, "no timelock addresses in proposal", report.Validation)
	assert.Equal(t, "5m0s", report.ProposalDelay)
	assert.Equal(t, analyzer.SeverityWarning, report.Severity)
}

func TestValidator_CanAnalyze_ScheduleOnly(t *testing.T) {
	t.Parallel()

	validator := Validator{}

	require.True(t, validator.CanAnalyze(t.Context(), testAnalyzeRequest(&mcms.TimelockProposal{
		Action: mcmstypes.TimelockActionSchedule,
	}), nil))

	require.False(t, validator.CanAnalyze(t.Context(), testAnalyzeRequest(&mcms.TimelockProposal{
		Action: mcmstypes.TimelockActionBypass,
	}), nil))

	require.False(t, validator.CanAnalyze(t.Context(), analyzer.ProposalAnalyzeRequest{
		AnalyzeEnvelope: analyzer.AnalyzeEnvelope{
			ExecutionContext: testExecutionContext(),
		},
	}, nil))
}

func TestBuildReport(t *testing.T) {
	t.Parallel()

	const chainSelector = uint64(16098325658947243212)
	proposal := &mcms.TimelockProposal{
		TimelockAddresses: map[mcmstypes.ChainSelector]string{
			mcmstypes.ChainSelector(chainSelector): "0x9Af873f951c444d37B27B440ae53AB63CE58E5e5",
		},
	}

	tests := []struct {
		name           string
		proposalDelay  mcmstypes.Duration
		chainDelays    []chainDelay
		verifyErrors   []string
		wantCheck      string
		wantSeverity   analyzer.Severity
		wantMinDelayOK bool
	}{
		{
			name:          "verify errors produce warning",
			proposalDelay: mcmstypes.NewDuration(0),
			verifyErrors:  []string{"pharos (16098325658947243212): missing EVM chain client"},
			wantCheck:     "unable to verify on-chain minDelay: pharos (16098325658947243212): missing EVM chain client",
			wantSeverity:  analyzer.SeverityWarning,
		},
		{
			name:          "no timelock addresses",
			proposalDelay: mcmstypes.NewDuration(time.Minute),
			wantCheck:     "no timelock addresses in proposal",
			wantSeverity:  analyzer.SeverityWarning,
		},
		{
			name:          "proposal delay below minDelay",
			proposalDelay: mcmstypes.NewDuration(0),
			chainDelays: []chainDelay{
				{selector: chainSelector, minDelay: mcmstypes.NewDuration(5 * time.Minute)},
			},
			wantCheck:      "proposal delay 0s is less than on-chain minDelay 5m0s",
			wantSeverity:   analyzer.SeverityError,
			wantMinDelayOK: true,
		},
		{
			name:          "proposal delay satisfies minDelay",
			proposalDelay: mcmstypes.NewDuration(10 * time.Minute),
			chainDelays: []chainDelay{
				{selector: chainSelector, minDelay: mcmstypes.NewDuration(5 * time.Minute)},
			},
			wantCheck:      "proposal delay 10m0s satisfies on-chain minDelay 5m0s",
			wantSeverity:   analyzer.SeverityInfo,
			wantMinDelayOK: true,
		},
		{
			name:          "zero minDelay with zero proposal delay passes",
			proposalDelay: mcmstypes.NewDuration(0),
			chainDelays: []chainDelay{
				{selector: chainSelector, minDelay: mcmstypes.NewDuration(0)},
			},
			wantCheck:      "proposal delay 0s satisfies on-chain minDelay 0s",
			wantSeverity:   analyzer.SeverityInfo,
			wantMinDelayOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			report := buildReport(tt.proposalDelay, tt.chainDelays, tt.verifyErrors, proposal)

			assert.Equal(t, tt.proposalDelay.String(), report.ProposalDelay)
			assert.Equal(t, tt.wantCheck, report.Validation)
			assert.Equal(t, tt.wantSeverity, report.Severity)
			assert.Equal(t, tt.wantMinDelayOK, len(report.ChainMinDelays) > 0)
		})
	}
}

func TestSortedTimelockEntries(t *testing.T) {
	t.Parallel()

	got := sortedTimelockEntries(&mcms.TimelockProposal{
		TimelockAddresses: map[mcmstypes.ChainSelector]string{
			3: "0x3",
			1: "0x1",
			2: "0x2",
		},
	})
	require.Len(t, got, 3)
	assert.Equal(t, []uint64{1, 2, 3}, []uint64{got[0].selector, got[1].selector, got[2].selector})
	assert.Equal(t, "0x1", got[0].address)
	assert.Equal(t, "0x2", got[1].address)
	assert.Equal(t, "0x3", got[2].address)

	assert.Nil(t, sortedTimelockEntries(nil))
}

func TestBuildReport_SortsUnorderedChainDelays(t *testing.T) {
	t.Parallel()

	const chainA = uint64(3)
	const chainB = uint64(1)
	proposal := &mcms.TimelockProposal{
		TimelockAddresses: map[mcmstypes.ChainSelector]string{
			mcmstypes.ChainSelector(chainA): "0xA",
			mcmstypes.ChainSelector(chainB): "0xB",
		},
	}

	report := buildReport(
		mcmstypes.NewDuration(time.Minute),
		[]chainDelay{
			{selector: chainA, minDelay: mcmstypes.NewDuration(30 * time.Second)},
			{selector: chainB, minDelay: mcmstypes.NewDuration(45 * time.Second)},
		},
		nil,
		proposal,
	)

	require.Len(t, report.ChainMinDelays, 2)
	assert.Equal(t, chainB, report.ChainMinDelays[0].ChainSelector)
	assert.Equal(t, format.ResolveChainName(chainB), report.ChainMinDelays[0].ChainName)
}

func TestDurationFromSeconds(t *testing.T) {
	t.Parallel()

	got, err := durationFromSeconds(300)
	require.NoError(t, err)
	assert.Equal(t, mcmstypes.NewDuration(5*time.Minute), got)

	_, err = durationFromSeconds(uint64(math.MaxInt64/time.Second) + 1)
	require.ErrorContains(t, err, "exceeds representable duration")
}

func TestReportAnnotations(t *testing.T) {
	t.Parallel()

	anns := reportAnnotations(Report{
		ProposalDelay: "1h0m0s",
		Validation:    "ok",
		Severity:      analyzer.SeverityInfo,
	})

	require.Len(t, anns, 1)
	assert.Equal(t, ReportName, anns[0].Name())
}

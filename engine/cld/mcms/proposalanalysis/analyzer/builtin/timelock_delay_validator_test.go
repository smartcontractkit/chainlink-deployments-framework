package builtin

import (
	"math"
	"testing"
	"time"

	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/assert"
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

func annotationValue(anns analyzer.Annotations, name string) (string, bool) {
	for _, ann := range anns {
		if ann.Name() == name {
			value, ok := ann.Value().(string)

			return value, ok
		}
	}

	return "", false
}

func annotationSeverity(anns analyzer.Annotations) (analyzer.Severity, bool) {
	for _, ann := range anns {
		if ann.Name() == analyzer.AnnotationSeverityName {
			level, ok := ann.Value().(string)
			if !ok {
				return "", false
			}

			return analyzer.Severity(level), true
		}
	}

	return "", false
}

func TestTimelockDelayValidator_Analyze_VerificationWarningWhenChainMissing(t *testing.T) {
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

	anns, err := validator.Analyze(t.Context(), req, nil)
	require.NoError(t, err)
	require.NotEmpty(t, anns)

	delay, ok := annotationValue(anns, annotProposalTimelockDelay)
	require.True(t, ok)
	assert.Equal(t, "0s", delay)

	check, ok := annotationValue(anns, annotTimelockDelayCheck)
	require.True(t, ok)
	assert.Contains(t, check, "unable to verify on-chain minDelay")

	severity, ok := annotationSeverity(anns)
	require.True(t, ok)
	assert.Equal(t, analyzer.SeverityWarning, severity)
}

func TestTimelockDelayValidator_Analyze_NoTimelockAddresses(t *testing.T) {
	t.Parallel()

	validator := TimelockDelayValidator{}
	anns, err := validator.Analyze(t.Context(), analyzer.ProposalAnalyzeRequest{
		ExecutionContext: testExecutionContext(
			&analyzer.ProposalExecutionMetadata{
				Action: mcmstypes.TimelockActionSchedule,
				Delay:  mcmstypes.NewDuration(5 * time.Minute),
			},
		),
	}, nil)
	require.NoError(t, err)

	check, ok := annotationValue(anns, annotTimelockDelayCheck)
	require.True(t, ok)
	assert.Equal(t, "no timelock addresses in proposal", check)

	severity, ok := annotationSeverity(anns)
	require.True(t, ok)
	assert.Equal(t, analyzer.SeverityWarning, severity)
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

func TestAppendTimelockDelayValidation(t *testing.T) {
	t.Parallel()

	const chainSelector = uint64(16098325658947243212)
	execCtx := testExecutionContext(
		&analyzer.ProposalExecutionMetadata{
			TimelockAddresses: map[uint64]string{
				chainSelector: "0x9Af873f951c444d37B27B440ae53AB63CE58E5e5",
			},
		},
	)

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

			anns := appendTimelockDelayValidation(tt.proposalDelay, tt.chainDelays, tt.verifyErrors, execCtx)

			check, ok := annotationValue(anns, annotTimelockDelayCheck)
			require.True(t, ok)
			assert.Equal(t, tt.wantCheck, check)

			severity, ok := annotationSeverity(anns)
			require.True(t, ok)
			assert.Equal(t, tt.wantSeverity, severity)

			_, ok = annotationValue(anns, annotTimelockMinDelay)
			assert.Equal(t, tt.wantMinDelayOK, ok)
		})
	}
}

func TestSortedTimelockAddresses(t *testing.T) {
	t.Parallel()

	execCtx := testExecutionContext(
		&analyzer.ProposalExecutionMetadata{
			TimelockAddresses: map[uint64]string{
				3: "0x3",
				1: "0x1",
				2: "0x2",
			},
		},
	)

	got := sortedTimelockAddresses(execCtx)
	require.Len(t, got, 3)
	assert.Equal(t, "0x1", got[1])
	assert.Equal(t, "0x2", got[2])
	assert.Equal(t, "0x3", got[3])

	assert.Nil(t, sortedTimelockAddresses(testExecutionContext(nil)))
}

func TestDurationFromSeconds(t *testing.T) {
	t.Parallel()

	got, err := durationFromSeconds(300)
	require.NoError(t, err)
	assert.Equal(t, mcmstypes.NewDuration(5*time.Minute), got)

	_, err = durationFromSeconds(uint64(math.MaxInt64/time.Second) + 1)
	require.ErrorContains(t, err, "exceeds representable duration")
}

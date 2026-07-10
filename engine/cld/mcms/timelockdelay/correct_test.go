package timelockdelay

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestMaxMinDelay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []ChainDelay
		want  mcmstypes.Duration
	}{
		{
			name: "nil input",
		},
		{
			name: "returns largest delay",
			input: []ChainDelay{
				{Selector: 1, MinDelay: mcmstypes.NewDuration(30 * time.Minute)},
				{Selector: 2, MinDelay: mcmstypes.NewDuration(time.Hour)},
			},
			want: mcmstypes.NewDuration(time.Hour),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, MaxMinDelay(tt.input))
		})
	}
}

func TestDurationFromSeconds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		seconds uint64
		want    mcmstypes.Duration
		wantErr string
	}{
		{
			name:    "converts seconds",
			seconds: 3600,
			want:    mcmstypes.NewDuration(time.Hour),
		},
		{
			name:    "rejects overflow",
			seconds: ^uint64(0),
			wantErr: "exceeds representable duration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := durationFromSeconds(tt.seconds)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestCorrectTimelockDelaysWithLookup(t *testing.T) {
	t.Parallel()

	minDelay := mcmstypes.NewDuration(3 * time.Hour)
	singleChainLookup := func(
		_ context.Context,
		_ *mcms.TimelockProposal,
		chainSelector uint64,
		_ string,
	) (mcmstypes.Duration, error) {
		if chainSelector != 1 {
			return mcmstypes.Duration{}, errors.New("unexpected chain")
		}

		return minDelay, nil
	}
	multiChainLookup := func(
		_ context.Context,
		_ *mcms.TimelockProposal,
		chainSelector uint64,
		_ string,
	) (mcmstypes.Duration, error) {
		switch chainSelector {
		case 1:
			return mcmstypes.NewDuration(time.Hour), nil
		case 2:
			return minDelay, nil
		default:
			return mcmstypes.Duration{}, errors.New("unexpected chain")
		}
	}
	failingLookup := func(
		_ context.Context,
		_ *mcms.TimelockProposal,
		_ uint64,
		_ string,
	) (mcmstypes.Duration, error) {
		return mcmstypes.Duration{}, errors.New("rpc unavailable")
	}
	partialLookup := func(
		_ context.Context,
		_ *mcms.TimelockProposal,
		chainSelector uint64,
		_ string,
	) (mcmstypes.Duration, error) {
		if chainSelector == 1 {
			return mcmstypes.NewDuration(time.Hour), nil
		}

		return mcmstypes.Duration{}, errors.New("rpc unavailable")
	}
	zeroLookup := func(
		_ context.Context,
		_ *mcms.TimelockProposal,
		_ uint64,
		_ string,
	) (mcmstypes.Duration, error) {
		return mcmstypes.NewDuration(0), nil
	}

	tests := []struct {
		name      string
		proposal  func() mcms.TimelockProposal
		lookup    MinDelayLookup
		contextFn func(t *testing.T) context.Context
		wantErr   error
		wantDelay mcmstypes.Duration
	}{
		{
			name:      "fills unset delay",
			proposal:  func() mcms.TimelockProposal { return scheduleProposal(mcmstypes.NewDuration(0)) },
			lookup:    singleChainLookup,
			wantDelay: minDelay,
		},
		{
			name:      "uses max minDelay across chains",
			proposal:  func() mcms.TimelockProposal { return scheduleMultiChainProposal(mcmstypes.NewDuration(0)) },
			lookup:    multiChainLookup,
			wantDelay: minDelay,
		},
		{
			name:      "bumps delay below minDelay",
			proposal:  func() mcms.TimelockProposal { return scheduleProposal(mcmstypes.NewDuration(time.Hour)) },
			lookup:    singleChainLookup,
			wantDelay: minDelay,
		},
		{
			name:      "leaves sufficient delay unchanged",
			proposal:  func() mcms.TimelockProposal { return scheduleProposal(mcmstypes.NewDuration(4 * time.Hour)) },
			lookup:    singleChainLookup,
			wantDelay: mcmstypes.NewDuration(4 * time.Hour),
		},
		{
			name: "skips bypass proposals",
			proposal: func() mcms.TimelockProposal {
				proposal := scheduleProposal(mcmstypes.NewDuration(0))
				proposal.Action = mcmstypes.TimelockActionBypass

				return proposal
			},
			lookup:    singleChainLookup,
			wantDelay: mcmstypes.NewDuration(0),
		},
		{
			name:      "rpc failure with unset delay returns error",
			proposal:  func() mcms.TimelockProposal { return scheduleProposal(mcmstypes.NewDuration(0)) },
			lookup:    failingLookup,
			wantErr:   ErrUnsetTimelockDelayUnverified,
			wantDelay: mcmstypes.NewDuration(0),
		},
		{
			name:      "rpc failure with explicit delay leaves proposal unchanged",
			proposal:  func() mcms.TimelockProposal { return scheduleProposal(mcmstypes.NewDuration(time.Hour)) },
			lookup:    failingLookup,
			wantDelay: mcmstypes.NewDuration(time.Hour),
		},
		{
			name:      "partial rpc failure with unset delay returns error",
			proposal:  func() mcms.TimelockProposal { return scheduleMultiChainProposal(mcmstypes.NewDuration(0)) },
			lookup:    partialLookup,
			wantErr:   ErrUnsetTimelockDelayUnverified,
			wantDelay: mcmstypes.NewDuration(0),
		},
		{
			name:      "zero on-chain minDelay with unset delay returns error",
			proposal:  func() mcms.TimelockProposal { return scheduleProposal(mcmstypes.NewDuration(0)) },
			lookup:    zeroLookup,
			wantErr:   ErrUnsetTimelockDelayUnverified,
			wantDelay: mcmstypes.NewDuration(0),
		},
		{
			name:      "context cancellation with explicit delay returns error",
			proposal:  func() mcms.TimelockProposal { return scheduleProposal(mcmstypes.NewDuration(time.Hour)) },
			lookup:    singleChainLookup,
			contextFn: cancelledTestContext,
			wantErr:   context.Canceled,
			wantDelay: mcmstypes.NewDuration(time.Hour),
		},
		{
			name:      "context cancellation with unset delay returns error",
			proposal:  func() mcms.TimelockProposal { return scheduleProposal(mcmstypes.NewDuration(0)) },
			lookup:    singleChainLookup,
			contextFn: cancelledTestContext,
			wantErr:   context.Canceled,
			wantDelay: mcmstypes.NewDuration(0),
		},
		{
			name: "no timelock addresses with unset delay returns error",
			proposal: func() mcms.TimelockProposal {
				proposal := scheduleProposal(mcmstypes.NewDuration(0))
				proposal.TimelockAddresses = nil

				return proposal
			},
			lookup:  singleChainLookup,
			wantErr: ErrUnsetTimelockDelayUnverified,
		},
		{
			name: "no timelock addresses with explicit delay leaves proposal unchanged",
			proposal: func() mcms.TimelockProposal {
				proposal := scheduleProposal(mcmstypes.NewDuration(time.Hour))
				proposal.TimelockAddresses = nil

				return proposal
			},
			lookup:    singleChainLookup,
			wantDelay: mcmstypes.NewDuration(time.Hour),
		},
		{
			name:      "zero on-chain minDelay with explicit delay leaves proposal unchanged",
			proposal:  func() mcms.TimelockProposal { return scheduleProposal(mcmstypes.NewDuration(time.Hour)) },
			lookup:    zeroLookup,
			wantDelay: mcmstypes.NewDuration(time.Hour),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := testContext(t)
			if tt.contextFn != nil {
				ctx = tt.contextFn(t)
			}

			proposals := []mcms.TimelockProposal{tt.proposal()}
			err := CorrectTimelockDelaysWithLookup(ctx, logger.Test(t), proposals, tt.lookup)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.wantDelay, proposals[0].Delay)
		})
	}
}

func TestCorrectTimelockDelays(t *testing.T) {
	t.Parallel()

	proposal := scheduleProposal(mcmstypes.NewDuration(0))
	proposals := []mcms.TimelockProposal{proposal}

	err := CorrectTimelockDelays(t.Context(), logger.Test(t), chain.NewBlockChains(nil), proposals)
	require.ErrorIs(t, err, ErrUnsetTimelockDelayUnverified)
	require.Equal(t, mcmstypes.NewDuration(0), proposals[0].Delay)
}

func scheduleProposal(delay mcmstypes.Duration) mcms.TimelockProposal {
	return mcms.TimelockProposal{
		BaseProposal: mcms.BaseProposal{
			ChainMetadata: map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata{
				1: {MCMAddress: "0xMCMS"},
			},
		},
		Action: mcmstypes.TimelockActionSchedule,
		Delay:  delay,
		TimelockAddresses: map[mcmstypes.ChainSelector]string{
			1: "0xTimelock",
		},
		Operations: []mcmstypes.BatchOperation{{
			ChainSelector: 1,
			Transactions:  []mcmstypes.Transaction{{To: "0xTo"}},
		}},
	}
}

func scheduleMultiChainProposal(delay mcmstypes.Duration) mcms.TimelockProposal {
	return mcms.TimelockProposal{
		BaseProposal: mcms.BaseProposal{
			ChainMetadata: map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata{
				1: {MCMAddress: "0xMCMS1"},
				2: {MCMAddress: "0xMCMS2"},
			},
		},
		Action: mcmstypes.TimelockActionSchedule,
		Delay:  delay,
		TimelockAddresses: map[mcmstypes.ChainSelector]string{
			1: "0xTimelock1",
			2: "0xTimelock2",
		},
		Operations: []mcmstypes.BatchOperation{
			{
				ChainSelector: 1,
				Transactions:  []mcmstypes.Transaction{{To: "0xTo1"}},
			},
			{
				ChainSelector: 2,
				Transactions:  []mcmstypes.Transaction{{To: "0xTo2"}},
			},
		},
	}
}

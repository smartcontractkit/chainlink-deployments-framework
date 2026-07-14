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
)

func TestReadChainMinDelays(t *testing.T) {
	t.Parallel()

	minDelay := mcmstypes.NewDuration(2 * time.Hour)
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

	tests := []struct {
		name            string
		contextFn       func(t *testing.T) context.Context
		read            func(ctx context.Context, proposal *mcms.TimelockProposal) ([]ChainDelay, []string)
		wantDelays      []ChainDelay
		wantVerifyCount int
		verifyContains  string
	}{
		{
			name:      "returns verify errors when chain is unavailable",
			contextFn: testContext,
			read: func(ctx context.Context, proposal *mcms.TimelockProposal) ([]ChainDelay, []string) {
				return ReadChainMinDelays(ctx, chain.NewBlockChains(nil), proposal)
			},
			wantVerifyCount: 1,
			verifyContains:  "chain 1",
		},
		{
			name:      "reads delays via lookup",
			contextFn: testContext,
			read: func(ctx context.Context, proposal *mcms.TimelockProposal) ([]ChainDelay, []string) {
				return readChainMinDelaysWithLookup(ctx, singleChainLookup, proposal)
			},
			wantDelays: []ChainDelay{{Selector: 1, MinDelay: minDelay}},
		},
		{
			name:      "returns context error when context is cancelled",
			contextFn: cancelledTestContext,
			read: func(ctx context.Context, proposal *mcms.TimelockProposal) ([]ChainDelay, []string) {
				return readChainMinDelaysWithLookup(ctx, singleChainLookup, proposal)
			},
			wantVerifyCount: 1,
			verifyContains:  context.Canceled.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := tt.contextFn(t)

			proposal := scheduleProposal(mcmstypes.NewDuration(0))
			delays, verifyErrors := tt.read(ctx, &proposal)

			if tt.wantDelays != nil {
				require.Equal(t, tt.wantDelays, delays)
			} else {
				require.Empty(t, delays)
			}
			require.Len(t, verifyErrors, tt.wantVerifyCount)
			if tt.verifyContains != "" {
				require.Contains(t, verifyErrors[0], tt.verifyContains)
			}
		})
	}
}

func TestSortedTimelockEntries(t *testing.T) {
	t.Parallel()

	multiChainProposal := scheduleMultiChainProposal(mcmstypes.NewDuration(0))

	tests := []struct {
		name     string
		proposal *mcms.TimelockProposal
		wantLen  int
		want     []timelockChainEntry
	}{
		{
			name:     "nil proposal",
			proposal: nil,
		},
		{
			name:     "empty timelock addresses",
			proposal: &mcms.TimelockProposal{},
		},
		{
			name:     "sorts selectors",
			proposal: &multiChainProposal,
			wantLen:  2,
			want: []timelockChainEntry{
				{selector: 1, address: "0xTimelock1"},
				{selector: 2, address: "0xTimelock2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entries := sortedTimelockEntries(tt.proposal)
			require.Len(t, entries, tt.wantLen)
			require.Equal(t, tt.want, entries)
		})
	}
}

func TestChainMetadataForInspector(t *testing.T) {
	t.Parallel()

	proposal := scheduleProposal(mcmstypes.NewDuration(0))

	tests := []struct {
		name          string
		proposal      *mcms.TimelockProposal
		chainSelector uint64
		want          mcmstypes.ChainMetadata
	}{
		{
			name:          "nil proposal",
			proposal:      nil,
			chainSelector: 1,
		},
		{
			name:          "empty chain metadata",
			proposal:      &mcms.TimelockProposal{},
			chainSelector: 1,
		},
		{
			name:          "known selector",
			proposal:      &proposal,
			chainSelector: 1,
			want:          mcmstypes.ChainMetadata{MCMAddress: "0xMCMS"},
		},
		{
			name:          "unknown selector",
			proposal:      &proposal,
			chainSelector: 99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, chainMetadataForInspector(tt.proposal, tt.chainSelector))
		})
	}
}

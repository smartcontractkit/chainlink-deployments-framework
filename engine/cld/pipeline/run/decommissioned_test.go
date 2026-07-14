package run

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	chainselremote "github.com/smartcontractkit/chain-selectors/remote"
	"github.com/stretchr/testify/require"
)

// stubChainDetailsChecker is a test-only chainDetailsChecker that returns
// pre-configured details for selectors in the map. Selectors not in the map
// return an "unknown chain selector" error, mimicking the real package's
// behavior for unknown selectors.
type stubChainDetailsChecker struct {
	details map[uint64]chainselremote.ChainDetailsWithMetadata
}

func (s stubChainDetailsChecker) GetChainDetails(_ context.Context, selector uint64) (chainselremote.ChainDetailsWithMetadata, error) {
	if d, ok := s.details[selector]; ok {
		return d, nil
	}

	return chainselremote.ChainDetailsWithMetadata{}, fmt.Errorf("unknown chain selector %d", selector)
}

// networkErrorChecker simulates a transient network failure for all lookups.
type networkErrorChecker struct{}

func (networkErrorChecker) GetChainDetails(context.Context, uint64) (chainselremote.ChainDetailsWithMetadata, error) {
	return chainselremote.ChainDetailsWithMetadata{}, errors.New("failed to fetch remote selectors from github.com: connection refused")
}

// formatSelector renders a selector as "selector (name)" for test assertions,
// matching the format used by checkDecommissionedChains. Uses the local-only
// chainsel package to avoid network calls during test setup.
func formatSelector(sel uint64) string {
	name, err := chainsel.GetChainNameFromSelector(sel)
	if err != nil {
		return strconv.FormatUint(sel, 10)
	}

	return fmt.Sprintf("%s (%s)", strconv.FormatUint(sel, 10), name)
}

// chainDetails is a test helper to build ChainDetailsWithMetadata with just
// the fields checkDecommissionedChains reads.
func chainDetails(selector uint64, name string, deprecated bool) chainselremote.ChainDetailsWithMetadata {
	return chainselremote.ChainDetailsWithMetadata{
		ChainDetails: chainsel.ChainDetails{
			ChainSelector: selector,
			ChainName:     name,
			Deprecated:    deprecated,
		},
	}
}

func TestCheckDecommissionedChains(t *testing.T) {
	t.Parallel()

	// Use real selectors so the error message includes human-readable names.
	eth := chainsel.ETHEREUM_MAINNET.Selector
	avax := chainsel.AVALANCHE_MAINNET.Selector

	tests := []struct {
		name      string
		checker   chainDetailsChecker
		selectors []uint64
		wantErr   string
	}{
		{
			name:      "nil selectors",
			checker:   stubChainDetailsChecker{details: map[uint64]chainselremote.ChainDetailsWithMetadata{}},
			selectors: nil,
		},
		{
			name:      "empty selectors",
			checker:   stubChainDetailsChecker{details: map[uint64]chainselremote.ChainDetailsWithMetadata{}},
			selectors: []uint64{},
		},
		{
			name: "all active chains",
			checker: stubChainDetailsChecker{details: map[uint64]chainselremote.ChainDetailsWithMetadata{
				eth:  chainDetails(eth, "ethereum-mainnet", false),
				avax: chainDetails(avax, "avalanche-mainnet", false),
			}},
			selectors: []uint64{eth, avax},
		},
		{
			name: "single decommissioned chain",
			checker: stubChainDetailsChecker{details: map[uint64]chainselremote.ChainDetailsWithMetadata{
				eth: chainDetails(eth, "ethereum-mainnet", true),
			}},
			selectors: []uint64{eth},
			wantErr:   "chain overrides contain 1 decommissioned chain(s): " + formatSelector(eth) + "; remove them or replace with an active chain",
		},
		{
			name: "mixed active and decommissioned",
			checker: stubChainDetailsChecker{details: map[uint64]chainselremote.ChainDetailsWithMetadata{
				avax: chainDetails(avax, "avalanche-mainnet", false),
				eth:  chainDetails(eth, "ethereum-mainnet", true),
			}},
			selectors: []uint64{avax, eth},
			wantErr:   "chain overrides contain 1 decommissioned chain(s): " + formatSelector(eth) + "; remove them or replace with an active chain",
		},
		{
			name: "all decommissioned",
			checker: stubChainDetailsChecker{details: map[uint64]chainselremote.ChainDetailsWithMetadata{
				eth:  chainDetails(eth, "ethereum-mainnet", true),
				avax: chainDetails(avax, "avalanche-mainnet", true),
			}},
			selectors: []uint64{eth, avax},
			wantErr:   "chain overrides contain 2 decommissioned chain(s): " + formatSelector(eth) + ", " + formatSelector(avax) + "; remove them or replace with an active chain",
		},
		{
			name:      "unknown selector returns error",
			checker:   stubChainDetailsChecker{details: map[uint64]chainselremote.ChainDetailsWithMetadata{}},
			selectors: []uint64{9999999999999999999},
			wantErr:   "failed to look up chain details for selector 9999999999999999999: unknown chain selector 9999999999999999999",
		},
		{
			name:      "network error is propagated",
			checker:   networkErrorChecker{},
			selectors: []uint64{eth},
			wantErr:   "failed to look up chain details for selector " + strconv.FormatUint(eth, 10) + ": failed to fetch remote selectors from github.com: connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := checkDecommissionedChains(t.Context(), tt.checker, tt.selectors)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Equal(t, tt.wantErr, err.Error())

				return
			}
			require.NoError(t, err)
		})
	}
}

package run

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	chainselremote "github.com/smartcontractkit/chain-selectors/remote"
)

// chainDetailsChecker looks up chain details (including the Deprecated flag)
// for a given selector. The default implementation delegates to the
// chain-selectors package's remote API, which checks local embedded data first
// and falls back to fetching the latest all_selectors.yml from GitHub; tests
// can supply a stub to exercise the error path without network calls.
type chainDetailsChecker interface {
	GetChainDetails(ctx context.Context, selector uint64) (chainselremote.ChainDetailsWithMetadata, error)
}

// defaultChainDetailsChecker uses chainselremote.GetChainDetailsBySelector,
// which checks local embedded data first and falls back to a remote fetch for
// chains not found locally. This ensures deprecations are caught even before
// the chain-selectors module is bumped to a version that includes them in the
// embedded YAML.
type defaultChainDetailsChecker struct{}

func (defaultChainDetailsChecker) GetChainDetails(ctx context.Context, selector uint64) (chainselremote.ChainDetailsWithMetadata, error) {
	return chainselremote.GetChainDetailsBySelector(ctx, selector)
}

// checkDecommissionedChains validates that none of the provided chain selectors
// have been marked as decommissioned/deprecated. It collects all offending
// selectors and returns a single error listing every one, so the caller sees
// the full picture in a single failure rather than discovering them one at a
// time.
//
// If the checker returns an error for any selector (e.g. unknown selector or
// transient network failure during remote fetch), the error is propagated
// immediately. An unknown selector would fail in LoadChains later anyway, so
// failing early here gives a clearer error message.
func checkDecommissionedChains(ctx context.Context, checker chainDetailsChecker, selectors []uint64) error {
	var decommissioned []string
	for _, sel := range selectors {
		details, err := checker.GetChainDetails(ctx, sel)
		if err != nil {
			return fmt.Errorf("failed to look up chain details for selector %d: %w", sel, err)
		}

		if details.Deprecated {
			decommissioned = append(decommissioned, fmt.Sprintf("%s (%s)", strconv.FormatUint(sel, 10), details.ChainName))
		}
	}

	if len(decommissioned) > 0 {
		return fmt.Errorf(
			"chain overrides contain %d decommissioned chain(s): %s; "+
				"remove them or replace with an active chain",
			len(decommissioned), strings.Join(decommissioned, ", "),
		)
	}

	return nil
}

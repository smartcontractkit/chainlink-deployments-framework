package mcmsutils

import (
	"context"
	"fmt"

	chainselectors "github.com/smartcontractkit/chain-selectors"
	mcmslib "github.com/smartcontractkit/mcms"
	mcmssdk "github.com/smartcontractkit/mcms/sdk"
	mcmstypes "github.com/smartcontractkit/mcms/types"
)

// convertTimelock converts a timelock proposal to an MCMS proposal by creating chain-specific
// converters for each chain in the proposal and then performing the conversion.
func convertTimelock(ctx context.Context, proposal mcmslib.TimelockProposal) (*mcmslib.Proposal, error) {
	converters := make(map[mcmstypes.ChainSelector]mcmssdk.TimelockConverter, 0)
	for selector := range proposal.ChainMetadata {
		family, err := chainselectors.GetSelectorFamily(uint64(selector))
		if err != nil {
			return nil, fmt.Errorf("failed to get selector family for chain (selector: %d): %w",
				selector, err,
			)
		}

		convFactory, err := GetConverterFactory(family)
		if err != nil {
			return nil, fmt.Errorf("failed to get converter factory for chain (selector: %d, family: %s): %w",
				selector, family, err,
			)
		}

		converter, err := convFactory.Make()
		if err != nil {
			return nil, fmt.Errorf("failed to create converter for chain (selector: %d, family: %s): %w",
				selector, family, err,
			)
		}

		converters[selector] = converter
	}

	p, _, err := proposal.Convert(ctx, converters)
	if err != nil {
		return nil, fmt.Errorf("failed to convert timelock proposal to MCMS proposal: %w", err)
	}

	return &p, nil
}

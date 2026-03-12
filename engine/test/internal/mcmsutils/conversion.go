package mcmsutils

import (
	"context"
	"fmt"

	mcmslib "github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/chainwrappers"
)

// convertTimelock converts a timelock proposal to an MCMS proposal by creating chain-specific
// converters for each chain in the proposal and then performing the conversion.
func convertTimelock(ctx context.Context, proposal mcmslib.TimelockProposal) (*mcmslib.Proposal, error) {
	converters, err := chainwrappers.BuildConverters(proposal.ChainMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to build converters: %w", err)
	}

	p, _, err := proposal.Convert(ctx, converters)
	if err != nil {
		return nil, fmt.Errorf("failed to convert timelock proposal to MCMS proposal: %w", err)
	}

	return &p, nil
}

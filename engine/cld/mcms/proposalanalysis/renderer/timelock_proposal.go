package renderer

import (
	"bytes"
	"fmt"

	"github.com/smartcontractkit/mcms"
)

// CloneTimelockProposal returns a deep copy of proposal using JSON round-trip.
// It returns nil when proposal is nil.
func CloneTimelockProposal(proposal *mcms.TimelockProposal) (*mcms.TimelockProposal, error) {
	if proposal == nil {
		return nil, nil //nolint:nilnil // nil proposal is valid input; callers treat absence as optional metadata
	}

	var buf bytes.Buffer
	if err := mcms.WriteTimelockProposal(&buf, proposal); err != nil {
		return nil, fmt.Errorf("marshal timelock proposal: %w", err)
	}

	cloned, err := mcms.NewTimelockProposal(&buf)
	if err != nil {
		return nil, fmt.Errorf("unmarshal timelock proposal: %w", err)
	}

	return cloned, nil
}

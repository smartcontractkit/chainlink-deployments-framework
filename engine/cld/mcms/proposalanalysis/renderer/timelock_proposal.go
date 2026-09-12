package renderer

import (
	"bytes"
	"encoding/json"
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

	// Cloning must preserve an already-loaded proposal without reapplying
	// wall-clock validity rules. Historical analysis accepts expired proposals.
	var cloned mcms.TimelockProposal
	decoder := json.NewDecoder(&buf)
	decoder.UseNumber()
	if err := decoder.Decode(&cloned); err != nil {
		return nil, fmt.Errorf("unmarshal timelock proposal: %w", err)
	}

	return &cloned, nil
}

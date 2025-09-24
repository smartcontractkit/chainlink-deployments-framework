package mcmsutils

import (
	"bytes"

	mcmslib "github.com/smartcontractkit/mcms"
)

// EncodeProposal serializes a standard MCMS Proposal to a JSON string format.
func EncodeProposal(proposal *mcmslib.Proposal) (string, error) {
	b := bytes.NewBuffer([]byte{})
	if err := mcmslib.WriteProposal(b, proposal); err != nil {
		return "", err
	}

	return b.String(), nil
}

// DecodeProposal deserializes a JSON string back into a standard MCMS Proposal struct.
func DecodeProposal(proposal string) (*mcmslib.Proposal, error) {
	return mcmslib.NewProposal(bytes.NewReader([]byte(proposal)))
}

// EncodeTimelockProposal serializes a TimelockProposal to a JSON string format.
func EncodeTimelockProposal(proposal *mcmslib.TimelockProposal) (string, error) {
	b := bytes.NewBuffer([]byte{})
	if err := mcmslib.WriteTimelockProposal(b, proposal); err != nil {
		return "", err
	}

	return b.String(), nil
}

// DecodeTimelockProposal deserializes a JSON string back into a TimelockProposal struct.
func DecodeTimelockProposal(proposal string) (*mcmslib.TimelockProposal, error) {
	return mcmslib.NewTimelockProposal(bytes.NewReader([]byte(proposal)))
}

package mcms

import (
	"encoding/json"
	"errors"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/sdk/sui"
	"github.com/smartcontractkit/mcms/types"
)

// suiMetadataFromProposal extracts Sui metadata from a timelock proposal.
func suiMetadataFromProposal(selector types.ChainSelector, proposal *mcms.TimelockProposal) (sui.AdditionalFieldsMetadata, error) {
	if proposal == nil {
		return sui.AdditionalFieldsMetadata{}, errors.New("sui timelock proposal is needed")
	}

	var metadata sui.AdditionalFieldsMetadata
	err := json.Unmarshal([]byte(proposal.ChainMetadata[selector].AdditionalFields), &metadata)
	if err != nil {
		return sui.AdditionalFieldsMetadata{}, err
	}

	err = metadata.Validate()
	if err != nil {
		return sui.AdditionalFieldsMetadata{}, err
	}

	return metadata, nil
}

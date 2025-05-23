package datastore

import (
	"encoding/json"
	"errors"
)

var ErrContractMetadataNotFound = errors.New("no contract metadata record can be found for the provided key")
var ErrContractMetadataExists = errors.New("a contract metadata record with the supplied key already exists")

// ContractMetadata implements the Record interface
var _ UniqueRecord[ContractMetadataKey, ContractMetadata] = ContractMetadata{}

// ContractMetadata is a generic struct that holds the metadata for a contract on a specific chain.
// It implements the Record interface and is used to store contract metadata in the datastore.
// The metadata is generic and can be of any type that implements the Cloneable interface.
type ContractMetadata struct {
	// Address is the address of the contract on the chain.
	Address string `json:"address"`
	// ChainSelector is the chain-selector of the chain where the contract is deployed.
	ChainSelector uint64 `json:"chainSelector"`
	// Metadata is the metadata associated with the contract.
	// It is a generic type that can be of any type that implements the Cloneable interface.
	Metadata CustomMetadata `json:"metadata"`
}

// Clone creates a copy of the ContractMetadata.
// The Metadata field is cloned using the Clone method of the Cloneable interface.
func (r ContractMetadata) Clone() ContractMetadata {
	return ContractMetadata{
		ChainSelector: r.ChainSelector,
		Address:       r.Address,
		Metadata:      r.Metadata.Clone(),
	}
}

// Key returns the ContractMetadataKey for the ContractMetadata.
// It is used to uniquely identify the contract metadata in the datastore.
func (r ContractMetadata) Key() ContractMetadataKey {
	return NewContractMetadataKey(r.ChainSelector, r.Address)
}

// Custom unmarshaler that uses DeferredMetadata
func (c *ContractMetadata) UnmarshalJSON(data []byte) error {
	type alias ContractMetadata // avoid recursion
	tmp := struct {
		Metadata json.RawMessage `json:"metadata"`
		*alias
	}{
		alias: (*alias)(c),
	}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	c.Metadata = RawMetadata{raw: tmp.Metadata}
	return nil
}

// Custom marshaler that outputs Metadata as raw JSON
func (c ContractMetadata) MarshalJSON() ([]byte, error) {
	type alias ContractMetadata // avoid recursion
	tmp := struct {
		Metadata json.RawMessage `json:"metadata"`
		alias
	}{
		alias: (alias)(c),
	}
	if c.Metadata != nil {
		b, err := json.Marshal(c.Metadata)
		if err != nil {
			return nil, err
		}
		tmp.Metadata = b
	}
	return json.Marshal(tmp)
}

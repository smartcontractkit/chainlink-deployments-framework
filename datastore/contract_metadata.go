package datastore

import "errors"

var ErrContractMetadataNotFound = errors.New("no contract metadata record can be found for the provided key")
var ErrContractMetadataExists = errors.New("a contract metadata record with the supplied key already exists")

// ContractMetadata implements the UniqueRecord interface
var _ UniqueRecord[ContractMetadataKey, ContractMetadata] = ContractMetadata{}

// ContractMetadata is a struct that holds the metadata for a contract on a specific chain.
// It implements the UniqueRecord interface and is used to store contract metadata in the datastore.
// NOTE: Metadata can be of any type. To convert from any to a specific type, use the utility method As.
type ContractMetadata struct {
	// Address is the address of the contract on the chain.
	Address string `json:"address"`
	// ChainSelector is the chain-selector of the chain where the contract is deployed.
	ChainSelector uint64 `json:"chainSelector"`
	// Metadata is the metadata associated with the contract.
	Metadata any `json:"metadata"`
}

// Clone creates a copy of the ContractMetadata.
func (r ContractMetadata) Clone() (ContractMetadata, error) {
	metaClone, err := clone(r.Metadata)
	if err != nil {
		return ContractMetadata{}, err
	}

	return ContractMetadata{
		ChainSelector: r.ChainSelector,
		Address:       r.Address,
		Metadata:      metaClone,
	}, nil
}

// Key returns the ContractMetadataKey for the ContractMetadata.
// It is used to uniquely identify the contract metadata in the datastore.
func (r ContractMetadata) Key() ContractMetadataKey {
	return NewContractMetadataKey(r.ChainSelector, r.Address)
}

package datastore

import (
	"encoding/json"
)

// As is a utility function that converts a source value of any type to a destination type T.
// It uses JSON marshaling and unmarshaling to perform the conversion. It can be used to
// convert metadata of any type to a specific type, as shown in the example below.
//
// Example usage:
//
//	record, err := store.ContractMetadata().Get(NewContractMetadataKey(chainSelector, address))
//	if err != nil {
//	    return nil, err
//	}
//	concrete, err := As[ConcreteMetadataType](record.Metadata)
//	if err != nil {
//	    return nil, err
//	}
func As[T any](src any) (T, error) {
	var zero T
	bytes, err := json.Marshal(src)
	if err != nil {
		return zero, err
	}

	var dst T
	err = json.Unmarshal(bytes, &dst)

	return dst, err
}

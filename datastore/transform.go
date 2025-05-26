package datastore

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ToDefault is a utility function that converts a DataStore with domain specific
// metadata types into a DataStore with DefaultMetadata types.
// NOTE: It is assumed that the domain specific metadata types are JSON serializable.
func ToDefault[CM Cloneable[CM], EM Cloneable[EM]](
	dataStore DataStore[CM, EM],
) (MutableDataStore[DefaultMetadata, DefaultMetadata], error) {
	converted := NewMemoryDataStore[DefaultMetadata, DefaultMetadata]()

	// Copy all addressRef over to the new data store, no conversion is needed
	addressRefs, addrErr := dataStore.Addresses().Fetch()
	if addrErr != nil {
		return nil, fmt.Errorf("error fetching AddressRefs: %w", addrErr)
	}

	for _, ar := range addressRefs {
		addrErr = converted.Addresses().Add(ar)
		if addrErr != nil {
			return nil, fmt.Errorf("error adding AddressRef: for %s@%v: %w",
				ar.Address, ar.ChainSelector, addrErr)
		}
	}

	// Copy all contractMetadata over to the new data store and convert the metadata
	// to a JSON string. This is done by marshaling the metadata into a JSON string.
	contractMetadata, cmetaErr := dataStore.ContractMetadata().Fetch()
	if cmetaErr != nil {
		return nil, fmt.Errorf("error fetching ContractMetadata: %w", cmetaErr)
	}

	for _, cm := range contractMetadata {
		jsonData, cmetaErr := json.Marshal(cm.Metadata)
		if cmetaErr != nil {
			return nil, fmt.Errorf("error marshaling ContractMetadata for %s@%v: %w",
				cm.Address, cm.ChainSelector, cmetaErr)
		}

		cmetaErr = converted.ContractMetadata().Add(ContractMetadata[DefaultMetadata]{
			ChainSelector: cm.ChainSelector,
			Address:       cm.Address,
			Metadata: DefaultMetadata{
				Data: string(jsonData),
			},
		})
		if cmetaErr != nil {
			return nil, fmt.Errorf("error adding ContractMetadata for %s@%v: %w",
				cm.Address, cm.ChainSelector, cmetaErr)
		}
	}

	// Fetch the EnvMetadata and check if it was set.
	envMetadata, envmetaErr := dataStore.EnvMetadata().Get()
	if envmetaErr != nil {
		if errors.Is(envmetaErr, ErrEnvMetadataNotSet) {
			// If the env metadata was not set, Get() will return ErrEnvMetadataNotSet.
			// In this case, we don't need to do anything.
			return converted, nil
		}

		return nil, envmetaErr
	}

	// Convert the EnvMetadata to a JSON string. This is done by marshaling the metadata
	// into a JSON string.
	jsonData, envmetaErr := json.Marshal(envMetadata.Metadata)
	if envmetaErr != nil {
		return nil, fmt.Errorf("error marshaling EnvMetadata: %w", envmetaErr)
	}

	// Set the EnvMetadata in the new data store with the JSON string.
	envmetaErr = converted.EnvMetadata().Set(EnvMetadata[DefaultMetadata]{
		Metadata: DefaultMetadata{
			Data: string(jsonData),
		},
	})
	if envmetaErr != nil {
		return nil, fmt.Errorf("error updating EnvMetadata: %w", envmetaErr)
	}

	return converted, nil
}

// FromDefault is a utility function that converts a DataStore with DefaultMetadata types
// into a DataStore with domain specific metadata types.
// NOTE: It is assumed that the domain specific metadata types are JSON deserializable.
func FromDefault[CM Cloneable[CM], EM Cloneable[EM]](
	defaultStore DataStore[DefaultMetadata, DefaultMetadata],
) (DataStore[CM, EM], error) {
	converted := NewMemoryDataStore[CM, EM]()

	// Copy all addressRef over to the new data store, no conversion is needed
	addressRefs, addrErr := defaultStore.Addresses().Fetch()
	if addrErr != nil {
		return nil, fmt.Errorf("error fetching AddressRefs: %w", addrErr)
	}

	for _, ar := range addressRefs {
		addrErr = converted.Addresses().Add(ar)
		if addrErr != nil {
			return nil, fmt.Errorf("error adding AddressRef: for %s@%v: %w",
				ar.Address, ar.ChainSelector, addrErr)
		}
	}

	// Copy all contractMetadata over to the new data store and convert the metadata
	// to the domain specific type. This is done by unmarshaling the JSON string
	// representing the metadata into the concrete type.
	contractMetadata, cmetaErr := defaultStore.ContractMetadata().Fetch()
	if cmetaErr != nil {
		return nil, fmt.Errorf("error fetching ContractMetadata: %w", cmetaErr)
	}

	for _, cm := range contractMetadata {
		var metadata CM
		cmetaErr = json.Unmarshal([]byte(cm.Metadata.Data), &metadata)
		if cmetaErr != nil {
			return nil, fmt.Errorf("error unmarshaling ContractMetadata for %s@%v: %w",
				cm.Address, cm.ChainSelector, cmetaErr)
		}

		cmetaErr = converted.ContractMetadata().Add(ContractMetadata[CM]{
			ChainSelector: cm.ChainSelector,
			Address:       cm.Address,
			Metadata:      metadata,
		})
		if cmetaErr != nil {
			return nil, fmt.Errorf("error adding ContractMetadata for %s@%v: %w",
				cm.Address, cm.ChainSelector, cmetaErr)
		}
	}

	// Fetch the EnvMetadata and check if it was set.
	envMetadata, envmetaErr := defaultStore.EnvMetadata().Get()
	if envmetaErr != nil {
		if errors.Is(envmetaErr, ErrEnvMetadataNotSet) {
			// If the env metadata was not set, Get() will return ErrEnvMetadataNotSet.
			// In this case, we don't need to do anything.
			return converted.Seal(), nil
		}

		return nil, envmetaErr
	}

	// Convert the EnvMetadata to the domain specific type. This is done by unmarshaling
	// the JSON string representing the metadata into the concrete type.
	var metadata EM
	envmetaErr = json.Unmarshal([]byte(envMetadata.Metadata.Data), &metadata)
	if envmetaErr != nil {
		return nil, fmt.Errorf("error unmarshaling EnvMetadata: %w", envmetaErr)
	}

	// Set the EnvMetadata in the new data store with the domain specific type.
	envmetaErr = converted.EnvMetadata().Set(EnvMetadata[EM]{
		Metadata: metadata,
	})
	if envmetaErr != nil {
		return nil, fmt.Errorf("error updating EnvMetadata: %w", envmetaErr)
	}

	return converted.Seal(), nil
}

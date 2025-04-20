package deployment

import (
	"github.com/smartcontractkit/chainlink/deployment/datastore"
)

// AddressBookToDataStore converts an AddressBook to a DataStore
// AddressBook is deprecated and will be removed in the future. You can use this function to migrate or interact with legacy code that uses AddressBook.
func AddressBookToDataStore[CM datastore.Cloneable[CM], EM datastore.Cloneable[EM]](ab AddressBook) (datastore.DataStore[CM, EM], error) {
	ds := datastore.NewMemoryDataStore[CM, EM]()

	// Get all addresses from the AddressBook
	addresses, err := ab.Addresses()
	if err != nil {
		return nil, err
	}

	// For each address, create an AddressRef and add it to the DataStore
	for chainSelector, chainAddresses := range addresses {
		for address, tv := range chainAddresses {
			// Create an AddressRef with the chain selector stored in ChainSelector field
			addressRef := datastore.AddressRef{
				Address:       address,
				ChainSelector: chainSelector,
				Labels:        datastore.LabelSet(tv.Labels),
				Type:          datastore.ContractType(tv.Type),
				Version:       &tv.Version,
			}

			// Add the AddressRef to the DataStore
			err := ds.Addresses().Upsert(addressRef)
			if err != nil {
				return nil, err
			}
		}
	}

	return ds.Seal(), nil
}

// DataStoreToAddressBook converts a DataStore to an AddressBook
// DataStore ContractMetadata and EnvMetadata are not preserved in the AddressBook.
// AddressBook is deprecated and will be removed in the future. You can use this function to migrate or interact with legacy code that uses AddressBook.
func DataStoreToAddressBook[CM datastore.Cloneable[CM], EM datastore.Cloneable[EM]](ds datastore.DataStore[CM, EM]) (AddressBook, error) {
	ab := NewMemoryAddressBook()

	// Get all addresses from the DataStore
	addressRefs, err := ds.Addresses().Fetch()
	if err != nil {
		return nil, err
	}

	// For each address, create a TypeAndVersion and add it to the AddressBook
	for _, addressRef := range addressRefs {
		// Create a TypeAndVersion
		tv := TypeAndVersion{
			Type:    ContractType(addressRef.Type),
			Version: *addressRef.Version,
			Labels:  LabelSet(addressRef.Labels),
		}

		// Add the TypeAndVersion to the AddressBook
		err := ab.Save(addressRef.ChainSelector, addressRef.Address, tv)
		if err != nil {
			return nil, err
		}
	}

	return ab, nil
}

// AddressBookToNewDataStore converts an AddressBook to a new mutable DataStore
// AddressBook is deprecated and will be removed in the future. You can use this function to migrate or interact with legacy code that uses AddressBook.
func AddressBookToNewDataStore[CM datastore.Cloneable[CM], EM datastore.Cloneable[EM]](ab AddressBook) (*datastore.MemoryDataStore[CM, EM], error) {
	ds := datastore.NewMemoryDataStore[CM, EM]()

	// Get all addresses from the AddressBook
	addresses, err := ab.Addresses()
	if err != nil {
		return nil, err
	}

	// For each address, create an AddressRef and add it to the DataStore
	for chainSelector, chainAddresses := range addresses {
		for address, tv := range chainAddresses {
			addressRef := datastore.AddressRef{
				Address:       address,
				ChainSelector: chainSelector,
				Labels:        datastore.LabelSet(tv.Labels),
				Type:          datastore.ContractType(tv.Type),
				Version:       &tv.Version,
			}

			err := ds.Addresses().Upsert(addressRef)
			if err != nil {
				return nil, err
			}
		}
	}

	return ds, nil
}

package deployment

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

// AddressBookToDataStore converts an AddressBook to a DataStore
// AddressBook is deprecated and will be removed in the future. You can use this function to migrate or interact with legacy code that uses AddressBook.
func AddressBookToDataStore[CM datastore.Cloneable[CM], EM datastore.Cloneable[EM]](ab AddressBook) (datastore.DataStore[CM, EM], error) {
	ds, err := AddressBookToMutableDataStore[CM, EM](ab)
	if err != nil {
		return nil, err
	}
	return ds.Seal(), nil
}

// AddressBookToMutableDataStore converts an AddressBook to a new mutable DataStore
// AddressBook is deprecated and will be removed in the future. You can use this function to migrate or interact with legacy code that uses AddressBook.
func AddressBookToMutableDataStore[CM datastore.Cloneable[CM], EM datastore.Cloneable[EM]](ab AddressBook) (datastore.MutableDataStore[CM, EM], error) {
	ds := datastore.NewMemoryDataStore[CM, EM]()

	addresses, err := ab.Addresses()
	if err != nil {
		return nil, err
	}

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

// DataStoreToAddressBook converts a DataStore to an AddressBook
// DataStore ContractMetadata and EnvMetadata are not preserved in the resulting AddressBook.
// AddressBook is deprecated and will be removed in the future. You can use this function to migrate or interact with legacy code that uses AddressBook.
// If you have a MutableDataStore `Seal()` it before passing it to this function.
func DataStoreToAddressBook[CM datastore.Cloneable[CM], EM datastore.Cloneable[EM]](ds datastore.DataStore[CM, EM]) (AddressBook, error) {
	ab := NewMemoryAddressBook()

	addressRefs, err := ds.Addresses().Fetch()
	if err != nil {
		return nil, err
	}

	for _, addressRef := range addressRefs {
		tv := TypeAndVersion{
			Type:    ContractType(addressRef.Type),
			Version: *addressRef.Version,
			Labels:  LabelSet(addressRef.Labels),
		}
		err := ab.Save(addressRef.ChainSelector, addressRef.Address, tv)
		if err != nil {
			return nil, err
		}
	}

	return ab, nil
}

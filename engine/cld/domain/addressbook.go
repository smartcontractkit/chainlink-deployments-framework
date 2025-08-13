package domain

import (
	"errors"
	"fmt"

	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/lib/jsonutils"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

// MigrateAddressBook migrates the address book for the domain's environment directory
// to the new datastore format. It reads the existing address book and converts its records.
// When converting address book entries to datastore addressRefs, some assumptions are made to
// guarantee the conversion is successful.
func (d EnvDir) MigrateAddressBook() error {
	addrBook, err := d.AddressBook()
	if err != nil {
		return err
	}

	addrs, err := addrBook.Addresses()
	if err != nil {
		return err
	}

	ds := datastore.NewMemoryDataStore()

	for chainSelector, chainAddresses := range addrs {
		for addr, typever := range chainAddresses {
			ref := datastore.AddressRef{
				ChainSelector: chainSelector,
				Address:       addr,
				Type:          datastore.ContractType(typever.Type),
				Version:       &typever.Version,
				// Since the address book does not have a qualifier, we use the address and type as a
				// unique identifier for the addressRef. Otherwise, we would have some clashes in the
				// between address refs.
				Qualifier: fmt.Sprintf("%s-%s", addr, typever.Type),
			}

			// If the address book has labels, we need to add them to the addressRef
			if !typever.Labels.IsEmpty() {
				ref.Labels = datastore.NewLabelSet(typever.Labels.List()...)
			}

			if err = ds.Addresses().Add(ref); err != nil {
				return err
			}
		}
	}

	err = jsonutils.WriteFile(d.AddressRefsFilePath(), ds.AddressRefStore.Records)
	if err != nil {
		return errors.New("failed to write address refs store file")
	}

	err = jsonutils.WriteFile(d.ChainMetadataFilePath(), ds.ChainMetadataStore.Records)
	if err != nil {
		return errors.New("failed to write chain metadata store file")
	}

	err = jsonutils.WriteFile(d.ContractMetadataFilePath(), ds.ContractMetadataStore.Records)
	if err != nil {
		return errors.New("failed to write contract metadata store file %err, err")
	}

	err = jsonutils.WriteFile(d.EnvMetadataFilePath(), ds.EnvMetadataStore.Record)
	if err != nil {
		return errors.New("failed to write environment datastore file")
	}

	return nil
}

func loadAddressBookByMigrationKey(artDir *ArtifactsDir, migKey, timestamp string) (cldf.AddressBook, error) {
	// Set the durable pipelines directory and timestamp if provided
	if timestamp != "" {
		if err := artDir.SetDurablePipelines(timestamp); err != nil {
			return &cldf.AddressBookMap{}, err
		}
	}

	// Load the migration address book where the artifacts group name is the migration key
	migAddrBook, err := artDir.LoadAddressBookByMigrationKey(migKey)
	if err != nil {
		if errors.Is(err, ErrArtifactNotFound) {
			fmt.Println("No migration address book found, skipping merge")

			return &cldf.AddressBookMap{}, nil
		}

		return &cldf.AddressBookMap{}, err
	}

	return migAddrBook, nil
}

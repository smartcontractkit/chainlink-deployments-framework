package domain

import (
	"errors"
	"fmt"
	"strings"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/internal/jsonutils"

	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

// MigrateAddressBookOptions configures address book migration behavior.
type MigrateAddressBookOptions struct {
	// PreserveExisting keeps existing address refs and only adds address book
	// entries whose chain, address, type, and version are not already present.
	// Other datastore files are left unchanged when this is set.
	PreserveExisting bool
	// ChainSelector limits migration to a single chain. Zero migrates all chains.
	ChainSelector uint64
}

type addressRefIdentity struct {
	chainSelector uint64
	address       string
	contractType  fdatastore.ContractType
	version       string
}

// MigrateAddressBook migrates the address book for the domain's environment directory
// to the new datastore format. It reads the existing address book and converts its records.
// When converting address book entries to datastore addressRefs, some assumptions are made to
// guarantee the conversion is successful.
func (d EnvDir) MigrateAddressBook(opts MigrateAddressBookOptions) error {
	addrBook, err := d.AddressBook()
	if err != nil {
		return err
	}

	addrs, err := addrBook.Addresses()
	if err != nil {
		return err
	}

	chainScoped := opts.ChainSelector != 0
	addressRefsOnly := opts.PreserveExisting || chainScoped

	var ds fdatastore.MutableDataStore
	if addressRefsOnly {
		ds, err = d.MutableDataStore()
		if err != nil {
			return err
		}
	} else {
		ds = fdatastore.NewMemoryDataStore()
	}

	existing := make(map[addressRefIdentity]struct{})
	if opts.PreserveExisting {
		refs, fetchErr := ds.Addresses().Fetch()
		if fetchErr != nil {
			return fetchErr
		}
		for _, ref := range refs {
			existing[addressRefIdentityFromRef(ref)] = struct{}{}
		}
	}

	if chainScoped && !opts.PreserveExisting {
		refs, fetchErr := ds.Addresses().Fetch()
		if fetchErr != nil {
			return fetchErr
		}
		for _, ref := range refs {
			if ref.ChainSelector != opts.ChainSelector {
				continue
			}
			if err = ds.Addresses().Delete(ref.Key()); err != nil {
				return err
			}
		}
	}

	for chainselector, chainAddresses := range addrs {
		if chainScoped && chainselector != opts.ChainSelector {
			continue
		}

		for addr, typever := range chainAddresses {
			if opts.PreserveExisting {
				if _, found := existing[addressRefIdentityFromAddressBook(chainselector, addr, typever)]; found {
					continue
				}
			}

			ref := fdatastore.AddressRef{
				ChainSelector: chainselector,
				Address:       addr,
				Type:          fdatastore.ContractType(typever.Type),
				Version:       &typever.Version,
				// Since the address book does not have a qualifier, we use the address and type as a
				// unique identifier for the addressRef. Otherwise, we would have some clashes in the
				// between address refs.
				Qualifier: fmt.Sprintf("%s-%s", addr, typever.Type),
			}

			// If the address book has labels, we need to add them to the addressRef
			if !typever.Labels.IsEmpty() {
				ref.Labels = fdatastore.NewLabelSet(typever.Labels.List()...)
			}

			if err = ds.Addresses().Add(ref); err != nil {
				return err
			}
		}
	}

	if addressRefsOnly {
		return d.writeAddressRefs(ds)
	}

	err = jsonutils.WriteFile(d.AddressRefsFilePath(), ds.(*fdatastore.MemoryDataStore).AddressRefStore.Records)
	if err != nil {
		return errors.New("failed to write address refs store file")
	}

	err = jsonutils.WriteFile(d.ChainMetadataFilePath(), ds.(*fdatastore.MemoryDataStore).ChainMetadataStore.Records)
	if err != nil {
		return errors.New("failed to write chain metadata store file")
	}

	err = jsonutils.WriteFile(d.ContractMetadataFilePath(), ds.(*fdatastore.MemoryDataStore).ContractMetadataStore.Records)
	if err != nil {
		return errors.New("failed to write contract metadata store file %err, err")
	}

	err = jsonutils.WriteFile(d.EnvMetadataFilePath(), ds.(*fdatastore.MemoryDataStore).EnvMetadataStore.Record)
	if err != nil {
		return errors.New("failed to write environment datastore file")
	}

	return nil
}

func (d EnvDir) writeAddressRefs(ds fdatastore.MutableDataStore) error {
	dataStoreConcrete, ok := ds.(*fdatastore.MemoryDataStore)
	if !ok {
		return errors.New("failed to cast dataStore to concrete type MemoryDataStore")
	}

	err := jsonutils.WriteFile(d.AddressRefsFilePath(), dataStoreConcrete.AddressRefStore.Records)
	if err != nil {
		return errors.New("failed to write address refs store file")
	}

	return nil
}

func addressRefIdentityFromRef(ref fdatastore.AddressRef) addressRefIdentity {
	version := ""
	if ref.Version != nil {
		version = ref.Version.String()
	}

	return addressRefIdentity{
		chainSelector: ref.ChainSelector,
		address:       strings.ToLower(ref.Address),
		contractType:  ref.Type,
		version:       version,
	}
}

func addressRefIdentityFromAddressBook(
	chainSelector uint64,
	addr string,
	typever fdeployment.TypeAndVersion,
) addressRefIdentity {
	return addressRefIdentity{
		chainSelector: chainSelector,
		address:       strings.ToLower(addr),
		contractType:  fdatastore.ContractType(typever.Type),
		version:       typever.Version.String(),
	}
}

func loadAddressBookByChangesetKey(artDir *ArtifactsDir, csKey, timestamp string) (fdeployment.AddressBook, error) {
	// Set the durable pipelines directory and timestamp if provided
	if timestamp != "" {
		if err := artDir.SetDurablePipelines(timestamp); err != nil {
			return &fdeployment.AddressBookMap{}, err
		}
	}

	// Load the changeset address book where the artifacts group name is the changeset key
	csAddrBook, err := artDir.LoadAddressBookByChangesetKey(csKey)
	if err != nil {
		if errors.Is(err, ErrArtifactNotFound) {
			fmt.Println("No changeset address book found, skipping merge")

			return &fdeployment.AddressBookMap{}, nil
		}

		return &fdeployment.AddressBookMap{}, err
	}

	return csAddrBook, nil
}

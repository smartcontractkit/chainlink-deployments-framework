package datastore

import "fmt"

// MetadataBundle is address refs plus contract, chain, and env metadata.
type MetadataBundle struct {
	// Addresses are the contract addresses that were deployed.
	Addresses []AddressRef
	// Contracts defines any metadata pertaining to contracts that were deployed.
	Contracts []ContractMetadata
	// Chains defines metadata pertaining to chains that were operated against.
	Chains []ChainMetadata
	// Env defines any metadata pertaining to the environment that was deployed to.
	Env *EnvMetadata
}

type addressRefWriteMode int

const (
	addressRefWriteAdd addressRefWriteMode = iota
	addressRefWriteUpsert
)

type writeMetadataConfig struct {
	addressRefMode addressRefWriteMode
}

// WriteMetadataOption configures WriteMetadataToDatastore.
type WriteMetadataOption func(*writeMetadataConfig)

// WithUpsertAddressRefs writes address refs with Upsert instead of the default Add.
// Upsert replaces the full record for an existing key (chain, type, version, qualifier),
// including Address.
func WithUpsertAddressRefs() WriteMetadataOption {
	return func(cfg *writeMetadataConfig) {
		cfg.addressRefMode = addressRefWriteUpsert
	}
}

func applyWriteMetadataOptions(opts []WriteMetadataOption) writeMetadataConfig {
	cfg := writeMetadataConfig{addressRefMode: addressRefWriteAdd}
	for _, opt := range opts {
		opt(&cfg)
	}

	return cfg
}

// WriteMetadataToDataStore writes address refs and upserts contract and chain metadata and sets env metadata.
// Address refs use Add by default; pass WithUpsertAddressRefs to insert or replace by key.
func WriteMetadataToDataStore(ds MutableDataStore, bundle MetadataBundle, opts ...WriteMetadataOption) error {
	cfg := applyWriteMetadataOptions(opts)

	for _, ref := range bundle.Addresses {
		var err error
		switch cfg.addressRefMode {
		case addressRefWriteAdd:
			err = ds.Addresses().Add(ref)
		case addressRefWriteUpsert:
			err = ds.Addresses().Upsert(ref)
		default:
			err = fmt.Errorf("unknown address ref write mode: %d", cfg.addressRefMode)
		}
		if err != nil {
			verb := "add"
			if cfg.addressRefMode == addressRefWriteUpsert {
				verb = "upsert"
			}

			return fmt.Errorf("failed to %s %s %v at %s on chain %d to datastore: %w",
				verb, ref.Type, ref.Version, ref.Address, ref.ChainSelector, err)
		}
	}
	for _, contract := range bundle.Contracts {
		if err := ds.ContractMetadata().Upsert(contract); err != nil {
			return fmt.Errorf("failed to upsert contract metadata for %s on chain %d to datastore: %w",
				contract.Address, contract.ChainSelector, err)
		}
	}
	for _, chain := range bundle.Chains {
		if err := ds.ChainMetadata().Upsert(chain); err != nil {
			return fmt.Errorf("failed to upsert chain metadata for chain %d to datastore: %w",
				chain.ChainSelector, err)
		}
	}
	if bundle.Env != nil {
		if err := ds.EnvMetadata().Set(*bundle.Env); err != nil {
			return fmt.Errorf("failed to set env metadata to datastore: %w", err)
		}
	}

	return nil
}

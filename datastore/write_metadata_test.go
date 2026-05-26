package datastore

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/require"
)

func TestWriteMetadataToDataStore(t *testing.T) {
	t.Parallel()

	contractOne := ContractMetadata{
		Address:       "0xaaa",
		ChainSelector: 1,
		Metadata:      testMetadata{Field: "contract-one", ChainSelector: 1},
	}
	contractTwo := ContractMetadata{
		Address:       "0xbbb",
		ChainSelector: 2,
		Metadata:      testMetadata{Field: "contract-two", ChainSelector: 2},
	}
	chainMD := ChainMetadata{
		ChainSelector: 1,
		Metadata:      testMetadata{Field: "chain", ChainSelector: 1},
	}
	envMD := EnvMetadata{
		Metadata: testMetadata{Field: "env", ChainSelector: 0},
	}

	t.Run("via mutable interface", func(t *testing.T) {
		t.Parallel()
		var ds MutableDataStore = NewMemoryDataStore()
		require.NoError(t, WriteMetadataToDataStore(ds, MetadataBundle{}))
	})

	t.Run("writes addresses and contract metadata", func(t *testing.T) {
		t.Parallel()
		ds := NewMemoryDataStore()
		ref := AddressRef{
			Address:       "0xabc",
			ChainSelector: 1,
			Type:          "Timelock",
			Version:       semver.MustParse("1.0.0"),
		}
		contractMD := ContractMetadata{
			Address:       "0xabc",
			ChainSelector: 1,
			Metadata:      testMetadata{Field: "contract", ChainSelector: 1},
		}

		require.NoError(t, WriteMetadataToDataStore(ds, MetadataBundle{
			Addresses: []AddressRef{ref},
			Contracts: []ContractMetadata{contractMD},
		}))

		refs, err := ds.Addresses().Fetch()
		require.NoError(t, err)
		require.Len(t, refs, 1)

		contracts, err := ds.ContractMetadata().Fetch()
		require.NoError(t, err)
		require.Len(t, contracts, 1)

		err = WriteMetadataToDataStore(ds, MetadataBundle{Addresses: []AddressRef{ref}})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrAddressRefExists)
	})

	t.Run("memory datastore convenience method", func(t *testing.T) {
		t.Parallel()
		ds := NewMemoryDataStore()
		require.NoError(t, ds.WriteMetadata(MetadataBundle{}))
	})

	t.Run("empty bundle is a no-op", func(t *testing.T) {
		t.Parallel()
		ds := NewMemoryDataStore()
		require.NoError(t, WriteMetadataToDataStore(ds, MetadataBundle{}))

		contracts, err := ds.ContractMetadata().Fetch()
		require.NoError(t, err)
		require.Empty(t, contracts)

		chains, err := ds.ChainMetadata().Fetch()
		require.NoError(t, err)
		require.Empty(t, chains)

		_, err = ds.EnvMetadata().Get()
		require.ErrorIs(t, err, ErrEnvMetadataNotSet)
	})

	t.Run("writes multiple chain metadata entries", func(t *testing.T) {
		t.Parallel()
		ds := NewMemoryDataStore()
		chainTwo := ChainMetadata{
			ChainSelector: 2,
			Metadata:      testMetadata{Field: "chain-two", ChainSelector: 2},
		}

		require.NoError(t, WriteMetadataToDataStore(ds, MetadataBundle{
			Chains: []ChainMetadata{chainMD, chainTwo},
		}))

		chains, err := ds.ChainMetadata().Fetch()
		require.NoError(t, err)
		require.Len(t, chains, 2)
	})

	t.Run("writes contracts chain and env", func(t *testing.T) {
		t.Parallel()
		ds := NewMemoryDataStore()
		require.NoError(t, WriteMetadataToDataStore(ds, MetadataBundle{
			Contracts: []ContractMetadata{contractOne, contractTwo},
			Chains:    []ChainMetadata{chainMD},
			Env:       &envMD,
		}))

		contracts, err := ds.ContractMetadata().Fetch()
		require.NoError(t, err)
		require.Len(t, contracts, 2)

		chains, err := ds.ChainMetadata().Fetch()
		require.NoError(t, err)
		require.Len(t, chains, 1)
		require.Equal(t, uint64(1), chains[0].ChainSelector)

		gotEnv, err := ds.EnvMetadata().Get()
		require.NoError(t, err)
		gotEnvMD, err := As[testMetadata](gotEnv.Metadata)
		require.NoError(t, err)
		require.Equal(t, "env", gotEnvMD.Field)
	})

	t.Run("WithUpsertAddressRefs overwrites existing address ref", func(t *testing.T) {
		t.Parallel()
		ds := NewMemoryDataStore()
		ref := AddressRef{
			Address:       "0xabc",
			ChainSelector: 1,
			Type:          "Timelock",
			Version:       semver.MustParse("1.0.0"),
		}
		updatedRef := ref
		updatedRef.Address = "0xdef"

		require.NoError(t, WriteMetadataToDataStore(ds, MetadataBundle{Addresses: []AddressRef{ref}}))
		require.NoError(t, WriteMetadataToDataStore(ds, MetadataBundle{Addresses: []AddressRef{updatedRef}}, WithUpsertAddressRefs()))

		refs, err := ds.Addresses().Fetch()
		require.NoError(t, err)
		require.Len(t, refs, 1)
		require.Equal(t, "0xdef", refs[0].Address)
	})

	t.Run("upserts overwrite existing records", func(t *testing.T) {
		t.Parallel()
		ds := NewMemoryDataStore()
		require.NoError(t, WriteMetadataToDataStore(ds, MetadataBundle{
			Contracts: []ContractMetadata{contractOne},
			Chains:    []ChainMetadata{chainMD},
			Env:       &envMD,
		}))

		updatedContract := contractOne
		updatedContract.Metadata = testMetadata{Field: "updated", ChainSelector: 1}
		updatedChain := ChainMetadata{
			ChainSelector: 1,
			Metadata:      testMetadata{Field: "updated-chain", ChainSelector: 1},
		}
		updatedEnv := EnvMetadata{
			Metadata: testMetadata{Field: "updated-env", ChainSelector: 0},
		}

		require.NoError(t, WriteMetadataToDataStore(ds, MetadataBundle{
			Contracts: []ContractMetadata{updatedContract},
			Chains:    []ChainMetadata{updatedChain},
			Env:       &updatedEnv,
		}))

		contracts, err := ds.ContractMetadata().Fetch()
		require.NoError(t, err)
		require.Len(t, contracts, 1)
		gotContractMD, err := As[testMetadata](contracts[0].Metadata)
		require.NoError(t, err)
		require.Equal(t, "updated", gotContractMD.Field)

		chains, err := ds.ChainMetadata().Fetch()
		require.NoError(t, err)
		require.Len(t, chains, 1)
		gotChainMD, err := As[testMetadata](chains[0].Metadata)
		require.NoError(t, err)
		require.Equal(t, "updated-chain", gotChainMD.Field)

		gotEnv, err := ds.EnvMetadata().Get()
		require.NoError(t, err)
		gotEnvMD, err := As[testMetadata](gotEnv.Metadata)
		require.NoError(t, err)
		require.Equal(t, "updated-env", gotEnvMD.Field)
	})
}

package datastore

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/require"
)

func TestMemoryDataStore_Merge(t *testing.T) {
	t.Parallel()

	var (
		addressRefRecord = AddressRef{
			Address:   "0x123",
			Type:      "type1",
			Version:   semver.MustParse("1.0.0"),
			Qualifier: "qualifier1",
		}
		chainMetadataRecord = ChainMetadata{
			ChainSelector: 1,
			Metadata: testMetadata{
				Field:         "test field",
				ChainSelector: 1,
			},
		}
		contractMetadataRecord = ContractMetadata{
			Address:       "0x123",
			ChainSelector: 2,
			Metadata: testMetadata{
				Field:         "another test field",
				ChainSelector: 2,
			},
		}
		envMetadataRecord = EnvMetadata{
			Metadata: testMetadata{
				Field:         "env test field",
				ChainSelector: 1,
			},
		}
	)

	tests := []struct {
		name                          string
		setup                         func() (*MemoryDataStore, *MemoryDataStore)
		expectedAddrRefsCount         int
		excpectedChainMetadataCount   int
		expectedContractMetadataCount int
		expectedError                 error
		expectedAddressDRK            []string
		expectedChainMetaDRK          []string
		expectedContractMetaDRK       []string
	}{
		{
			name: "Merge single address",
			setup: func() (*MemoryDataStore, *MemoryDataStore) {
				dataStore1 := NewMemoryDataStore()
				dataStore2 := NewMemoryDataStore()
				err := dataStore2.Addresses().Add(addressRefRecord)
				require.NoError(t, err, "Adding data to dataStore2 should not fail")

				err = dataStore2.ChainMetadata().Add(chainMetadataRecord)
				require.NoError(t, err, "Adding chain metadata to dataStore2 should not fail")

				err = dataStore2.ContractMetadata().Add(contractMetadataRecord)
				require.NoError(t, err, "Adding another chain metadata to dataStore2 should not fail")

				err = dataStore2.EnvMetadata().Set(envMetadataRecord)
				require.NoError(t, err, "Adding env metadata to dataStore2 should not fail")

				return dataStore1, dataStore2
			},
			expectedAddrRefsCount:         1,
			excpectedChainMetadataCount:   1,
			expectedContractMetadataCount: 1,
		},
		{
			name: "Merge propagate deletions: stages delete for address ref record not present in destination",
			setup: func() (*MemoryDataStore, *MemoryDataStore) {
				dataStore1 := NewMemoryDataStore()
				dataStore2 := NewMemoryDataStore()

				err := dataStore1.Addresses().Add(addressRefRecord)
				require.NoError(t, err, "Adding data to dataStore1 should not fail")

				err = dataStore1.ChainMetadata().Add(chainMetadataRecord)
				require.NoError(t, err, "Adding chain metadata to dataStore1 should not fail")

				err = dataStore1.ContractMetadata().Add(contractMetadataRecord)
				require.NoError(t, err, "Adding another chain metadata to dataStore1 should not fail")

				err = dataStore1.EnvMetadata().Set(envMetadataRecord)
				require.NoError(t, err, "Adding env metadata to dataStore1 should not fail")

				// dataStore2 stages a record for deletion that does not exist in dataStore1
				stagedKey := NewAddressRefKey(0, "typeA", semver.MustParse("1.0.0"), "q")
				require.NoError(t, dataStore2.Addresses().RemoteDelete(stagedKey))

				return dataStore1, dataStore2
			},
			expectedAddrRefsCount:         1,
			excpectedChainMetadataCount:   1,
			expectedContractMetadataCount: 1,
			expectedAddressDRK:            []string{NewAddressRefKey(0, "typeA", semver.MustParse("1.0.0"), "q").String()},
		},
		{
			name: "Merge propagate deletions: stages delete for chain metadata record not present in destination",
			setup: func() (*MemoryDataStore, *MemoryDataStore) {
				dataStore1 := NewMemoryDataStore()
				dataStore2 := NewMemoryDataStore()

				err := dataStore1.Addresses().Add(addressRefRecord)
				require.NoError(t, err, "Adding data to dataStore2 should not fail")

				err = dataStore1.ChainMetadata().Add(chainMetadataRecord)
				require.NoError(t, err, "Adding chain metadata to dataStore2 should not fail")

				err = dataStore1.ContractMetadata().Add(contractMetadataRecord)
				require.NoError(t, err, "Adding another chain metadata to dataStore2 should not fail")

				err = dataStore1.EnvMetadata().Set(envMetadataRecord)
				require.NoError(t, err, "Adding env metadata to dataStore2 should not fail")

				// dataStore2 stages a record for deletion that does not exist in dataStore1
				stagedKey := NewChainMetadataKey(10)
				require.NoError(t, dataStore2.ChainMetadata().RemoteDelete(stagedKey))

				return dataStore1, dataStore2
			},
			expectedAddrRefsCount:         1,
			excpectedChainMetadataCount:   1,
			expectedContractMetadataCount: 1,
			expectedChainMetaDRK:          []string{NewChainMetadataKey(10).String()},
		},
		{
			name: "Merge propagate deletions: stages delete for contract metadata record not present in destination",
			setup: func() (*MemoryDataStore, *MemoryDataStore) {
				dataStore1 := NewMemoryDataStore()
				dataStore2 := NewMemoryDataStore()

				err := dataStore1.Addresses().Add(addressRefRecord)
				require.NoError(t, err, "Adding data to dataStore2 should not fail")

				err = dataStore1.ChainMetadata().Add(chainMetadataRecord)
				require.NoError(t, err, "Adding chain metadata to dataStore2 should not fail")

				err = dataStore1.ContractMetadata().Add(contractMetadataRecord)
				require.NoError(t, err, "Adding another chain metadata to dataStore2 should not fail")

				err = dataStore1.EnvMetadata().Set(envMetadataRecord)
				require.NoError(t, err, "Adding env metadata to dataStore2 should not fail")

				// dataStore2 stages a record for deletion that does not exist in dataStore1
				stagedKey := NewContractMetadataKey(10, "0x111")
				require.NoError(t, dataStore2.ContractMetadata().RemoteDelete(stagedKey))

				return dataStore1, dataStore2
			},
			expectedAddrRefsCount:         1,
			excpectedChainMetadataCount:   1,
			expectedContractMetadataCount: 1,
			expectedContractMetaDRK:       []string{NewContractMetadataKey(10, "0x111").String()},
		},
		{
			name: "Merge propagate deletions: deletes record from remote data store",
			setup: func() (*MemoryDataStore, *MemoryDataStore) {
				dataStore1 := NewMemoryDataStore()
				dataStore2 := NewMemoryDataStore()

				err := dataStore1.Addresses().Add(addressRefRecord)
				require.NoError(t, err, "Adding data to dataStore2 should not fail")

				err = dataStore1.ChainMetadata().Add(chainMetadataRecord)
				require.NoError(t, err, "Adding chain metadata to dataStore2 should not fail")

				err = dataStore1.ContractMetadata().Add(contractMetadataRecord)
				require.NoError(t, err, "Adding another chain metadata to dataStore2 should not fail")

				err = dataStore1.EnvMetadata().Set(envMetadataRecord)
				require.NoError(t, err, "Adding env metadata to dataStore2 should not fail")

				// dataStore2 stages a record for deletion that does not exist in dataStore1
				require.NoError(t, dataStore2.Addresses().RemoteDelete(addressRefRecord.Key()))
				require.NoError(t, dataStore2.ChainMetadata().RemoteDelete(chainMetadataRecord.Key()))
				require.NoError(t, dataStore2.ContractMetadata().RemoteDelete(contractMetadataRecord.Key()))

				return dataStore1, dataStore2
			},
			expectedAddrRefsCount:         0,
			excpectedChainMetadataCount:   0,
			expectedContractMetadataCount: 0,
			expectedAddressDRK:            []string{addressRefRecord.Key().String()},
			expectedChainMetaDRK:          []string{chainMetadataRecord.Key().String()},
			expectedContractMetaDRK:       []string{contractMetadataRecord.Key().String()},
		},
		{
			name: "Match existing address with labels",
			setup: func() (*MemoryDataStore, *MemoryDataStore) {
				dataStore1 := NewMemoryDataStore()
				dataStore2 := NewMemoryDataStore()

				// Add initial data to dataStore1
				err := dataStore1.Addresses().Add(AddressRef{
					Address:   "0x123",
					Type:      "type1",
					Version:   semver.MustParse("1.0.0"),
					Qualifier: "qualifier1",
					Labels:    NewLabelSet("label1"),
				})
				require.NoError(t, err, "Adding initial data to dataStore1 should not fail")

				err = dataStore1.ChainMetadata().Add(ChainMetadata{
					ChainSelector: 1,
					Metadata: testMetadata{
						Field:         "test field",
						ChainSelector: 1,
					},
				})
				require.NoError(t, err, "Adding chain metadata to dataStore1 should not fail")

				err = dataStore1.ContractMetadata().Add(ContractMetadata{
					Address:       "0x123",
					ChainSelector: 2,
					Metadata: testMetadata{
						Field:         "another test field",
						ChainSelector: 2,
					},
				})
				require.NoError(t, err, "Adding contract metadata to dataStore1 should not fail")

				err = dataStore1.EnvMetadata().Set(EnvMetadata{
					Metadata: testMetadata{
						Field:         "env test field",
						ChainSelector: 1,
					},
				})
				require.NoError(t, err, "Adding env metadata to dataStore1 should not fail")

				// Add matching data to dataStore2
				err = dataStore2.Addresses().Add(AddressRef{
					Address:   "0x123",
					Type:      "type1",
					Version:   semver.MustParse("1.0.0"),
					Qualifier: "qualifier1",
					Labels:    NewLabelSet("label2"),
				})
				require.NoError(t, err, "Adding matching data to dataStore2 should not fail")

				return dataStore1, dataStore2
			},
			expectedAddrRefsCount:         1,
			excpectedChainMetadataCount:   1,
			expectedContractMetadataCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dataStore1, dataStore2 := tt.setup()

			// Merge dataStore2 into dataStore1
			if tt.expectedError != nil {
				err := dataStore1.Merge(dataStore2.Seal())
				require.ErrorIs(t, err, tt.expectedError, "Merging dataStore2 into dataStore1 should return the expected error")
			} else {
				err := dataStore1.Merge(dataStore2.Seal())
				require.NoError(t, err, "Merging dataStore2 into dataStore1 should not fail")
			}

			// Verify that dataStore1 contains the merged data
			addressRefs, err := dataStore1.Addresses().Fetch()
			require.NoError(t, err, "Fetching addresses from dataStore1 should not fail")
			require.Len(t, addressRefs, tt.expectedAddrRefsCount, "dataStore1 should contain the expected number of addresses after merge")

			chainMetadata, err := dataStore1.ChainMetadata().Fetch()
			require.NoError(t, err, "Fetching chain metadata from dataStore1 should not fail")
			require.Len(t, chainMetadata, tt.excpectedChainMetadataCount, "dataStore1 should contain the expected number of chain metadata after merge")

			contractMetadata, err := dataStore1.ContractMetadata().Fetch()
			require.NoError(t, err, "Fetching contract metadata from dataStore1 should not fail")
			require.Len(t, contractMetadata, tt.expectedContractMetadataCount, "dataStore1 should contain the expected number of contract metadata after merge")

			_, err = dataStore1.EnvMetadata().Get()
			require.NoError(t, err, "Fetching env metadata from dataStore1 should not fail")

			// Verify staged deletes were propagated to DeletedRemoteKeys
			if len(tt.expectedAddressDRK) > 0 {
				require.ElementsMatch(t, tt.expectedAddressDRK, dataStore1.AddressRefStore.DeletedRemoteKeys,
					"dataStore1 AddressRefStore.DeletedRemoteKeys should contain the expected staged keys after merge")
			}
			if len(tt.expectedChainMetaDRK) > 0 {
				require.ElementsMatch(t, tt.expectedChainMetaDRK, dataStore1.ChainMetadataStore.DeletedRemoteKeys,
					"dataStore1 ChainMetadataStore.DeletedRemoteKeys should contain the expected staged keys after merge")
			}
			if len(tt.expectedContractMetaDRK) > 0 {
				require.ElementsMatch(t, tt.expectedContractMetaDRK, dataStore1.ContractMetadataStore.DeletedRemoteKeys,
					"dataStore1 ContractMetadataStore.DeletedRemoteKeys should contain the expected staged keys after merge")
			}
		})
	}
}

func TestMemoryDataStore_Merge_ChainedComposition(t *testing.T) {
	t.Parallel()

	recA := AddressRef{
		Address:   "0xAAA",
		Type:      "typeA",
		Version:   semver.MustParse("2.0.0"),
		Qualifier: "qa",
	}

	t.Run("Chained merge preserves DRK across two hops", func(t *testing.T) {
		t.Parallel()

		dataStore1 := NewMemoryDataStore()
		require.NoError(t, dataStore1.Addresses().Add(recA))

		dataStore2 := NewMemoryDataStore()
		require.NoError(t, dataStore2.Addresses().RemoteDelete(recA.Key()))

		dataStore3 := NewMemoryDataStore()
		require.NoError(t, dataStore3.Merge(dataStore2.Seal()))

		dataStore4 := NewMemoryDataStore()
		require.NoError(t, dataStore4.Merge(dataStore3.Seal()))

		require.Contains(t, dataStore4.AddressRefStore.DeletedRemoteKeys, recA.Key().String(),
			"DRK should survive two merge hops")

		addressRefs, err := dataStore4.Addresses().Fetch()
		require.NoError(t, err)
		require.Empty(t, addressRefs, "recA should not appear in dataStore4 records after chained deletes")
	})

	t.Run("Merge is idempotent on staged deletes", func(t *testing.T) {
		t.Parallel()

		dataStore1 := NewMemoryDataStore()
		require.NoError(t, dataStore1.Addresses().Add(recA))

		dataStore2 := NewMemoryDataStore()
		require.NoError(t, dataStore2.Addresses().RemoteDelete(recA.Key()))

		sealed := dataStore2.Seal()

		require.NoError(t, dataStore1.Merge(sealed), "first merge should succeed")
		require.NoError(t, dataStore1.Merge(sealed), "second merge should succeed (idempotent)")

		addressRefs, err := dataStore1.Addresses().Fetch()
		require.NoError(t, err)
		require.Empty(t, addressRefs, "recA should not be present after idempotent merges")

		require.Contains(t, dataStore1.AddressRefStore.DeletedRemoteKeys, recA.Key().String(),
			"DRK should be present after idempotent merges")
	})
}

// TestMemoryDataStore_Merge_SourceLiveRecordClearsDestStagedDelete pins the
// precedence rule documented on MemoryDataStore.Merge: when src.<Store>.Records
// contains a live record for key K and src.<Store>.DeletedRemoteKeys does NOT
// contain K, Merge's upsert clears K from dst.<Store>.DeletedRemoteKeys.
//
// In other words, a live record on the source side overrides a staged delete on
// the destination side for the same key. Staged deletes are sticky only on the
// source side of Merge.
//
// If this test starts failing, the precedence note on MemoryDataStore.Merge is
// out of date — update either the assertion or the docstring to keep them in sync.
func TestMemoryDataStore_Merge_SourceLiveRecordClearsDestStagedDelete(t *testing.T) {
	t.Parallel()

	rec := AddressRef{
		Address:   "0xBEEF",
		Type:      "typeP",
		Version:   semver.MustParse("1.0.0"),
		Qualifier: "qP",
	}

	dst := NewMemoryDataStore()
	require.NoError(t, dst.Addresses().RemoteDelete(rec.Key()))
	require.Contains(t, dst.AddressRefStore.DeletedRemoteKeys, rec.Key().String(),
		"precondition: dst.DRK should contain the staged key before Merge")

	src := NewMemoryDataStore()
	require.NoError(t, src.Addresses().Add(rec))

	require.NoError(t, dst.Merge(src.Seal()))

	addressRefs, err := dst.Addresses().Fetch()
	require.NoError(t, err)
	require.Len(t, addressRefs, 1, "dst should now contain src's live record")

	require.NotContains(t, dst.AddressRefStore.DeletedRemoteKeys, rec.Key().String(),
		"dst.DRK should no longer contain the previously-staged key — Upsert cleared it")
}

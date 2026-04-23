package datastore

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/require"
)

func TestMemoryDataStore_Merge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                          string
		setup                         func() (*MemoryDataStore, *MemoryDataStore)
		expectedAddrRefsCount         int
		excpectedChainMetadataCount   int
		expectedContractMetadataCount int
	}{
		{
			name: "Merge single address",
			setup: func() (*MemoryDataStore, *MemoryDataStore) {
				dataStore1 := NewMemoryDataStore()
				dataStore2 := NewMemoryDataStore()
				err := dataStore2.Addresses().Add(AddressRef{
					Address:   "0x123",
					Type:      "type1",
					Version:   semver.MustParse("1.0.0"),
					Qualifier: "qualifier1",
				})
				require.NoError(t, err, "Adding data to dataStore2 should not fail")

				err = dataStore2.ChainMetadata().Add(ChainMetadata{
					ChainSelector: 1,
					Metadata: testMetadata{
						Field:         "test field",
						ChainSelector: 1,
					},
				})
				require.NoError(t, err, "Adding chain metadata to dataStore2 should not fail")

				err = dataStore2.ContractMetadata().Add(ContractMetadata{
					Address:       "0x123",
					ChainSelector: 2,
					Metadata: testMetadata{
						Field:         "another test field",
						ChainSelector: 2,
					},
				})
				require.NoError(t, err, "Adding another chain metadata to dataStore2 should not fail")

				err = dataStore2.EnvMetadata().Set(EnvMetadata{
					Metadata: testMetadata{
						Field:         "env test field",
						ChainSelector: 1,
					},
				})
				require.NoError(t, err, "Adding env metadata to dataStore2 should not fail")

				return dataStore1, dataStore2
			},
			expectedAddrRefsCount:         1,
			excpectedChainMetadataCount:   1,
			expectedContractMetadataCount: 1,
		},
		{
			name: "Merge propagates deletions: delete wins over existing records",
			setup: func() (*MemoryDataStore, *MemoryDataStore) {
				dataStore1 := NewMemoryDataStore()
				dataStore2 := NewMemoryDataStore()

				// dataStore1 has records A and B for each store
				require.NoError(t, dataStore1.Addresses().Add(AddressRef{
					Address: "0x111", Type: "typeA", Version: semver.MustParse("1.0.0"), Qualifier: "q",
				}))
				require.NoError(t, dataStore1.Addresses().Add(AddressRef{
					Address: "0x222", Type: "typeB", Version: semver.MustParse("1.0.0"), Qualifier: "q",
				}))
				require.NoError(t, dataStore1.ChainMetadata().Add(ChainMetadata{
					ChainSelector: 10, Metadata: testMetadata{Field: "a"},
				}))
				require.NoError(t, dataStore1.ChainMetadata().Add(ChainMetadata{
					ChainSelector: 20, Metadata: testMetadata{Field: "b"},
				}))
				require.NoError(t, dataStore1.ContractMetadata().Add(ContractMetadata{
					ChainSelector: 10, Address: "0x111", Metadata: testMetadata{Field: "a"},
				}))
				require.NoError(t, dataStore1.ContractMetadata().Add(ContractMetadata{
					ChainSelector: 20, Address: "0x222", Metadata: testMetadata{Field: "b"},
				}))
				require.NoError(t, dataStore1.EnvMetadata().Set(EnvMetadata{
					Metadata: testMetadata{Field: "env"},
				}))

				// dataStore2 deletes key A from each store (key A is not in dataStore2's Records)
				require.NoError(t, dataStore2.Addresses().Delete(NewAddressRefKey(0, "typeA", semver.MustParse("1.0.0"), "q")))
				require.NoError(t, dataStore2.ChainMetadata().Delete(NewChainMetadataKey(10)))
				require.NoError(t, dataStore2.ContractMetadata().Delete(NewContractMetadataKey(10, "0x111")))

				return dataStore1, dataStore2
			},
			expectedAddrRefsCount:         1,
			excpectedChainMetadataCount:   1,
			expectedContractMetadataCount: 1,
		},
		{
			name: "Merge propagates deletions: delete of non-existent key is a no-op",
			setup: func() (*MemoryDataStore, *MemoryDataStore) {
				dataStore1 := NewMemoryDataStore()
				dataStore2 := NewMemoryDataStore()

				// dataStore1 is empty; dataStore2 deletes a key that was never in dataStore1
				require.NoError(t, dataStore2.Addresses().Delete(NewAddressRefKey(0, "typeA", semver.MustParse("1.0.0"), "q")))
				require.NoError(t, dataStore2.ChainMetadata().Delete(NewChainMetadataKey(10)))
				require.NoError(t, dataStore2.ContractMetadata().Delete(NewContractMetadataKey(10, "0x111")))

				require.NoError(t, dataStore1.EnvMetadata().Set(EnvMetadata{
					Metadata: testMetadata{Field: "env"},
				}))

				return dataStore1, dataStore2
			},
			expectedAddrRefsCount:         0,
			excpectedChainMetadataCount:   0,
			expectedContractMetadataCount: 0,
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
			err := dataStore1.Merge(dataStore2.Seal())
			require.NoError(t, err, "Merging dataStore2 into dataStore1 should not fail")

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
		})
	}
}

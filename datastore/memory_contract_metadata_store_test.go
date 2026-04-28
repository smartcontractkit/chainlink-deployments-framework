package datastore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chainsel "github.com/smartcontractkit/chain-selectors"
)

func TestMemoryContractMetadataStore_indexOf(t *testing.T) {
	t.Parallel()

	var (
		recordOne = ContractMetadata{
			ChainSelector: 1,
			Address:       "0x2324224",
			Metadata:      testMetadata{Field: "metadata1", ChainSelector: 0},
		}

		recordTwo = ContractMetadata{
			ChainSelector: 2,
			Address:       "0x2324224",
			Metadata:      testMetadata{Field: "metadata2", ChainSelector: 0},
		}
	)

	tests := []struct {
		name          string
		givenState    []ContractMetadata
		giveKey       ContractMetadataKey
		expectedIndex int
	}{
		{
			name: "success: returns index of record",
			givenState: []ContractMetadata{
				recordOne,
				recordTwo,
			},
			giveKey:       recordTwo.Key(),
			expectedIndex: 1,
		},
		{
			name: "success: returns -1 if record not found",
			givenState: []ContractMetadata{
				recordOne,
			},
			giveKey:       recordTwo.Key(),
			expectedIndex: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryContractMetadataStore{Records: tt.givenState}
			idx := store.indexOf(tt.giveKey)
			require.Equal(t, tt.expectedIndex, idx)
		})
	}
}

func TestMemoryContractMetadataStore_Add(t *testing.T) {
	t.Parallel()

	var (
		record = ContractMetadata{
			ChainSelector: 1,
			Address:       "0x2324224",
			Metadata:      testMetadata{Field: "metadata1", ChainSelector: 0},
		}
	)

	tests := []struct {
		name              string
		givenState        []ContractMetadata
		giveRecord        ContractMetadata
		expectedState     []ContractMetadata
		deletedRemoteKeys []string
		expectedError     error
	}{
		{
			name:       "success: adds new record",
			givenState: []ContractMetadata{},
			giveRecord: record,
			expectedState: []ContractMetadata{
				record,
			},
		},
		{
			name: "error: already existing record",
			givenState: []ContractMetadata{
				record,
			},
			giveRecord:    record,
			expectedError: ErrContractMetadataExists,
		},
		{
			name:              "success: add record that was staged for deletion",
			givenState:        []ContractMetadata{},
			deletedRemoteKeys: []string{record.Key().String()},
			giveRecord:        record,
			expectedState: []ContractMetadata{
				record,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryContractMetadataStore{Records: tt.givenState, DeletedRemoteKeys: tt.deletedRemoteKeys}
			assert.Len(t, store.DeletedRemoteKeys, len(tt.deletedRemoteKeys))
			err := store.Add(tt.giveRecord)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedError, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedState, store.Records)
				assert.Empty(t, store.DeletedRemoteKeys)
			}
		})
	}
}

func TestMemoryContractMetadataStore_Upsert(t *testing.T) {
	t.Parallel()

	var (
		oldRecord = ContractMetadata{
			ChainSelector: 1,
			Address:       "0x2324224",
			Metadata:      testMetadata{Field: "metadata1", ChainSelector: 0},
		}

		newRecord = ContractMetadata{
			ChainSelector: 1,
			Address:       "0x2324224",
			Metadata:      testMetadata{Field: "metadata2", ChainSelector: 0},
		}
	)

	tests := []struct {
		name              string
		givenState        []ContractMetadata
		expectedState     []ContractMetadata
		giveRecord        ContractMetadata
		deletedRemoteKeys []string
	}{
		{
			name:       "success: adds new record",
			givenState: []ContractMetadata{},
			giveRecord: oldRecord,
			expectedState: []ContractMetadata{
				oldRecord,
			},
		},
		{
			name: "success: updates existing record",
			givenState: []ContractMetadata{
				oldRecord,
			},
			giveRecord: newRecord,
			expectedState: []ContractMetadata{
				newRecord,
			},
		},
		{
			name:              "success: upsert record that was staged for deletion",
			givenState:        []ContractMetadata{},
			deletedRemoteKeys: []string{oldRecord.Key().String()},
			giveRecord:        oldRecord,
			expectedState: []ContractMetadata{
				oldRecord,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryContractMetadataStore{Records: tt.givenState, DeletedRemoteKeys: tt.deletedRemoteKeys}
			assert.Len(t, store.DeletedRemoteKeys, len(tt.deletedRemoteKeys))
			// Check the error for the in-memory store, which will always be nil for the
			// in memory implementation, to satisfy the linter
			err := store.Upsert(tt.giveRecord)
			require.NoError(t, err)
			require.Equal(t, tt.expectedState, store.Records)
			assert.Empty(t, store.DeletedRemoteKeys)
		})
	}
}

func TestMemoryContractMetadataStore_Update(t *testing.T) {
	t.Parallel()

	var (
		oldRecord = ContractMetadata{
			ChainSelector: 1,
			Address:       "0x2324224",
			Metadata:      testMetadata{Field: "metadata1", ChainSelector: 0},
		}

		newRecord = ContractMetadata{
			ChainSelector: 1,
			Address:       "0x2324224",
			Metadata:      testMetadata{Field: "metadata2", ChainSelector: 0},
		}
	)

	tests := []struct {
		name              string
		givenState        []ContractMetadata
		expectedState     []ContractMetadata
		giveRecord        ContractMetadata
		expectedError     error
		deletedRemoteKeys []string
	}{
		{
			name: "success: updates existing record",
			givenState: []ContractMetadata{
				oldRecord,
			},
			giveRecord: newRecord,
			expectedState: []ContractMetadata{
				newRecord,
			},
		},
		{
			name:          "error: record not found",
			givenState:    []ContractMetadata{},
			giveRecord:    newRecord,
			expectedError: ErrContractMetadataNotFound,
		},
		{
			name: "success: update record that was staged for deletion",
			givenState: []ContractMetadata{
				oldRecord,
			},
			deletedRemoteKeys: []string{oldRecord.Key().String()},
			giveRecord:        oldRecord,
			expectedState: []ContractMetadata{
				oldRecord,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryContractMetadataStore{Records: tt.givenState, DeletedRemoteKeys: tt.deletedRemoteKeys}
			assert.Len(t, store.DeletedRemoteKeys, len(tt.deletedRemoteKeys))
			err := store.Update(tt.giveRecord)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedError, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedState, store.Records)
				assert.Empty(t, store.DeletedRemoteKeys)
			}
		})
	}
}

func TestMemoryMemoryContractMetadataStore_Delete(t *testing.T) {
	t.Parallel()

	var (
		recordOne = ContractMetadata{
			ChainSelector: 1,
			Address:       "0x2324224",
			Metadata:      testMetadata{Field: "metadata1", ChainSelector: 0},
		}

		recordTwo = ContractMetadata{
			ChainSelector: 2,
			Address:       "0x2324224",
			Metadata:      testMetadata{Field: "metadata2", ChainSelector: 0},
		}

		recordThree = ContractMetadata{
			ChainSelector: 3,
			Address:       "0x2324224",
			Metadata:      testMetadata{Field: "metadata3", ChainSelector: 0},
		}
	)

	t.Run("success: deletes record from Records", func(t *testing.T) {
		t.Parallel()

		store := MemoryContractMetadataStore{Records: []ContractMetadata{recordOne, recordTwo, recordThree}}
		err := store.Delete(recordTwo.Key())
		require.NoError(t, err)
		require.Equal(t, []ContractMetadata{recordOne, recordThree}, store.Records)
		require.Empty(t, store.DeletedRemoteKeys)
	})

	t.Run("error: absent key returns ErrContractMetadataNotFound", func(t *testing.T) {
		t.Parallel()

		store := MemoryContractMetadataStore{Records: []ContractMetadata{recordOne, recordThree}}
		err := store.Delete(recordTwo.Key())
		require.ErrorIs(t, err, ErrContractMetadataNotFound)
		require.Equal(t, []ContractMetadata{recordOne, recordThree}, store.Records)
		require.Empty(t, store.DeletedRemoteKeys)
	})

	t.Run("error: second Delete returns ErrContractMetadataNotFound", func(t *testing.T) {
		t.Parallel()

		store := MemoryContractMetadataStore{Records: []ContractMetadata{recordOne, recordTwo, recordThree}}
		require.NoError(t, store.Delete(recordTwo.Key()))
		require.Len(t, store.Records, 2)
		require.ErrorIs(t, store.Delete(recordTwo.Key()), ErrContractMetadataNotFound)
		require.Len(t, store.Records, 2)
		require.Empty(t, store.DeletedRemoteKeys)
	})
}

func TestMemoryContractMetadataStore_RemoteDelete(t *testing.T) {
	t.Parallel()

	var (
		recordOne = ContractMetadata{
			ChainSelector: 1,
			Address:       "0x2324224",
			Metadata:      testMetadata{Field: "metadata1", ChainSelector: 0},
		}

		recordTwo = ContractMetadata{
			ChainSelector: 2,
			Address:       "0x2324224",
			Metadata:      testMetadata{Field: "metadata2", ChainSelector: 0},
		}

		recordThree = ContractMetadata{
			ChainSelector: 3,
			Address:       "0x2324224",
			Metadata:      testMetadata{Field: "metadata3", ChainSelector: 0},
		}
	)

	t.Run("success: stages key in DeletedRemoteKeys without removing from Records", func(t *testing.T) {
		t.Parallel()

		store := NewMemoryContractMetadataStore()
		store.Records = append(store.Records, recordOne, recordTwo, recordThree)
		require.Len(t, store.Records, 3)
		require.Empty(t, store.DeletedRemoteKeys)
		err := store.RemoteDelete(recordTwo.Key())
		require.NoError(t, err)
		require.Len(t, store.Records, 3)
		require.Len(t, store.DeletedRemoteKeys, 1)
		require.Contains(t, store.DeletedRemoteKeys, recordTwo.Key().String())
	})

	t.Run("success: second RemoteDelete does not duplicate entry", func(t *testing.T) {
		t.Parallel()

		store := NewMemoryContractMetadataStore()
		store.Records = append(store.Records, recordOne, recordTwo, recordThree)
		require.NoError(t, store.RemoteDelete(recordTwo.Key()))
		require.NoError(t, store.RemoteDelete(recordTwo.Key()))
		require.Len(t, store.Records, 3)
		require.Len(t, store.DeletedRemoteKeys, 1)
	})
}

func TestMemoryContractMetadataStore_Fetch(t *testing.T) {
	t.Parallel()

	var (
		recordOne = ContractMetadata{
			ChainSelector: 1,
			Address:       "0x2324224",
			Metadata: testMetadata{
				Field:         "test field",
				ChainSelector: chainsel.APTOS_MAINNET.Selector,
			},
		}

		recordTwo = ContractMetadata{
			ChainSelector: 2,
			Address:       "0x2324224",
			Metadata: testMetadata{
				Field:         "test field 2",
				ChainSelector: chainsel.APTOS_MAINNET.Selector,
			},
		}
	)

	tests := []struct {
		name            string
		givenState      []ContractMetadata
		expectedRecords []ContractMetadata
		expectedError   error
	}{
		{
			name: "success: fetches all records",
			givenState: []ContractMetadata{
				recordOne,
				recordTwo,
			},
			expectedRecords: []ContractMetadata{
				recordOne,
				recordTwo,
			},
			expectedError: nil,
		},
		{
			name:            "success: fetches no records",
			givenState:      []ContractMetadata{},
			expectedRecords: []ContractMetadata{},
			expectedError:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryContractMetadataStore{Records: tt.givenState}
			records, err := store.Fetch()

			if tt.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedError, err)
			} else {
				require.NoError(t, err)
				require.Len(t, records, len(tt.expectedRecords), "Expected number of records to match")

				// Ensure that the records are the same by comparing each field
				for i := range tt.expectedRecords {
					require.Equal(t, tt.expectedRecords[i].ChainSelector, records[i].ChainSelector)
					require.Equal(t, tt.expectedRecords[i].Address, records[i].Address)

					typedMetaActual, err := As[testMetadata](records[i].Metadata)
					require.NoError(t, err)
					require.Equal(t, tt.expectedRecords[i].Metadata, typedMetaActual)
				}
			}
		})
	}
}

func TestMemoryContractMetadataStore_Get(t *testing.T) {
	t.Parallel()

	var (
		recordOne = ContractMetadata{
			ChainSelector: 1,
			Address:       "0x2324224",
			Metadata:      testMetadata{Field: "metadata1", ChainSelector: 0},
		}

		recordTwo = ContractMetadata{
			ChainSelector: 2,
			Address:       "0x2324224",
			Metadata:      testMetadata{Field: "metadata2", ChainSelector: 0},
		}
	)

	tests := []struct {
		name           string
		givenState     []ContractMetadata
		giveKey        ContractMetadataKey
		expectedRecord ContractMetadata
		expectedError  error
	}{
		{
			name: "success: record exists",
			givenState: []ContractMetadata{
				recordOne,
				recordTwo,
			},
			giveKey:        recordTwo.Key(),
			expectedRecord: recordTwo,
		},
		{
			name:           "error: record not found",
			givenState:     []ContractMetadata{},
			giveKey:        recordTwo.Key(),
			expectedRecord: ContractMetadata{},
			expectedError:  ErrContractMetadataNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryContractMetadataStore{Records: tt.givenState}
			record, err := store.Get(tt.giveKey)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedError, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedRecord.ChainSelector, record.ChainSelector)
				require.Equal(t, tt.expectedRecord.Address, record.Address)
				typedMetaActual, err := As[testMetadata](record.Metadata)
				require.NoError(t, err)
				require.Equal(t, tt.expectedRecord.Metadata, typedMetaActual)
			}
		})
	}
}

func TestMemoryContractMetadataStore_Filter(t *testing.T) {
	t.Parallel()

	var (
		recordOne = ContractMetadata{
			ChainSelector: 1,
			Address:       "0x2324224",
			Metadata:      testMetadata{Field: "metadata1", ChainSelector: 0},
		}

		recordTwo = ContractMetadata{
			ChainSelector: 2,
			Address:       "0x2324224",
			Metadata:      testMetadata{Field: "metadata2", ChainSelector: 0},
		}

		recordThree = ContractMetadata{
			ChainSelector: 3,
			Address:       "0x2324224",
			Metadata:      testMetadata{Field: "metadata3", ChainSelector: 0},
		}
	)

	tests := []struct {
		name           string
		givenState     []ContractMetadata
		giveFilters    []FilterFunc[ContractMetadataKey, ContractMetadata]
		expectedResult []ContractMetadata
	}{{
		name: "success: no filters returns all records",
		givenState: []ContractMetadata{
			recordOne,
			recordTwo,
			recordThree,
		},
		giveFilters:    []FilterFunc[ContractMetadataKey, ContractMetadata]{},
		expectedResult: []ContractMetadata{recordOne, recordTwo, recordThree},
	},
		{
			name: "success: returns record with given chain and type",
			givenState: []ContractMetadata{
				recordOne,
				recordTwo,
				recordThree,
			},
			giveFilters: []FilterFunc[ContractMetadataKey, ContractMetadata]{
				ContractMetadataByChainSelector(2),
			},
			expectedResult: []ContractMetadata{recordTwo},
		},
		{
			name: "success: returns no record with given chain and type",
			givenState: []ContractMetadata{
				recordOne,
				recordTwo,
				recordThree,
			},
			giveFilters: []FilterFunc[ContractMetadataKey, ContractMetadata]{
				ContractMetadataByChainSelector(4),
			},
			expectedResult: []ContractMetadata{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryContractMetadataStore{Records: tt.givenState}
			filteredRecords := store.Filter(tt.giveFilters...)
			require.Equal(t, tt.expectedResult, filteredRecords)
		})
	}
}

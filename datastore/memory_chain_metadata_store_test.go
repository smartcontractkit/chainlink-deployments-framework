package datastore

import (
	"testing"

	"github.com/stretchr/testify/require"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
)

func TestMemoryChainMetadataStore_indexOf(t *testing.T) {
	t.Parallel()

	var (
		recordOne = ChainMetadata{
			ChainSelector: 1,
			Metadata:      testMetadata{Field: "metadata1", ChainSelector: 0},
		}

		recordTwo = ChainMetadata{
			ChainSelector: 2,
			Metadata:      testMetadata{Field: "metadata2", ChainSelector: 0},
		}
	)

	tests := []struct {
		name          string
		givenState    []ChainMetadata
		giveKey       ChainMetadataKey
		expectedIndex int
	}{
		{
			name: "success: returns index of record",
			givenState: []ChainMetadata{
				recordOne,
				recordTwo,
			},
			giveKey:       recordTwo.Key(),
			expectedIndex: 1,
		},
		{
			name: "success: returns -1 if record not found",
			givenState: []ChainMetadata{
				recordOne,
			},
			giveKey:       recordTwo.Key(),
			expectedIndex: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryChainMetadataStore{Records: tt.givenState}
			idx := store.indexOf(tt.giveKey)
			require.Equal(t, tt.expectedIndex, idx)
		})
	}
}

func TestMemoryChainMetadataStore_Add(t *testing.T) {
	t.Parallel()

	var (
		record = ChainMetadata{
			ChainSelector: 1,
			Metadata:      testMetadata{Field: "metadata1", ChainSelector: 0},
		}
	)

	tests := []struct {
		name          string
		givenState    []ChainMetadata
		giveRecord    ChainMetadata
		expectedState []ChainMetadata
		expectedError error
	}{
		{
			name:       "success: adds new record",
			givenState: []ChainMetadata{},
			giveRecord: record,
			expectedState: []ChainMetadata{
				record,
			},
		},
		{
			name: "error: already existing record",
			givenState: []ChainMetadata{
				record,
			},
			giveRecord:    record,
			expectedError: ErrChainMetadataExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryChainMetadataStore{Records: tt.givenState}
			err := store.Add(tt.giveRecord)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedError, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedState, store.Records)
			}
		})
	}
}

func TestMemoryChainMetadataStore_Upsert(t *testing.T) {
	t.Parallel()

	var (
		oldRecord = ChainMetadata{
			ChainSelector: 1,
			Metadata:      testMetadata{Field: "metadata1", ChainSelector: 0},
		}

		newRecord = ChainMetadata{
			ChainSelector: 1,
			Metadata:      testMetadata{Field: "metadata2", ChainSelector: 0},
		}
	)

	tests := []struct {
		name          string
		givenState    []ChainMetadata
		expectedState []ChainMetadata
		giveRecord    ChainMetadata
	}{
		{
			name:       "success: adds new record",
			givenState: []ChainMetadata{},
			giveRecord: oldRecord,
			expectedState: []ChainMetadata{
				oldRecord,
			},
		},
		{
			name: "success: updates existing record",
			givenState: []ChainMetadata{
				oldRecord,
			},
			giveRecord: newRecord,
			expectedState: []ChainMetadata{
				newRecord,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryChainMetadataStore{Records: tt.givenState}
			// Check the error for the in-memory store, which will always be nil for the
			// in memory implementation, to satisfy the linter
			err := store.Upsert(tt.giveRecord)
			require.NoError(t, err)
			require.Equal(t, tt.expectedState, store.Records)
		})
	}
}

func TestMemoryChainMetadataStore_Update(t *testing.T) {
	t.Parallel()

	var (
		oldRecord = ChainMetadata{
			ChainSelector: 1,
			Metadata:      testMetadata{Field: "metadata1", ChainSelector: 0},
		}

		newRecord = ChainMetadata{
			ChainSelector: 1,
			Metadata:      testMetadata{Field: "metadata2", ChainSelector: 0},
		}
	)

	tests := []struct {
		name          string
		givenState    []ChainMetadata
		expectedState []ChainMetadata
		giveRecord    ChainMetadata
		expectedError error
	}{
		{
			name: "success: updates existing record",
			givenState: []ChainMetadata{
				oldRecord,
			},
			giveRecord: newRecord,
			expectedState: []ChainMetadata{
				newRecord,
			},
		},
		{
			name:          "error: record not found",
			givenState:    []ChainMetadata{},
			giveRecord:    newRecord,
			expectedError: ErrChainMetadataNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryChainMetadataStore{Records: tt.givenState}
			err := store.Update(tt.giveRecord)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedError, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedState, store.Records)
			}
		})
	}
}

func TestMemoryMemoryChainMetadataStore_Delete(t *testing.T) {
	t.Parallel()

	var (
		recordOne = ChainMetadata{
			ChainSelector: 1,
			Metadata:      testMetadata{Field: "metadata1", ChainSelector: 0},
		}

		recordTwo = ChainMetadata{
			ChainSelector: 2,
			Metadata:      testMetadata{Field: "metadata2", ChainSelector: 0},
		}

		recordThree = ChainMetadata{
			ChainSelector: 3,
			Metadata:      testMetadata{Field: "metadata3", ChainSelector: 0},
		}
	)

	tests := []struct {
		name          string
		givenState    []ChainMetadata
		expectedState []ChainMetadata
		giveKey       ChainMetadataKey
		expectedError error
	}{
		{
			name: "success: deletes given record",
			givenState: []ChainMetadata{
				recordOne,
				recordTwo,
				recordThree,
			},
			giveKey: recordTwo.Key(),
			expectedState: []ChainMetadata{
				recordOne,
				recordThree,
			},
		},
		{
			name: "error: record not found",
			givenState: []ChainMetadata{
				recordOne,
				recordThree,
			},
			giveKey:       recordTwo.Key(),
			expectedError: ErrChainMetadataNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryChainMetadataStore{Records: tt.givenState}
			err := store.Delete(tt.giveKey)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedError, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedState, store.Records)
			}
		})
	}
}

func TestMemoryChainMetadataStore_Fetch(t *testing.T) {
	t.Parallel()

	var (
		recordOne = ChainMetadata{
			ChainSelector: 1,
			Metadata: testMetadata{
				Field:         "test field",
				ChainSelector: chain_selectors.APTOS_MAINNET.Selector,
			},
		}

		recordTwo = ChainMetadata{
			ChainSelector: 2,
			Metadata: testMetadata{
				Field:         "test field 2",
				ChainSelector: chain_selectors.APTOS_MAINNET.Selector,
			},
		}
	)

	tests := []struct {
		name            string
		givenState      []ChainMetadata
		expectedRecords []ChainMetadata
		expectedError   error
	}{
		{
			name: "success: fetches all records",
			givenState: []ChainMetadata{
				recordOne,
				recordTwo,
			},
			expectedRecords: []ChainMetadata{
				recordOne,
				recordTwo,
			},
			expectedError: nil,
		},
		{
			name:            "success: fetches no records",
			givenState:      []ChainMetadata{},
			expectedRecords: []ChainMetadata{},
			expectedError:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryChainMetadataStore{Records: tt.givenState}
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

					typedMetaActual, err := As[testMetadata](records[i].Metadata)
					require.NoError(t, err)
					require.Equal(t, tt.expectedRecords[i].Metadata, typedMetaActual)
				}
			}
		})
	}
}

func TestMemoryChainMetadataStore_Get(t *testing.T) {
	t.Parallel()

	var (
		recordOne = ChainMetadata{
			ChainSelector: 1,
			Metadata:      testMetadata{Field: "metadata1", ChainSelector: 0},
		}

		recordTwo = ChainMetadata{
			ChainSelector: 2,
			Metadata:      testMetadata{Field: "metadata2", ChainSelector: 0},
		}
	)

	tests := []struct {
		name           string
		givenState     []ChainMetadata
		giveKey        ChainMetadataKey
		expectedRecord ChainMetadata
		expectedError  error
	}{
		{
			name: "success: record exists",
			givenState: []ChainMetadata{
				recordOne,
				recordTwo,
			},
			giveKey:        recordTwo.Key(),
			expectedRecord: recordTwo,
		},
		{
			name:           "error: record not found",
			givenState:     []ChainMetadata{},
			giveKey:        recordTwo.Key(),
			expectedRecord: ChainMetadata{},
			expectedError:  ErrChainMetadataNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryChainMetadataStore{Records: tt.givenState}
			record, err := store.Get(tt.giveKey)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedError, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedRecord.ChainSelector, record.ChainSelector)
				typedMetaActual, err := As[testMetadata](record.Metadata)
				require.NoError(t, err)
				require.Equal(t, tt.expectedRecord.Metadata, typedMetaActual)
			}
		})
	}
}

func TestMemoryChainMetadataStore_Filter(t *testing.T) {
	t.Parallel()

	var (
		recordOne = ChainMetadata{
			ChainSelector: 1,
			Metadata:      testMetadata{Field: "metadata1", ChainSelector: 0},
		}

		recordTwo = ChainMetadata{
			ChainSelector: 2,
			Metadata:      testMetadata{Field: "metadata2", ChainSelector: 0},
		}

		recordThree = ChainMetadata{
			ChainSelector: 3,
			Metadata:      testMetadata{Field: "metadata3", ChainSelector: 0},
		}
	)

	tests := []struct {
		name           string
		givenState     []ChainMetadata
		giveFilters    []FilterFunc[ChainMetadataKey, ChainMetadata]
		expectedResult []ChainMetadata
	}{{
		name: "success: no filters returns all records",
		givenState: []ChainMetadata{
			recordOne,
			recordTwo,
			recordThree,
		},
		giveFilters:    []FilterFunc[ChainMetadataKey, ChainMetadata]{},
		expectedResult: []ChainMetadata{recordOne, recordTwo, recordThree},
	},
		{
			name: "success: returns record with given chain and type",
			givenState: []ChainMetadata{
				recordOne,
				recordTwo,
				recordThree,
			},
			giveFilters: []FilterFunc[ChainMetadataKey, ChainMetadata]{
				ChainMetadataByChainSelector(2),
			},
			expectedResult: []ChainMetadata{recordTwo},
		},
		{
			name: "success: returns no record with given chain and type",
			givenState: []ChainMetadata{
				recordOne,
				recordTwo,
				recordThree,
			},
			giveFilters: []FilterFunc[ChainMetadataKey, ChainMetadata]{
				ChainMetadataByChainSelector(4),
			},
			expectedResult: []ChainMetadata{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryChainMetadataStore{Records: tt.givenState}
			filteredRecords := store.Filter(tt.giveFilters...)
			require.Equal(t, tt.expectedResult, filteredRecords)
		})
	}
}

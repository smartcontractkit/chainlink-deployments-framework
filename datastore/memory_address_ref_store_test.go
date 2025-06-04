package datastore

import (
	"encoding/json"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryAddressRefStore_indexOf(t *testing.T) {
	t.Parallel()

	var (
		recordOne = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 1,
			Type:          "type1",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label1", "label2", "label3",
			),
		}

		recordTwo = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 2,
			Type:          "typeX",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label13", "label23", "label33",
			),
		}
	)

	tests := []struct {
		name          string
		givenState    []AddressRef
		giveKey       AddressRefKey
		expectedIndex int
	}{
		{
			name: "success: returns index of record",
			givenState: []AddressRef{
				recordOne,
				recordTwo,
			},
			giveKey:       recordTwo.Key(),
			expectedIndex: 1,
		},
		{
			name: "success: returns -1 if record not found",
			givenState: []AddressRef{
				recordOne,
			},
			giveKey:       recordTwo.Key(),
			expectedIndex: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryAddressRefStore{Records: tt.givenState}
			idx := store.indexOf(tt.giveKey)
			assert.Equal(t, tt.expectedIndex, idx)
		})
	}
}

func TestMemoryAddressRefStore_Add(t *testing.T) {
	t.Parallel()

	var (
		record = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 1,
			Type:          "type1",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label1", "label2", "label3",
			),
		}
	)

	tests := []struct {
		name          string
		givenState    []AddressRef
		giveRecord    AddressRef
		expectedState []AddressRef
		expectedError error
	}{
		{
			name:       "success: adds new record",
			givenState: []AddressRef{},
			giveRecord: record,
			expectedState: []AddressRef{
				record,
			},
		},
		{
			name: "error: already existing record",
			givenState: []AddressRef{
				record,
			},
			giveRecord:    record,
			expectedError: ErrAddressRefExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryAddressRefStore{Records: tt.givenState}
			err := store.Add(tt.giveRecord)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedState, store.Records)
			}
		})
	}
}

func TestMemoryAddressRefStore_Upsert(t *testing.T) {
	t.Parallel()

	var (
		oldRecord = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 1,
			Type:          "type1",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label1", "label2", "label3",
			),
		}
		newRecord = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 1,
			Type:          "type1",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label13", "label23", "label33",
			),
		}
	)

	tests := []struct {
		name          string
		givenState    []AddressRef
		expectedState []AddressRef
		giveRecord    AddressRef
	}{
		{
			name:       "success: adds new record",
			givenState: []AddressRef{},
			giveRecord: oldRecord,
			expectedState: []AddressRef{
				oldRecord,
			},
		},
		{
			name: "success: updates existing record",
			givenState: []AddressRef{
				oldRecord,
			},
			giveRecord: newRecord,
			expectedState: []AddressRef{
				newRecord,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryAddressRefStore{Records: tt.givenState}
			// Check the error, which will always be nil for the
			// in memory implementation, to satisfy the linter
			err := store.Upsert(tt.giveRecord)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedState, store.Records)
		})
	}
}

func TestMemoryAddressRefStore_Update(t *testing.T) {
	t.Parallel()

	var (
		oldRecord = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 1,
			Type:          "type1",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label1", "label2", "label3",
			),
		}
		newRecord = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 1,
			Type:          "type1",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label13", "label23", "label33",
			),
		}
	)

	tests := []struct {
		name          string
		givenState    []AddressRef
		expectedState []AddressRef
		giveRecord    AddressRef
		expectedError error
	}{
		{
			name: "success: updates existing record",
			givenState: []AddressRef{
				oldRecord,
			},
			giveRecord: newRecord,
			expectedState: []AddressRef{
				newRecord,
			},
		},
		{
			name:          "error: record not found",
			givenState:    []AddressRef{},
			giveRecord:    newRecord,
			expectedError: ErrAddressRefNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryAddressRefStore{Records: tt.givenState}
			err := store.Update(tt.giveRecord)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedState, store.Records)
			}
		})
	}
}

func TestMemoryAddressRefStore_Delete(t *testing.T) {
	t.Parallel()

	var (
		recordOne = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 1,
			Type:          "type1",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label1", "label2", "label3",
			),
		}

		recordTwo = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 2,
			Type:          "typeX",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label13", "label23", "label33",
			),
		}

		recordThree = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 3,
			Type:          "typeZ",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label13", "label23", "label33",
			),
		}
	)

	tests := []struct {
		name          string
		givenState    []AddressRef
		expectedState []AddressRef
		giveKey       AddressRefKey
		expectedError error
	}{
		{
			name: "success: deletes given record",
			givenState: []AddressRef{
				recordOne,
				recordTwo,
				recordThree,
			},
			giveKey: recordTwo.Key(),
			expectedState: []AddressRef{
				recordOne,
				recordThree,
			},
		},
		{
			name: "error: record not found",
			givenState: []AddressRef{
				recordOne,
				recordThree,
			},
			giveKey:       recordTwo.Key(),
			expectedError: ErrAddressRefNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryAddressRefStore{Records: tt.givenState}
			err := store.Delete(tt.giveKey)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedState, store.Records)
			}
		})
	}
}

func TestMemoryAddressRefStore_Fetch(t *testing.T) {
	t.Parallel()

	var (
		recordOne = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 1,
			Type:          "type1",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label1", "label2", "label3",
			),
		}

		recordTwo = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 2,
			Type:          "typeX",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label13", "label23", "label33",
			),
		}
	)

	tests := []struct {
		name            string
		givenState      []AddressRef
		expectedRecords []AddressRef
	}{
		{
			name: "success: fetches all records",
			givenState: []AddressRef{
				recordOne,
				recordTwo,
			},
			expectedRecords: []AddressRef{
				recordOne,
				recordTwo,
			},
		},
		{
			name:            "success: fetches no records",
			givenState:      []AddressRef{},
			expectedRecords: []AddressRef{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryAddressRefStore{Records: tt.givenState}
			// Check the error, which will always be nil for the
			// in memory implementation, to satisfy the linter
			records, err := store.Fetch()
			require.NoError(t, err)
			assert.Equal(t, tt.expectedRecords, records)
		})
	}
}

func TestMemoryAddressRefStore_Get(t *testing.T) {
	t.Parallel()

	var (
		recordOne = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 1,
			Type:          "type1",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label1", "label2", "label3",
			),
		}

		recordTwo = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 2,
			Type:          "typeX",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label13", "label23", "label33",
			),
		}
	)

	tests := []struct {
		name           string
		givenState     []AddressRef
		giveKey        AddressRefKey
		expectedRecord AddressRef
		expectedError  error
	}{
		{
			name: "success: record exists",
			givenState: []AddressRef{
				recordOne,
				recordTwo,
			},
			giveKey:        recordTwo.Key(),
			expectedRecord: recordTwo,
			expectedError:  nil,
		},
		{
			name:           "error: record not found",
			givenState:     []AddressRef{},
			giveKey:        recordTwo.Key(),
			expectedRecord: AddressRef{},
			expectedError:  ErrAddressRefNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryAddressRefStore{Records: tt.givenState}
			record, err := store.Get(tt.giveKey)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedRecord, record)
			}
		})
	}
}

func TestMemoryAddressRefStore_Filter(t *testing.T) {
	t.Parallel()

	var (
		recordOne = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 1,
			Type:          "type1",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label1", "label2", "label3",
			),
		}

		recordTwo = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 2,
			Type:          "typeX",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label13", "label23", "label33",
			),
		}

		recordThree = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 3,
			Type:          "typeZ",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label13", "label23", "label33",
			),
		}
	)

	tests := []struct {
		name           string
		givenState     []AddressRef
		giveFilters    []FilterFunc[AddressRefKey, AddressRef]
		expectedResult []AddressRef
	}{
		{
			name: "success: no filters returns all records",
			givenState: []AddressRef{
				recordOne,
				recordTwo,
				recordThree,
			},
			giveFilters:    []FilterFunc[AddressRefKey, AddressRef]{},
			expectedResult: []AddressRef{recordOne, recordTwo, recordThree},
		},
		{
			name: "success: returns record with given chain and type",
			givenState: []AddressRef{
				recordOne,
				recordTwo,
				recordThree,
			},
			giveFilters: []FilterFunc[AddressRefKey, AddressRef]{
				AddressRefByChainSelector(2),
				AddressRefByType("typeX"),
			},
			expectedResult: []AddressRef{recordTwo},
		},
		{
			name: "success: returns no record with given chain and type",
			givenState: []AddressRef{
				recordOne,
				recordTwo,
				recordThree,
			},
			giveFilters: []FilterFunc[AddressRefKey, AddressRef]{
				AddressRefByChainSelector(4),
				AddressRefByType("typeX"),
			},
			expectedResult: []AddressRef{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryAddressRefStore{Records: tt.givenState}
			filteredRecords := store.Filter(tt.giveFilters...)
			assert.Equal(t, tt.expectedResult, filteredRecords)
		})
	}
}

func TestMemoryAddressRefStore_RecordsOrdering(t *testing.T) {
	t.Parallel()

	var (
		recordOne = AddressRef{
			Address:       "0x1111111",
			ChainSelector: 1,
			Type:          "contract1",
			Version:       semver.MustParse("1.0.0"),
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label1", "label2",
			),
		}

		recordTwo = AddressRef{
			Address:       "0x2222222",
			ChainSelector: 2,
			Type:          "contract2",
			Version:       semver.MustParse("2.0.0"),
			Qualifier:     "qual2",
			Labels: NewLabelSet(
				"label3", "label4",
			),
		}

		recordThree = AddressRef{
			Address:       "0x3333333",
			ChainSelector: 3,
			Type:          "contract3",
			Version:       semver.MustParse("3.0.0"),
			Qualifier:     "qual3",
			Labels: NewLabelSet(
				"label5", "label6",
			),
		}

		recordFour = AddressRef{
			Address:       "0x4444444",
			ChainSelector: 4,
			Type:          "contract4",
			Version:       semver.MustParse("4.0.0"),
			Qualifier:     "qual4",
			Labels: NewLabelSet(
				"label7", "label8",
			),
		}
	)

	// Create a store with records in specific order
	originalStore := MemoryAddressRefStore{
		Records: []AddressRef{
			recordOne,
			recordTwo,
			recordThree,
			recordFour,
		},
	}

	// marshal the store to JSON
	data, err := json.Marshal(&originalStore)
	require.NoError(t, err, "Failed to marshal store")

	// unmarshal back to a new store instance
	var unmarshaledStore MemoryAddressRefStore
	err = json.Unmarshal(data, &unmarshaledStore)
	require.NoError(t, err, "Failed to unmarshal store")

	// verify the records are in the same order
	require.Equal(t, len(originalStore.Records), len(unmarshaledStore.Records),
		"number of records should match after unmarshaling")

	// compare each record to ensure order is maintained
	for i, originalRecord := range originalStore.Records {
		require.Equal(t, originalRecord.Address, unmarshaledStore.Records[i].Address,
			"address at position %d should match", i)
		require.Equal(t, originalRecord.ChainSelector, unmarshaledStore.Records[i].ChainSelector,
			"chainSelector at position %d should match", i)
		require.Equal(t, originalRecord.Type, unmarshaledStore.Records[i].Type,
			"type at position %d should match", i)
		require.Equal(t, originalRecord.Qualifier, unmarshaledStore.Records[i].Qualifier,
			"qualifier at position %d should match", i)

		// Version is a pointer, so we need to compare the actual version string
		require.NotNil(t, unmarshaledStore.Records[i].Version, "Version at position %d should not be nil", i)
		require.Equal(t, originalRecord.Version.String(), unmarshaledStore.Records[i].Version.String(),
			"version at position %d should match", i)

		require.Equal(t, originalRecord.Labels, unmarshaledStore.Records[i].Labels,
			"labels at position %d should match", i)
	}

	// additionally, verify the keys and lookups still work
	for i, originalRecord := range originalStore.Records {
		idx := unmarshaledStore.indexOf(originalRecord.Key())
		require.Equal(t, i, idx, "Index lookup for record %d should match original position", i)
	}
}

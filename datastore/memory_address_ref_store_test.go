package datastore

import (
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
		name              string
		givenState        []AddressRef
		deletedRemoteKeys []string
		giveRecord        AddressRef
		expectedState     []AddressRef
		expectedError     error
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
		{
			name:       "error: version is nil",
			givenState: []AddressRef{},
			giveRecord: AddressRef{
				Address:       "0x2324224",
				ChainSelector: 1,
				Type:          "type1",
				Qualifier:     "qual1",
				Labels: NewLabelSet(
					"label1", "label2", "label3",
				),
			},
			expectedError: ErrAddressRefVersionRequired,
		},
		{
			name:              "success: add record that was staged for deletion",
			givenState:        []AddressRef{},
			deletedRemoteKeys: []string{record.Key().String()},
			giveRecord:        record,
			expectedState: []AddressRef{
				record,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryAddressRefStore{Records: tt.givenState, DeletedRemoteKeys: tt.deletedRemoteKeys}
			assert.Len(t, store.DeletedRemoteKeys, len(tt.deletedRemoteKeys))
			err := store.Add(tt.giveRecord)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedState, store.Records)
				assert.Empty(t, store.DeletedRemoteKeys)
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
		name              string
		givenState        []AddressRef
		expectedState     []AddressRef
		giveRecord        AddressRef
		deletedRemoteKeys []string
		expectedError     error
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
		{
			name:              "success: upsert record that was staged for deletion",
			givenState:        []AddressRef{},
			deletedRemoteKeys: []string{oldRecord.Key().String()},
			giveRecord:        oldRecord,
			expectedState: []AddressRef{
				oldRecord,
			},
		},
		{
			name:              "error: version is nil",
			givenState:        []AddressRef{},
			deletedRemoteKeys: []string{oldRecord.Key().String()},
			giveRecord: AddressRef{
				Address:       "0x2324224",
				ChainSelector: 1,
				Type:          "type1",
				Version:       nil,
				Qualifier:     "qual1",
				Labels: NewLabelSet(
					"label1", "label2", "label3",
				),
			},
			expectedError: ErrAddressRefVersionRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryAddressRefStore{Records: tt.givenState, DeletedRemoteKeys: tt.deletedRemoteKeys}
			assert.Len(t, store.DeletedRemoteKeys, len(tt.deletedRemoteKeys))
			// Check the error, which will always be nil for the
			// in memory implementation, to satisfy the linter
			err := store.Upsert(tt.giveRecord)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedState, store.Records)
				assert.Empty(t, store.DeletedRemoteKeys)
			}
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
		nilRecord = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 1,
			Type:          "type1",
			Version:       nil,
			Qualifier:     "qual1",
			Labels: NewLabelSet(
				"label13", "label23", "label33",
			),
		}
	)

	tests := []struct {
		name              string
		givenState        []AddressRef
		expectedState     []AddressRef
		deletedRemoteKeys []string
		giveRecord        AddressRef
		expectedError     error
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
		{
			name: "error: version is nil",
			givenState: []AddressRef{
				nilRecord,
			},
			giveRecord: AddressRef{
				Address:       "0x2324224",
				ChainSelector: 1,
				Type:          "type1",
				Version:       nil,
				Qualifier:     "qual1",
				Labels: NewLabelSet(
					"label1", "label2", "label3",
				),
			},
			expectedError: ErrAddressRefVersionRequired,
		},
		{
			name: "success: update record that was staged for deletion",
			givenState: []AddressRef{
				oldRecord,
			},
			deletedRemoteKeys: []string{oldRecord.Key().String()},
			giveRecord:        oldRecord,
			expectedState: []AddressRef{
				oldRecord,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryAddressRefStore{Records: tt.givenState, DeletedRemoteKeys: tt.deletedRemoteKeys}
			assert.Len(t, store.DeletedRemoteKeys, len(tt.deletedRemoteKeys))
			err := store.Update(tt.giveRecord)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedState, store.Records)
				assert.Empty(t, store.DeletedRemoteKeys)
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

	t.Run("success: deletes record from Records", func(t *testing.T) {
		t.Parallel()

		store := MemoryAddressRefStore{Records: []AddressRef{recordOne, recordTwo, recordThree}}
		assert.Len(t, store.Records, 3)
		assert.Empty(t, store.DeletedRemoteKeys)
		err := store.Delete(recordTwo.Key())
		require.NoError(t, err)
		assert.Equal(t, []AddressRef{recordOne, recordThree}, store.Records)
		assert.Empty(t, store.DeletedRemoteKeys)
	})

	t.Run("error: absent key returns ErrAddressRefNotFound", func(t *testing.T) {
		t.Parallel()

		store := MemoryAddressRefStore{Records: []AddressRef{recordOne, recordThree}}
		assert.Len(t, store.Records, 2)
		assert.Empty(t, store.DeletedRemoteKeys)
		err := store.Delete(recordTwo.Key())
		require.ErrorIs(t, err, ErrAddressRefNotFound)
		assert.Equal(t, []AddressRef{recordOne, recordThree}, store.Records)
		assert.Empty(t, store.DeletedRemoteKeys)
	})

	t.Run("error: second Delete returns ErrAddressRefNotFound", func(t *testing.T) {
		t.Parallel()

		store := MemoryAddressRefStore{Records: []AddressRef{recordOne, recordTwo, recordThree}}
		assert.Len(t, store.Records, 3)
		require.NoError(t, store.Delete(recordTwo.Key()))
		assert.Len(t, store.Records, 2)
		require.ErrorIs(t, store.Delete(recordTwo.Key()), ErrAddressRefNotFound)
		assert.Len(t, store.Records, 2)
		assert.Empty(t, store.DeletedRemoteKeys)
	})
}

func TestMemoryAddressRefStore_RemoteDelete(t *testing.T) {
	t.Parallel()

	var (
		recordOne = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 1,
			Type:          "type1",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels:        NewLabelSet("label1", "label2", "label3"),
		}

		recordTwo = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 2,
			Type:          "typeX",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels:        NewLabelSet("label13", "label23", "label33"),
		}

		recordThree = AddressRef{
			Address:       "0x2324224",
			ChainSelector: 3,
			Type:          "typeZ",
			Version:       semver.MustParse("0.5.0"),
			Qualifier:     "qual1",
			Labels:        NewLabelSet("label13", "label23", "label33"),
		}
	)

	t.Run("success: stages key in DeletedRemoteKeys without removing from Records", func(t *testing.T) {
		t.Parallel()

		store := NewMemoryAddressRefStore()
		store.Records = append(store.Records, recordOne, recordTwo, recordThree)
		assert.Len(t, store.Records, 3)
		assert.Empty(t, store.DeletedRemoteKeys)
		err := store.RemoteDelete(recordTwo.Key())
		require.NoError(t, err)
		assert.Len(t, store.Records, 3)
		assert.Len(t, store.DeletedRemoteKeys, 1)
		assert.Contains(t, store.DeletedRemoteKeys, recordTwo.Key().String())
	})

	t.Run("success: second RemoteDelete does not duplicate entry", func(t *testing.T) {
		t.Parallel()

		store := NewMemoryAddressRefStore()
		store.Records = append(store.Records, recordOne, recordTwo, recordThree)
		require.NoError(t, store.RemoteDelete(recordTwo.Key()))
		require.NoError(t, store.RemoteDelete(recordTwo.Key()))
		assert.Len(t, store.Records, 3)
		assert.Len(t, store.DeletedRemoteKeys, 1)
	})
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

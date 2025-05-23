package datastore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMemoryEnvMetadataStore_Get(t *testing.T) {
	t.Parallel()

	var (
		recordOne = EnvMetadata{Metadata: TestMetadata{
			Data:    "metadata2",
			Version: 1,
			Tags:    []string{"tag1", "tag2"},
			Extra:   map[string]string{"key1": "value1", "key2": "value2"},
			Nested:  NestedMeta{Flag: true, Detail: "detail1"},
		}}
	)

	tests := []struct {
		name              string
		givenState        *EnvMetadata
		domain            string
		recordShouldExist bool
		expectedRecord    *EnvMetadata
		expectedError     error
	}{
		{
			name:              "env metadata set",
			givenState:        &recordOne,
			domain:            "example.com",
			recordShouldExist: true,
			expectedRecord:    &recordOne,
		},
		{
			name:              "env metadata not set",
			domain:            "nonexistent.com",
			recordShouldExist: false,
			expectedRecord:    &EnvMetadata{},
			expectedError:     ErrEnvMetadataNotSet,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryEnvMetadataStore{Record: tt.givenState}

			record, err := store.Get()
			if tt.recordShouldExist {
				require.NoError(t, err)
				require.Equal(t, *tt.expectedRecord, record)
			} else {
				require.Equal(t, tt.expectedError, err)
				require.Equal(t, *tt.expectedRecord, record)
			}
		})
	}
}

func TestMemoryEnvMetadataStore_Set(t *testing.T) {
	t.Parallel()

	var (
		recordOne = EnvMetadata{Metadata: TestMetadata{
			Data:    "data1",
			Version: 1,
			Tags:    []string{"tagA", "tagB"},
			Extra:   map[string]string{"foo": "bar"},
			Nested:  NestedMeta{Flag: false, Detail: "nested1"},
		}}
		recordTwo = EnvMetadata{Metadata: TestMetadata{
			Data:    "data2",
			Version: 2,
			Tags:    []string{"tagC", "tagD"},
			Extra:   map[string]string{"baz": "qux"},
			Nested:  NestedMeta{Flag: true, Detail: "nested2"},
		}}
	)

	tests := []struct {
		name           string
		initialState   *EnvMetadata
		updateRecord   EnvMetadata
		expectedRecord EnvMetadata
	}{
		{
			name:           "update existing record",
			initialState:   &recordOne,
			updateRecord:   recordTwo,
			expectedRecord: recordTwo,
		},
		{
			name:           "add new record",
			updateRecord:   recordOne,
			expectedRecord: recordOne,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := MemoryEnvMetadataStore{Record: tt.initialState}

			err := store.Set(tt.updateRecord)
			require.NoError(t, err)

			record, err := store.Get()
			require.NoError(t, err)
			require.Equal(t, tt.expectedRecord, record)
		})
	}
}

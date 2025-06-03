package datastore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMemoryEnvMetadataStore_Get(t *testing.T) {
	t.Parallel()

	var (
		recordOne = EnvMetadata{
			Metadata: DefaultMetadata{Data: "data1"},
		}
	)

	tests := []struct {
		name              string
		givenState        *EnvMetadata
		domain            string
		recordShouldExist bool
		expectedRecord    EnvMetadata
		expectedError     error
	}{
		{
			name:              "env metadata set",
			givenState:        &recordOne,
			domain:            "example.com",
			recordShouldExist: true,
			expectedRecord:    recordOne,
		},
		{
			name:              "env metadata not set",
			domain:            "nonexistent.com",
			recordShouldExist: false,
			expectedRecord:    EnvMetadata{},
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
				concrete, convErr := As[DefaultMetadata](record.Metadata)
				require.NoError(t, convErr, "As should not return an error for DefaultMetadata")
				require.Equal(t, tt.givenState.Metadata, concrete)
			} else {
				require.Equal(t, tt.expectedError, err)
			}
		})
	}
}

func TestMemoryEnvMetadataStore_Set(t *testing.T) {
	t.Parallel()

	var (
		recordOne = EnvMetadata{
			Metadata: DefaultMetadata{Data: "data1"},
		}
		recordTwo = EnvMetadata{
			Metadata: DefaultMetadata{Data: "data2"},
		}
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

			concrete, err := As[DefaultMetadata](record.Metadata)
			require.NoError(t, err, "As should not return an error for DefaultMetadata")
			require.Equal(t, tt.updateRecord.Metadata, concrete)
		})
	}
}

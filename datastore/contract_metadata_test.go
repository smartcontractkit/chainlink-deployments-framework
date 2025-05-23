package datastore

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContractMetadata_Clone(t *testing.T) {
	t.Parallel()

	original := ContractMetadata{
		ChainSelector: 1,
		Address:       "0x123",
		Metadata: TestMetadata{
			Data:    "test data",
			Version: 1,
			Tags:    []string{"tagA", "tagB"},
			Extra:   map[string]string{"foo": "bar"},
			Nested:  NestedMeta{Flag: true, Detail: "nested"},
		},
	}

	cloned := original.Clone()

	require.Equal(t, original.ChainSelector, cloned.ChainSelector)
	require.Equal(t, original.Address, cloned.Address)
	require.Equal(t, original.Metadata, cloned.Metadata)

	// Modify the original and ensure the cloned remains unchanged
	original.ChainSelector = 2
	original.Address = "0x456"
	original.Metadata = TestMetadata{Data: "updated data"}

	require.NotEqual(t, original.ChainSelector, cloned.ChainSelector)
	require.NotEqual(t, original.Address, cloned.Address)
	require.NotEqual(t, original.Metadata, cloned.Metadata)
}

func TestContractMetadata_Key(t *testing.T) {
	t.Parallel()

	metadata := ContractMetadata{
		ChainSelector: 1,
		Address:       "0x123",
		Metadata: TestMetadata{
			Data:    "test data",
			Version: 1,
			Tags:    []string{"tagA", "tagB"},
			Extra:   map[string]string{"foo": "bar"},
			Nested:  NestedMeta{Flag: true, Detail: "nested"},
		},
	}

	key := metadata.Key()
	expectedKey := NewContractMetadataKey(1, "0x123")

	require.Equal(t, expectedKey, key)
}

func TestContractMetadata_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    ContractMetadata
		expected string
	}{
		{
			name: "all fields set",
			input: ContractMetadata{
				Address:       "0xabc",
				ChainSelector: 123,
				Metadata: TestMetadata{
					Data:    "foo",
					Version: 42,
					Tags:    []string{"tag1", "tag2"},
					Extra:   map[string]string{"k1": "v1", "k2": "v2"},
					Nested:  NestedMeta{Flag: true, Detail: "details"},
				},
			},
			expected: `{"address":"0xabc","chainSelector":123,"metadata":{"data":"foo","version":42,"tags":["tag1","tag2"],"extra":{"k1":"v1","k2":"v2"},"nested":{"flag":true,"detail":"details"}}}`,
		},
		{
			name: "nil metadata",
			input: ContractMetadata{
				Address:       "0xdef",
				ChainSelector: 456,
				Metadata:      nil,
			},
			expected: `{"address":"0xdef","chainSelector":456,"metadata":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.input)
			require.NoError(t, err)
			require.JSONEq(t, tt.expected, string(b))
		})
	}
}

func TestContractMetadata_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantAddr string
		wantSel  uint64
		wantMeta *TestMetadata
	}{
		{
			name:     "all fields set",
			input:    `{"address":"0xabc","chainSelector":123,"metadata":{"data":"foo","version":42,"tags":["tag1","tag2"],"extra":{"k1":"v1","k2":"v2"},"nested":{"flag":true,"detail":"details"}}}`,
			wantAddr: "0xabc",
			wantSel:  123,
			wantMeta: &TestMetadata{
				Data:    "foo",
				Version: 42,
				Tags:    []string{"tag1", "tag2"},
				Extra:   map[string]string{"k1": "v1", "k2": "v2"},
				Nested:  NestedMeta{Flag: true, Detail: "details"},
			},
		},
		{
			name:     "null metadata",
			input:    `{"address":"0xdef","chainSelector":456,"metadata":null}`,
			wantAddr: "0xdef",
			wantSel:  456,
			wantMeta: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var meta ContractMetadata
			err := json.Unmarshal([]byte(tt.input), &meta)
			require.NoError(t, err)
			require.Equal(t, tt.wantAddr, meta.Address)
			require.Equal(t, tt.wantSel, meta.ChainSelector)
			if tt.wantMeta != nil {
				var got TestMetadata
				err = json.Unmarshal(meta.Metadata.(RawMetadata).raw, &got)
				require.NoError(t, err)
				require.Equal(t, *tt.wantMeta, got)
			} else {
				require.Equal(t, "null", string(meta.Metadata.(RawMetadata).raw))
			}
		})
	}
}

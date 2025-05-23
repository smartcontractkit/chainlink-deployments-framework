package datastore

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAs_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       CustomMetadata
		expect      TestMetadata
		expectError string
	}{
		{
			name: "success: RawMetadata with full TestMetadata",
			input: func() RawMetadata {
				meta := TestMetadata{
					Data:    "test-data",
					Version: 42,
					Tags:    []string{"tag1", "tag2"},
					Extra:   map[string]string{"foo": "bar", "baz": "qux"},
					Nested:  NestedMeta{Flag: true, Detail: "deep"},
				}
				b, _ := json.Marshal(meta)
				return RawMetadata{raw: b}
			}(),
			expect: TestMetadata{
				Data:    "test-data",
				Version: 42,
				Tags:    []string{"tag1", "tag2"},
				Extra:   map[string]string{"foo": "bar", "baz": "qux"},
				Nested:  NestedMeta{Flag: true, Detail: "deep"},
			},
		},
		{
			name:        "error: not RawMetadata",
			input:       TestMetadata{Data: "foo"},
			expectError: "metadata is not RawMetadata",
		},
		{
			name:        "error: unmarshal failure",
			input:       RawMetadata{raw: []byte("not-json")},
			expectError: "failed to unmarshal to target type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := As[TestMetadata](tt.input)
			if tt.expectError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expect, result)
			}
		})
	}
}

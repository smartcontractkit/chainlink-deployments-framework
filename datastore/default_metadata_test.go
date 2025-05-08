package datastore

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultMetadataMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    DefaultMetadata
		expected string
	}{
		{
			name:     "Supports any string",
			input:    DefaultMetadata{Data: "basic data"},
			expected: `{"data":"basic data"}`,
		},
		{
			name:     "Supports JSON object without double encoding",
			input:    DefaultMetadata{Data: `{"link":1,"name":"satoshi"}`},
			expected: `{"data":{"link":1,"name":"satoshi"}}`,
		},
		{
			name:     "Valid JSON array",
			input:    DefaultMetadata{Data: `[1,2,3,4]`},
			expected: `{"data":[1,2,3,4]}`,
		},
		{
			name:     "Empty string",
			input:    DefaultMetadata{Data: ""},
			expected: `{"data":""}`,
		},
		{
			name:     "Invalid JSON",
			input:    DefaultMetadata{Data: `{admin:"0xdeadbeef"}`},
			expected: `{"data":"{admin:\"0xdeadbeef\"}"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := json.Marshal(tt.input)
			require.NoError(t, err)
			require.NotNil(t, result)

			require.Equal(t, tt.expected, string(result), "Expected output does not match")
		})
	}
}

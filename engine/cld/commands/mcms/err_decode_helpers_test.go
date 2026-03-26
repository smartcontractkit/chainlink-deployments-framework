package mcms

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadExecutionErrorFromFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		expectError string
	}{
		{
			name: "success parses execution_error payload",
			input: `{
  "execution_error": {
    "RevertReasonDecoded": "Error(test)",
    "UnderlyingReasonRaw": "0x70de1b4b"
  }
}`,
		},
		{
			name:        "missing execution_error key",
			input:       `{"other_key": {}}`,
			expectError: "must contain an 'execution_error' key",
		},
		{
			name:        "invalid json",
			input:       `{"execution_error":`,
			expectError: "error unmarshaling JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ReadExecutionErrorFromFile([]byte(tt.input))
			if tt.expectError != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.expectError)
				require.Nil(t, got)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			require.Equal(t, "Error(test)", got.RevertReasonDecoded)
			require.Equal(t, "0x70de1b4b", got.UnderlyingReasonRaw)
		})
	}
}

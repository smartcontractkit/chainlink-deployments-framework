package commands

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConvertToJSONSafe_NumberPreservation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        any
		expectedType any // Use actual type for comparison
		expectedJSON string
		description  string
	}{
		{
			name:         "large integer should be preserved as json.Number",
			input:        float64(2e+21), // This is how YAML parses 2000000000000000000000
			expectedType: json.Number(""),
			expectedJSON: "2000000000000000000000",
			description:  "Large integers from YAML should preserve exact representation",
		},
		{
			name:         "another large integer",
			input:        float64(1e+16),
			expectedType: json.Number(""),
			expectedJSON: "10000000000000000",
			description:  "Another large integer case",
		},
		{
			name:         "negative large integer",
			input:        float64(-5e+15),
			expectedType: json.Number(""),
			expectedJSON: "-5000000000000000",
			description:  "Negative large integers should also be preserved",
		},
		{
			name:         "normal integer should stay as float64",
			input:        float64(123),
			expectedType: float64(0),
			expectedJSON: "123",
			description:  "Small integers don't need special handling",
		},
		{
			name:         "normal float should stay as float64",
			input:        float64(3.14),
			expectedType: float64(0),
			expectedJSON: "3.14",
			description:  "Regular floats should not be converted",
		},
		{
			name:         "threshold boundary - exactly 1e15",
			input:        float64(1e15),
			expectedType: json.Number(""),
			expectedJSON: "1000000000000000",
			description:  "Numbers at the threshold should be converted",
		},
		{
			name:         "just under threshold",
			input:        float64(9.99999e14),
			expectedType: float64(0),
			expectedJSON: "999999000000000",
			description:  "Numbers just under threshold should not be converted",
		},
		{
			name:         "string should pass through unchanged",
			input:        "hello world",
			expectedType: "",
			expectedJSON: `"hello world"`,
			description:  "Non-numeric types should pass through",
		},
		{
			name:         "boolean should pass through unchanged",
			input:        true,
			expectedType: true,
			expectedJSON: "true",
			description:  "Booleans should pass through unchanged",
		},
		{
			name: "nested map with large number",
			input: map[string]any{
				"bigInt":    float64(2e+21),
				"normalInt": float64(123),
				"message":   "test",
			},
			expectedType: map[string]any{},
			expectedJSON: `{"bigInt":2000000000000000000000,"message":"test","normalInt":123}`,
			description:  "Nested structures should preserve large numbers",
		},
		{
			name:         "array with large numbers",
			input:        []any{float64(2e+21), float64(123), "test"},
			expectedType: []any{},
			expectedJSON: `[2000000000000000000000,123,"test"]`,
			description:  "Arrays should preserve large numbers",
		},
		{
			name: "map with interface{} keys (YAML parsing artifact)",
			input: map[interface{}]any{
				"bigInt":                    float64(2e+21),
				uint64(1601528660175782575): "ethereum-sepolia", // Chain selector as key (smaller number)
			},
			expectedType: map[string]any{},
			expectedJSON: `{"1601528660175782575":"ethereum-sepolia","bigInt":2000000000000000000000}`,
			description:  "Should handle map[interface{}]any from YAML parsing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := convertToJSONSafe(tt.input)
			require.NoError(t, err, tt.description)

			// Check the type
			require.IsType(t, tt.expectedType, result, "Result should be of expected type")

			// Marshal to JSON and check the output
			jsonBytes, err := json.Marshal(result)
			require.NoError(t, err, "Should be able to marshal result to JSON")

			require.JSONEq(t, tt.expectedJSON, string(jsonBytes), "JSON output should match expected")
		})
	}
}

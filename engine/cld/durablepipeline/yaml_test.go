package durablepipeline

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
		expectedType any
		expectedJSON string
	}{
		{
			name:         "large integer should be preserved as json.Number",
			input:        float64(2e+21),
			expectedType: json.Number(""),
			expectedJSON: "2000000000000000000000",
		},
		{
			name:         "another large integer",
			input:        float64(1e+16),
			expectedType: json.Number(""),
			expectedJSON: "10000000000000000",
		},
		{
			name:         "negative large integer",
			input:        float64(-5e+15),
			expectedType: json.Number(""),
			expectedJSON: "-5000000000000000",
		},
		{
			name:         "normal integer should stay as float64",
			input:        float64(123),
			expectedType: float64(0),
			expectedJSON: "123",
		},
		{
			name:         "normal float should stay as float64",
			input:        float64(3.14),
			expectedType: float64(0),
			expectedJSON: "3.14",
		},
		{
			name:         "threshold boundary - exactly 1e15",
			input:        float64(1e15),
			expectedType: json.Number(""),
			expectedJSON: "1000000000000000",
		},
		{
			name:         "just under threshold",
			input:        float64(9.99999e14),
			expectedType: float64(0),
			expectedJSON: "999999000000000",
		},
		{
			name:         "string should pass through unchanged",
			input:        "hello world",
			expectedType: "",
			expectedJSON: `"hello world"`,
		},
		{
			name:         "boolean should pass through unchanged",
			input:        true,
			expectedType: true,
			expectedJSON: "true",
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
		},
		{
			name:         "array with large numbers",
			input:        []any{float64(2e+21), float64(123), "test"},
			expectedType: []any{},
			expectedJSON: `[2000000000000000000000,123,"test"]`,
		},
		{
			name: "map with interface{} keys",
			input: map[interface{}]any{
				"bigInt":                    float64(2e+21),
				uint64(1601528660175782575): "ethereum-sepolia",
			},
			expectedType: map[string]any{},
			expectedJSON: `{"1601528660175782575":"ethereum-sepolia","bigInt":2000000000000000000000}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := convertToJSONSafe(tt.input)
			require.NoError(t, err)
			require.IsType(t, tt.expectedType, result)

			jsonBytes, err := json.Marshal(result)
			require.NoError(t, err)
			require.JSONEq(t, tt.expectedJSON, string(jsonBytes))
		})
	}
}

func TestFindChangesetInData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		changesets    any
		changesetName string
		expectError   bool
		expectedData  any
		errorContains string
	}{
		{
			name: "array format - changeset found",
			changesets: []any{
				map[string]any{
					"test_changeset": map[string]any{
						"payload": map[string]any{"value": 123},
					},
				},
				map[string]any{
					"other_changeset": map[string]any{
						"payload": map[string]any{"value": 456},
					},
				},
			},
			changesetName: "test_changeset",
			expectedData: map[string]any{
				"payload": map[string]any{"value": 123},
			},
		},
		{
			name: "array format - changeset not found",
			changesets: []any{
				map[string]any{
					"other_changeset": map[string]any{
						"payload": map[string]any{"value": 123},
					},
				},
			},
			changesetName: "test_changeset",
			expectError:   true,
		},
		{
			name:          "array format - empty",
			changesets:    []any{},
			changesetName: "test_changeset",
			expectError:   true,
		},
		{
			name: "object format - should be rejected",
			changesets: map[string]any{
				"test_changeset": map[string]any{
					"payload": map[string]any{"value": 123},
				},
			},
			changesetName: "test_changeset",
			expectError:   true,
			errorContains: "expected array format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := FindChangesetInData(tt.changesets, tt.changesetName)
			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.ErrorContains(t, err, tt.errorContains)
				}

				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expectedData, result)
		})
	}
}

func TestGetAllChangesetsInOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		changesets    any
		expectedNames []string
		expectError   bool
		errorContains string
	}{
		{
			name: "object format - should return error",
			changesets: map[string]any{
				"first":  map[string]any{"payload": map[string]any{"value": 1}},
				"second": map[string]any{"payload": map[string]any{"value": 2}},
			},
			expectError:   true,
			errorContains: "expected array format",
		},
		{
			name: "array format",
			changesets: []any{
				map[string]any{
					"first": map[string]any{"payload": map[string]any{"value": 1}},
				},
				map[string]any{
					"second": map[string]any{"payload": map[string]any{"value": 2}},
				},
			},
			expectedNames: []string{"first", "second"},
		},
		{
			name: "array with invalid item",
			changesets: []any{
				"not a map",
			},
			expectedNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := GetAllChangesetsInOrder(tt.changesets)
			if tt.expectError {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.errorContains)

				return
			}

			require.NoError(t, err)
			var actualNames []string
			for _, changeset := range result {
				actualNames = append(actualNames, changeset.Name)
			}
			if len(tt.expectedNames) == 0 {
				require.Empty(t, actualNames)
				return
			}
			require.Equal(t, tt.expectedNames, actualNames)
		})
	}
}

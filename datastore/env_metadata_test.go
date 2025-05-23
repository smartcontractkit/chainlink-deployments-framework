package datastore

import (
	"encoding/json"
	"testing"

	require "github.com/stretchr/testify/require"
)

func TestEnvMetadata_Clone(t *testing.T) {
	t.Parallel()

	original := EnvMetadata{
		Metadata: TestMetadata{
			Data:    "test-value",
			Version: 1,
			Tags:    []string{"tagA", "tagB"},
			Extra:   map[string]string{"foo": "bar"},
			Nested:  NestedMeta{Flag: false, Detail: "clone-detail"},
		},
	}

	cloned := original.Clone()

	require.Equal(t, original.Metadata, cloned.Metadata)
	require.NotSame(t, &original.Metadata, &cloned.Metadata) // Ensure Metadata is a deep copy
}

func TestEnvMetadata_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    EnvMetadata
		expected string
	}{
		{
			name: "all fields set",
			input: EnvMetadata{
				Metadata: TestMetadata{
					Data:    "foo",
					Version: 42,
					Tags:    []string{"tag1", "tag2"},
					Extra:   map[string]string{"k1": "v1", "k2": "v2"},
					Nested:  NestedMeta{Flag: true, Detail: "details"},
				},
			},
			expected: `{"metadata":{"data":"foo","version":42,"tags":["tag1","tag2"],"extra":{"k1":"v1","k2":"v2"},"nested":{"flag":true,"detail":"details"}}}`,
		},
		{
			name: "nil metadata",
			input: EnvMetadata{
				Metadata: nil,
			},
			expected: `{"metadata":null}`,
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

func TestEnvMetadata_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantMeta *TestMetadata
	}{
		{
			name:  "all fields set",
			input: `{"metadata":{"data":"foo","version":42,"tags":["tag1","tag2"],"extra":{"k1":"v1","k2":"v2"},"nested":{"flag":true,"detail":"details"}}}`,
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
			input:    `{"metadata":null}`,
			wantMeta: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var env EnvMetadata
			err := json.Unmarshal([]byte(tt.input), &env)
			require.NoError(t, err)
			if tt.wantMeta != nil {
				var got TestMetadata
				err = json.Unmarshal(env.Metadata.(RawMetadata).raw, &got)
				require.NoError(t, err)
				require.Equal(t, *tt.wantMeta, got)
			} else {
				require.Equal(t, "null", string(env.Metadata.(RawMetadata).raw))
			}
		})
	}
}

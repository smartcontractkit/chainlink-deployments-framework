package input

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

func TestFindChangesetInData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		changesets any
		csName     string
		fileName   string
		want       any
		wantErr    string
	}{
		{
			name: "success",
			changesets: []any{
				map[string]any{"cs_a": map[string]any{"payload": "a"}},
				map[string]any{"cs_b": map[string]any{"payload": "b"}},
			},
			csName:   "cs_b",
			fileName: "test.yaml",
			want:     map[string]any{"payload": "b"},
		},
		{
			name: "not found",
			changesets: []any{
				map[string]any{"cs_a": map[string]any{"payload": "a"}},
			},
			csName:   "cs_missing",
			fileName: "test.yaml",
			wantErr:  "changeset 'cs_missing' not found in input file test.yaml",
		},
		{
			name:       "invalid format",
			changesets: map[string]any{"x": 1},
			csName:     "cs",
			fileName:   "test.yaml",
			wantErr:    "input file test.yaml has invalid 'changesets' format, expected array format",
		},
		{
			name:       "empty array",
			changesets: []any{},
			csName:     "cs",
			fileName:   "test.yaml",
			wantErr:    "input file test.yaml has empty 'changesets' array",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := FindChangesetInData(tt.changesets, tt.csName, tt.fileName)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Equal(t, tt.wantErr, err.Error())

				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestGetAllChangesetsInOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		changesets any
		fileName   string
		want       []ChangesetItem
		wantErr    string
	}{
		{
			name: "success",
			changesets: []any{
				map[string]any{"first": map[string]any{"payload": 1}},
				map[string]any{"second": map[string]any{"payload": 2}},
			},
			fileName: "test.yaml",
			want: []ChangesetItem{
				{Name: "first", Data: map[string]any{"payload": 1}},
				{Name: "second", Data: map[string]any{"payload": 2}},
			},
		},
		{
			name:       "invalid format",
			changesets: map[string]any{"x": 1},
			fileName:   "test.yaml",
			wantErr:    "input file test.yaml has invalid 'changesets' format for index access, expected array format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := GetAllChangesetsInOrder(tt.changesets, tt.fileName)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Equal(t, tt.wantErr, err.Error())

				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestConvertToJSONSafe(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   any
		want any
	}{
		{
			name: "map string any",
			in:   map[string]any{"a": 1, "b": "x"},
			want: map[string]any{"a": 1, "b": "x"},
		},
		{
			name: "map interface",
			in:   map[interface{}]interface{}{"key": "val", "num": 42},
			want: map[string]any{"key": "val", "num": 42},
		},
		{
			name: "slice",
			in:   []any{1, "two", map[string]any{"nested": true}},
			want: []any{1, "two", map[string]any{"nested": true}},
		},
		{
			name: "string scalar",
			in:   "hello",
			want: "hello",
		},
		{
			name: "int scalar",
			in:   42,
			want: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ConvertToJSONSafe(tt.in)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestYamlNodeToAny_Nil(t *testing.T) {
	t.Parallel()

	got := YamlNodeToAny(nil)
	require.Nil(t, got)
}

func TestSetChangesetEnvironmentVariable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		csName        string
		changesetData any
		fileName      string
		wantErr       string
	}{
		{
			name:          "missing payload",
			csName:        "my_cs",
			changesetData: map[string]any{"other": 1},
			fileName:      "test.yaml",
			wantErr:       "changeset 'my_cs' in input file test.yaml is missing required 'payload' field",
		},
		{
			name:          "invalid changeset data",
			csName:        "my_cs",
			changesetData: "not-a-map",
			fileName:      "test.yaml",
			wantErr:       "changeset 'my_cs' in input file test.yaml is not a valid object",
		},
		{
			name:   "invalid chain override negative",
			csName: "my_cs",
			changesetData: map[string]any{
				"payload":        map[string]any{},
				"chainOverrides": []any{-1},
			},
			fileName: "test.yaml",
			wantErr:  "chain override value must be non-negative, got: -1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := SetChangesetEnvironmentVariable(tt.csName, tt.changesetData, tt.fileName)

			require.Error(t, err)
			require.Equal(t, tt.wantErr, err.Error())
		})
	}
}

func TestSetChangesetEnvironmentVariable_Success(t *testing.T) {
	t.Parallel()

	orig := os.Getenv("DURABLE_PIPELINE_INPUT")
	t.Cleanup(func() { _ = os.Setenv("DURABLE_PIPELINE_INPUT", orig) })

	err := SetChangesetEnvironmentVariable("my_cs", map[string]any{"payload": map[string]any{"x": 1}}, "test.yaml")
	require.NoError(t, err)

	val := os.Getenv("DURABLE_PIPELINE_INPUT")
	require.NotEmpty(t, val)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(val), &decoded))
	require.Equal(t, map[string]any{"x": float64(1)}, decoded["payload"])
}

//nolint:paralleltest
func TestParseDurablePipelineYAML(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		fileName    string
		domKey      string
		envKey      string
		wantEnv     string
		wantDomain  string
		wantErr     string
	}{
		{
			name: "success",
			yamlContent: `environment: testnet
domain: mydomain
changesets:
  - cs1:
      payload:
        x: 1
`,
			fileName:   "p.yaml",
			domKey:     "mydomain",
			envKey:     "testnet",
			wantEnv:    "testnet",
			wantDomain: "mydomain",
		},
		{
			name: "missing environment",
			yamlContent: `domain: mydomain
changesets: []
`,
			fileName: "p.yaml",
			domKey:   "mydomain",
			envKey:   "testnet",
			wantErr:  "input file p.yaml is missing required 'environment' field",
		},
		{
			name: "missing changesets",
			yamlContent: `environment: testnet
domain: mydomain
`,
			fileName: "p.yaml",
			domKey:   "mydomain",
			envKey:   "testnet",
			wantErr:  "input file p.yaml is missing required 'changesets' field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(dir, "domains", tt.domKey, tt.envKey, "durable_pipelines", "inputs"), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "domains", tt.domKey, tt.envKey, "durable_pipelines", "inputs", tt.fileName), []byte(tt.yamlContent), 0o644)) //nolint:gosec

			originalWd, _ := os.Getwd()
			require.NoError(t, os.Chdir(dir))
			t.Cleanup(func() { _ = os.Chdir(originalWd) })

			dom := domain.NewDomain(dir, tt.domKey)
			got, err := ParseDurablePipelineYAML(tt.fileName, dom, tt.envKey)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Equal(t, tt.wantErr, err.Error())

				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantEnv, got.Environment)
			require.Equal(t, tt.wantDomain, got.Domain)
			require.NotNil(t, got.Changesets)
		})
	}
}

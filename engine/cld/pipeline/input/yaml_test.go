package input

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

func TestFindChangesetInData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		changesets any
		csName     string
		want       any
		wantErr    string
	}{
		{
			name: "success",
			changesets: []any{
				map[string]any{"cs_a": map[string]any{"payload": "a"}},
				map[string]any{"cs_b": map[string]any{"payload": "b"}},
			},
			csName: "cs_b",
			want:   map[string]any{"payload": "b"},
		},
		{
			name: "not found",
			changesets: []any{
				map[string]any{"cs_a": map[string]any{"payload": "a"}},
			},
			csName:  "cs_missing",
			wantErr: "changeset 'cs_missing' not found",
		},
		{
			name:       "invalid format",
			changesets: map[string]any{"x": 1},
			csName:     "cs",
			wantErr:    "invalid 'changesets' format, expected array format",
		},
		{
			name:       "empty array",
			changesets: []any{},
			csName:     "cs",
			wantErr:    "empty 'changesets' array",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := FindChangesetInData(tt.changesets, tt.csName)

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
		want       []ChangesetItem
		wantErr    string
	}{
		{
			name: "success",
			changesets: []any{
				map[string]any{"first": map[string]any{"payload": 1}},
				map[string]any{"second": map[string]any{"payload": 2}},
			},
			want: []ChangesetItem{
				{Name: "first", Data: map[string]any{"payload": 1}},
				{Name: "second", Data: map[string]any{"payload": 2}},
			},
		},
		{
			name:       "invalid format",
			changesets: map[string]any{"x": 1},
			wantErr:    "invalid 'changesets' format for index access, expected array format",
		},
		{
			name: "invalid array item type",
			changesets: []any{
				"first",
			},
			wantErr: "invalid changesets[0]: expected single-key object",
		},
		{
			name: "invalid multi-key item",
			changesets: []any{
				map[string]any{
					"first":  map[string]any{"payload": 1},
					"second": map[string]any{"payload": 2},
				},
			},
			wantErr: "invalid changesets[0]: expected single-key object, got 2 keys",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := GetAllChangesetsInOrder(tt.changesets)

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
		{
			name: "map with json number key",
			in:   map[interface{}]interface{}{json.Number("16015286601757825753"): "chain"},
			want: map[string]any{"16015286601757825753": "chain"},
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

	require.Nil(t, YamlNodeToAny(nil))
}

func TestYamlNodeToAny_AliasMapKeys(t *testing.T) {
	t.Parallel()

	yamlContent := `
sel: &sel_sepolia 16015286601757825753
other: &sel_arb 3478487238524512106
chains:
  *sel_sepolia: &cfg
    qualifier: UltraFastCurse
    timelockMinDelay: 0
  *sel_arb: *cfg
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	docAny, err := yamlNodeToAny(&root)
	require.NoError(t, err)
	doc := docAny.(map[string]any)
	chains := doc["chains"].(map[string]any)

	require.Equal(t, map[string]any{
		"16015286601757825753": map[string]any{
			"qualifier":        "UltraFastCurse",
			"timelockMinDelay": json.Number("0"),
		},
		"3478487238524512106": map[string]any{
			"qualifier":        "UltraFastCurse",
			"timelockMinDelay": json.Number("0"),
		},
	}, chains)
}

func TestYamlNodeToAny_MergeKeys(t *testing.T) {
	t.Parallel()

	yamlContent := `
base: &base_cfg
  qualifier: UltraFastCurse
  timelockMinDelay: 0
chains:
  16015286601757825753:
    <<: *base_cfg
    proposer:
      quorum: 1
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	docAny, err := yamlNodeToAny(&root)
	require.NoError(t, err)
	doc := docAny.(map[string]any)
	chains := doc["chains"].(map[string]any)
	chain := chains["16015286601757825753"].(map[string]any)

	require.Equal(t, "UltraFastCurse", chain["qualifier"])
	require.Equal(t, json.Number("0"), chain["timelockMinDelay"])
	require.Equal(t, map[string]any{"quorum": json.Number("1")}, chain["proposer"])
}

func TestYamlNodeToAny_MergeKeyOverrideAfter(t *testing.T) {
	t.Parallel()

	yamlContent := `
base: &base_cfg
  qualifier: Base
  timelockMinDelay: 0
chains:
  "123":
    <<: *base_cfg
    qualifier: Override
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	docAny, err := yamlNodeToAny(&root)
	require.NoError(t, err)
	chain := docAny.(map[string]any)["chains"].(map[string]any)["123"].(map[string]any)

	require.Equal(t, "Override", chain["qualifier"])
	require.Equal(t, json.Number("0"), chain["timelockMinDelay"])
}

func TestYamlNodeToAny_MergeKeyExplicitBeforeMerge(t *testing.T) {
	t.Parallel()

	yamlContent := `
base: &base_cfg
  qualifier: Base
  timelockMinDelay: 0
chains:
  "123":
    qualifier: Override
    <<: *base_cfg
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	docAny, err := yamlNodeToAny(&root)
	require.NoError(t, err)
	chain := docAny.(map[string]any)["chains"].(map[string]any)["123"].(map[string]any)

	require.Equal(t, "Override", chain["qualifier"])
	require.Equal(t, json.Number("0"), chain["timelockMinDelay"])
}

func TestYamlNodeToAny_MergeKeySequence(t *testing.T) {
	t.Parallel()

	yamlContent := `
a: &a
  x: 1
b: &b
  y: 2
m:
  <<: [*a, *b]
  z: 3
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	docAny, err := yamlNodeToAny(&root)
	require.NoError(t, err)
	m := docAny.(map[string]any)["m"].(map[string]any)

	require.Equal(t, json.Number("1"), m["x"])
	require.Equal(t, json.Number("2"), m["y"])
	require.Equal(t, json.Number("3"), m["z"])
}

func TestYamlNodeToAny_MergeKeySequenceConflict(t *testing.T) {
	t.Parallel()

	// YAML 1.1: earlier mappings in the merge sequence override later ones.
	// https://yaml.org/type/merge.html
	yamlContent := `
a: &a
  x: from_a
  only_a: 1
b: &b
  x: from_b
  only_b: 2
m:
  <<: [*a, *b]
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	docAny, err := yamlNodeToAny(&root)
	require.NoError(t, err)
	m := docAny.(map[string]any)["m"].(map[string]any)

	require.Equal(t, "from_a", m["x"])
	require.Equal(t, json.Number("1"), m["only_a"])
	require.Equal(t, json.Number("2"), m["only_b"])
}

func TestYamlNodeToAny_QuotedMergeLikeKey(t *testing.T) {
	t.Parallel()

	yamlContent := `"<<": 1`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	docAny, err := yamlNodeToAny(&root)
	require.NoError(t, err)
	doc := docAny.(map[string]any)

	require.Equal(t, json.Number("1"), doc["<<"])
}

func TestYamlNodeToAny_InvalidMergeValueSwallowsError(t *testing.T) {
	t.Parallel()

	yamlContent := `
m:
  <<: not_a_map
  z: 3
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	require.Nil(t, YamlNodeToAny(&root))
}

func TestYamlNodeToAny_InvalidMergeValue(t *testing.T) {
	t.Parallel()

	yamlContent := `
m:
  <<: not_a_map
  z: 3
`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yamlContent), &root))

	_, err := yamlNodeToAny(&root)
	require.Error(t, err)
	require.ErrorContains(t, err, "YAML merge key (<<) value must be a mapping or sequence of mappings")
}

func TestParseYAMLBytes_InvalidMergeValue(t *testing.T) {
	t.Parallel()

	yamlContent := `environment: testnet
domain: ccv
changesets:
  - cs1:
      payload:
        m:
          <<: not_a_map
          z: 3
`
	_, err := ParseYAMLBytes([]byte(yamlContent))
	require.Error(t, err)
	require.ErrorContains(t, err, "failed to decode YAML")
	require.ErrorContains(t, err, "YAML merge key (<<) value must be a mapping or sequence of mappings")
}

func TestBuildChangesetInputJSON_AliasChainSelectorKeys(t *testing.T) {
	t.Parallel()

	yamlContent := `
environment: staging_testnet
domain: ccv
changesets:
  - deploy_mcms:
      chainOverrides:
        - &sel_sepolia 16015286601757825753
      payload:
        adapterVersion: "1.0.0"
        chains:
          *sel_sepolia:
            qualifier: UltraFastCurse
            timelockMinDelay: 0
`
	dpYAML, err := ParseYAMLBytes([]byte(yamlContent))
	require.NoError(t, err)

	changesets, err := GetAllChangesetsInOrder(dpYAML.Changesets)
	require.NoError(t, err)
	require.Len(t, changesets, 1)

	inputJSON, err := BuildChangesetInputJSON(changesets[0].Name, changesets[0].Data)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(inputJSON), &decoded))

	payload := decoded["payload"].(map[string]any)
	chains := payload["chains"].(map[string]any)
	require.Contains(t, chains, "16015286601757825753")
	require.NotContains(t, chains, "sel_sepolia")
}

func TestSetChangesetEnvironmentVariable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		csName        string
		changesetData any
		wantErr       string
	}{
		{
			name:          "missing payload",
			csName:        "my_cs",
			changesetData: map[string]any{"other": 1},
			wantErr:       `failed to build input for changeset "my_cs": changeset "my_cs" is missing required 'payload' field`,
		},
		{
			name:          "invalid changeset data",
			csName:        "my_cs",
			changesetData: "not-a-map",
			wantErr:       `failed to build input for changeset "my_cs": changeset "my_cs" is not a valid object`,
		},
		{
			name:   "invalid chain override negative",
			csName: "my_cs",
			changesetData: map[string]any{
				"payload":        map[string]any{},
				"chainOverrides": []any{-1},
			},
			wantErr: `failed to build input for changeset "my_cs": chain override value must be non-negative, got: -1`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := SetChangesetEnvironmentVariable(tt.csName, tt.changesetData)

			require.Error(t, err)
			require.Equal(t, tt.wantErr, err.Error())
		})
	}
}

func TestSetChangesetEnvironmentVariable_Success(t *testing.T) {
	t.Parallel()

	orig := os.Getenv("DURABLE_PIPELINE_INPUT")
	t.Cleanup(func() { _ = os.Setenv("DURABLE_PIPELINE_INPUT", orig) })

	err := SetChangesetEnvironmentVariable("my_cs", map[string]any{"payload": map[string]any{"x": 1}})
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
			wantErr:  "input file p.yaml: missing required 'environment' field",
		},
		{
			name: "missing changesets",
			yamlContent: `environment: testnet
domain: mydomain
`,
			fileName: "p.yaml",
			domKey:   "mydomain",
			envKey:   "testnet",
			wantErr:  "input file p.yaml: missing required 'changesets' field",
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

			dom := domain.NewDomain(filepath.Join(dir, domain.DomainsDirName), tt.domKey)
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

//nolint:paralleltest // mutates process cwd and environment variables.
func TestPrepareInputForRunAuto(t *testing.T) {
	workspaceRoot := t.TempDir()
	dom := domain.NewDomain(workspaceRoot, "test")
	envKey := "testnet"
	inputsDir := filepath.Join(workspaceRoot, "domains", dom.String(), envKey, "durable_pipelines", "inputs")
	require.NoError(t, os.MkdirAll(inputsDir, 0o755))

	writeInput := func(name, content string) {
		t.Helper()
		require.NoError(t, os.WriteFile(filepath.Join(inputsDir, name), []byte(content), 0o600))
	}

	writeInput("single.yaml", `environment: testnet
domain: test
changesets:
  - only_changeset:
      payload:
        value: 1
`)
	writeInput("multi.yaml", `environment: testnet
domain: test
changesets:
  - first_changeset:
      payload:
        value: 1
  - second_changeset:
      payload:
        value: 2
`)

	originalWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workspaceRoot))
	t.Cleanup(func() { _ = os.Chdir(originalWd) })

	t.Cleanup(func() { _ = os.Unsetenv("DURABLE_PIPELINE_INPUT") })

	name, err := PrepareInputForRunAuto("single.yaml", dom, envKey)
	require.NoError(t, err)
	require.Equal(t, "only_changeset", name)
	require.Contains(t, os.Getenv("DURABLE_PIPELINE_INPUT"), `"value":1`)

	_, err = PrepareInputForRunAuto("multi.yaml", dom, envKey)
	require.Error(t, err)
	require.ErrorContains(t, err, "contains 2 changesets")

	writeInput("object-format.yaml", `environment: testnet
domain: test
changesets:
  only_changeset:
    payload:
      value: 1
`)
	_, err = PrepareInputForRunAuto("object-format.yaml", dom, envKey)
	require.Error(t, err)
	require.Equal(t, "input file object-format.yaml: invalid 'changesets' format, expected array format", err.Error())
}

package input

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/smartcontractkit/chainlink-deployments-framework/changeset/resolvers"
)

func TestResolveChangesetConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		yamlContent string
		csName      string
		resolver    resolvers.ConfigResolver
		wantErr     string
		checkGot    func(t *testing.T, got any)
	}{
		{
			name: "no resolver returns payload as-is",
			yamlContent: `
payload:
  foo: bar
  num: 42
`,
			csName:   "my_cs",
			resolver: nil,
			checkGot: func(t *testing.T, got any) {
				t.Helper()
				m, ok := got.(map[string]any)
				require.True(t, ok)
				require.Equal(t, "bar", m["foo"])
				require.EqualValues(t, 42, m["num"])
			},
		},
		{
			name: "with resolver returns resolved config",
			yamlContent: `
payload:
  input: value
`,
			csName: "my_cs",
			resolver: func(m map[string]any) (any, error) {
				return map[string]any{"resolved": true, "input": m["input"]}, nil
			},
			checkGot: func(t *testing.T, got any) {
				t.Helper()
				m, ok := got.(map[string]any)
				require.True(t, ok)
				require.Equal(t, true, m["resolved"])
				require.Equal(t, "value", m["input"])
			},
		},
		{
			name:        "invalid yaml returns decode error",
			yamlContent: "",
			csName:      "my_cs",
			resolver:    nil,
			wantErr:     "decode changeset data for my_cs: yaml: unmarshal errors:\n  line 0: cannot unmarshal !!str `not-a-map` into struct { Payload interface {} \"yaml:\\\"payload\\\"\" }",
			checkGot:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var valueNode *yaml.Node
			if tt.yamlContent != "" {
				var node yaml.Node
				require.NoError(t, yaml.Unmarshal([]byte(tt.yamlContent), &node))
				valueNode = node.Content[0]
			} else {
				valueNode = &yaml.Node{Kind: yaml.ScalarNode, Value: "not-a-map"}
			}

			got, err := ResolveChangesetConfig(valueNode, tt.csName, tt.resolver)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Equal(t, tt.wantErr, err.Error())

				return
			}
			require.NoError(t, err)
			if tt.checkGot != nil {
				tt.checkGot(t, got)
			}
		})
	}
}

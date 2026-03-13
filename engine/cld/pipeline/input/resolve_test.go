package input

import (
	"encoding/json"
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
				num, ok := m["num"].(json.Number)
				require.True(t, ok)
				require.Equal(t, "42", num.String())
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
			name: "with resolver preserves large integer as json number",
			yamlContent: `
payload:
  maxfeejuelspermsg: 200000000000000000111
`,
			csName: "my_cs",
			resolver: func(m map[string]any) (any, error) {
				maxFee, ok := m["maxfeejuelspermsg"].(json.Number)
				require.True(t, ok)
				require.Equal(t, "200000000000000000111", maxFee.String())

				return map[string]any{"resolved": true}, nil
			},
			checkGot: func(t *testing.T, got any) {
				t.Helper()
				m, ok := got.(map[string]any)
				require.True(t, ok)
				require.Equal(t, true, m["resolved"])
			},
		},
		{
			name: "missing payload returns explicit error",
			yamlContent: `
not_payload:
  value: 1
`,
			csName:   "my_cs",
			resolver: nil,
			wantErr:  "decode changeset data for my_cs: missing required 'payload' field",
			checkGot: nil,
		},
		{
			name:        "invalid yaml returns decode error",
			yamlContent: "",
			csName:      "my_cs",
			resolver:    nil,
			wantErr:     "decode changeset data for my_cs: expected mapping node",
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
				require.ErrorContains(t, err, tt.wantErr)

				return
			}
			require.NoError(t, err)
			if tt.checkGot != nil {
				tt.checkGot(t, got)
			}
		})
	}
}

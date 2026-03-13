package commands

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestResolveChangesetConfig_PreservesLargeIntegerForResolver(t *testing.T) {
	t.Parallel()

	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`
payload:
  maxfeejuelspermsg: 200000000000000000111
`), &node))
	valueNode := node.Content[0]

	resolved, err := resolveChangesetConfig(valueNode, "my_changeset", func(m map[string]any) (any, error) {
		maxFee, ok := m["maxfeejuelspermsg"].(json.Number)
		require.True(t, ok)
		require.Equal(t, "200000000000000000111", maxFee.String())

		return map[string]any{"resolved": true}, nil
	})
	require.NoError(t, err)

	resolvedMap, ok := resolved.(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, resolvedMap["resolved"])
}

func TestResolveChangesetConfig_NoResolverPreservesIntegerAsJSONNumber(t *testing.T) {
	t.Parallel()

	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`
payload:
  value: 42
`), &node))
	valueNode := node.Content[0]

	got, err := resolveChangesetConfig(valueNode, "my_changeset", nil)
	require.NoError(t, err)

	payload, ok := got.(map[string]any)
	require.True(t, ok)

	n, ok := payload["value"].(json.Number)
	require.True(t, ok)
	require.Equal(t, "42", n.String())
}

func TestResolveChangesetConfig_MissingPayloadReturnsError(t *testing.T) {
	t.Parallel()

	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`
not_payload:
  value: 1
`), &node))
	valueNode := node.Content[0]

	got, err := resolveChangesetConfig(valueNode, "my_changeset", nil)
	require.Error(t, err)
	require.Nil(t, got)
	require.ErrorContains(t, err, "decode changeset data for my_changeset: missing required 'payload' field")
}

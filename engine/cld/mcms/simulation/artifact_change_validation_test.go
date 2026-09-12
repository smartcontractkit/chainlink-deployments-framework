package simulation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/simulation/statediff"
)

func TestArtifactRejectsContradictoryChangeSides(t *testing.T) {
	t.Parallel()
	for _, kind := range []statediff.ChangeKind{statediff.KindAdded, statediff.KindRemoved} {
		t.Run(kind.String(), func(t *testing.T) {
			t.Parallel()
			for _, value := range []any{false, 0, "", []any{}, map[string]any{}, json.Number("9007199254740993")} {
				artifact := validationArtifact()
				change := &artifact.Chains["1"].Changes[0]
				change.Kind = kind
				change.Before, change.After = nil, nil
				if kind == statediff.KindAdded {
					change.Before = value
					require.ErrorContains(t, artifact.Validate(), "carries a before value")
				} else {
					change.After = value
					require.ErrorContains(t, artifact.Validate(), "carries an after value")
				}
			}
		})
	}
}

func TestArtifactNullChangesSurviveMarshalAndLoad(t *testing.T) {
	t.Parallel()
	for _, kind := range []statediff.ChangeKind{statediff.KindAdded, statediff.KindRemoved} {
		t.Run(kind.String(), func(t *testing.T) {
			t.Parallel()
			artifact := validationArtifact()
			change := &artifact.Chains["1"].Changes[0]
			change.Kind = kind
			change.Before, change.After = nil, nil
			require.NoError(t, artifact.Validate())
			raw, err := json.Marshal(artifact)
			require.NoError(t, err)
			path := filepath.Join(t.TempDir(), "null-change.json")
			require.NoError(t, os.WriteFile(path, raw, 0o600))
			loaded, err := LoadArtifact(path)
			require.NoError(t, err)
			require.Equal(t, kind, loaded.Chains["1"].Changes[0].Kind)
			require.Nil(t, loaded.Chains["1"].Changes[0].Before)
			require.Nil(t, loaded.Chains["1"].Changes[0].After)
		})
	}
}

func TestLoadArtifactRejectsMissingChangeKind(t *testing.T) {
	t.Parallel()
	raw, err := json.Marshal(validationArtifact())
	require.NoError(t, err)
	withoutKind := strings.Replace(string(raw), `"kind":"changed",`, "", 1)
	require.NotEqual(t, string(raw), withoutKind)
	path := filepath.Join(t.TempDir(), "missing-kind.json")
	require.NoError(t, os.WriteFile(path, []byte(withoutKind), 0o600))
	_, err = LoadArtifact(path)
	require.ErrorContains(t, err, "change kind is required")
}

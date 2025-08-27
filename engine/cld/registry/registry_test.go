package registry

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
)

// noopChangesetConfig holds configuration for the noop changeset.
type noopChangesetConfig struct {
}

// createNoopChangeset creates a noop changeset for testing.
func createNoopChangeset(t *testing.T, chainOverrides []uint64) changeset.ChangeSet {
	t.Helper()

	cs := cldf.CreateChangeSet(
		func(e cldf.Environment, config noopChangesetConfig) (cldf.ChangesetOutput, error) {
			return cldf.ChangesetOutput{}, nil
		},
		func(e cldf.Environment, config noopChangesetConfig) error {
			return nil
		},
	)

	if chainOverrides != nil {
		// Create a JSON input with chain overrides
		inputJSON := map[string]interface{}{
			"payload":        "{}",
			"chainOverrides": chainOverrides,
		}
		inputStr, err := json.Marshal(inputJSON)
		require.NoError(t, err)

		return changeset.Configure(cs).WithJSON(noopChangesetConfig{}, string(inputStr))
	}

	return changeset.Configure(cs).With(noopChangesetConfig{})
}

func Test_BaseRegistryProvider_Registry(t *testing.T) {
	t.Parallel()

	r := NewBaseRegistryProvider()

	require.NotNil(t, r.Registry())
}

func Test_BaseRegistryProvider_Init(t *testing.T) {
	t.Parallel()

	r := NewBaseRegistryProvider()

	require.NoError(t, r.Init())
}

func Test_Changesets_Apply(t *testing.T) {
	t.Parallel()

	var (
		archivedKey = "0001_archived_mig"
		archivedSHA = "abcdef"

		applyKey       = "0002_apply_mig"
		applyChangeset = createNoopChangeset(t, nil)
	)

	tests := []struct {
		name    string
		giveKey string
		want    cldf.ChangesetOutput
		wantErr string
	}{
		{
			name:    "with a registered changeset",
			giveKey: applyKey,
			want:    cldf.ChangesetOutput{},
		},
		{
			name:    "with an unregistered changeset",
			giveKey: "0003_unregistered",
			wantErr: "changeset '0003_unregistered' not found",
		},
		{
			name:    "with an archived changeset",
			giveKey: archivedKey,
			wantErr: "changeset '0001_archived_mig' is archived at SHA 'abcdef'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := NewChangesetsRegistry()

			// Setup the registry.
			r.Archive(archivedKey, archivedSHA)
			r.Add(applyKey, applyChangeset)

			got, err := r.Apply(tt.giveKey, cldf.Environment{})

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_Changesets_Add(t *testing.T) {
	t.Parallel()

	r := NewChangesetsRegistry()

	r.Add("0001_cap_reg", createNoopChangeset(t, nil))
	require.Equal(t, []string{"0001_cap_reg"}, r.keyHistory)

	r.Add("0002_cap_reg", createNoopChangeset(t, nil))
	require.Equal(t, []string{"0001_cap_reg", "0002_cap_reg"}, r.keyHistory)

	require.Panics(t, func() {
		r.Add("0002_same_index", createNoopChangeset(t, nil))
	}, "Add should panic when adding a key with the same index")

	require.Panics(t, func() {
		r.Add("0001_lower_index", createNoopChangeset(t, nil))
	}, "Add should panic when adding a key with lower index")

	require.Panics(t, func() {
		r.Add("xxxx_invalid_key", createNoopChangeset(t, nil))
	}, "Add should panic when adding a key with invalid format")

	require.Panics(t, func() {
		r.Add("InvalidChangesetKeyFormat", createNoopChangeset(t, nil))
	}, "Add should panic when adding an invalid changeset key format")

	r.SetValidate(false)
	require.NotPanics(t, func() {
		r.Add("0002_same_index", createNoopChangeset(t, nil))
	}, "Add should not panic when validation is disabled")
}

func Test_Changesets_Archive(t *testing.T) {
	t.Parallel()

	r := NewChangesetsRegistry()

	r.Archive("0001_cap_reg", "abcdef")
	require.Equal(t, []string{"0001_cap_reg"}, r.keyHistory)

	r.Archive("0002_cap_reg", "abcdef")
	require.Equal(t, []string{"0001_cap_reg", "0002_cap_reg"}, r.keyHistory)
}

func Test_Changesets_ListKeys(t *testing.T) {
	t.Parallel()

	r := NewChangesetsRegistry()

	r.Add("0001_cap_reg", createNoopChangeset(t, nil))
	r.Add("0002_cap_reg", createNoopChangeset(t, nil))
	require.Equal(t, []string{"0001_cap_reg", "0002_cap_reg"}, r.ListKeys())
}

func Test_Changesets_LatestKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		giveKeys []string
		want     string
		wantErr  string
	}{
		{
			name:     "with a single registered changeset",
			giveKeys: []string{"0001_cap_reg"},
			want:     "0001_cap_reg",
		},
		{
			name:     "with multiple registered changesets",
			giveKeys: []string{"0001_cap_reg", "0002_cap_reg", "0003_cap_reg"},
			want:     "0003_cap_reg",
		},
		{
			name:    "with no registered changesets",
			want:    "",
			wantErr: "no changesets found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := NewChangesetsRegistry()

			for _, key := range tt.giveKeys {
				r.Add(key, createNoopChangeset(t, nil))
			}

			got, err := r.LatestKey()

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_Changesets_SetValidate(t *testing.T) {
	t.Parallel()

	r := NewChangesetsRegistry()

	require.True(t, r.validate, "validate should be true by default")

	r.SetValidate(false)
	require.False(t, r.validate, "validate should be false after setting it to false")

	r.SetValidate(true)
	require.True(t, r.validate, "validate should be true after setting it to true")
}

func Test_Changesets_GetChangesetOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(*ChangesetsRegistry)
		giveKey string
		want    ChangesetConfig
		wantErr string
	}{
		{
			name:    "an invalid key",
			setup:   func(r *ChangesetsRegistry) {},
			giveKey: "invalid_key",
			wantErr: "changeset 'invalid_key' not found",
		},
		{
			name: "a changeset without options",
			setup: func(r *ChangesetsRegistry) {
				r.Add("0001_cap_reg", createNoopChangeset(t, nil))
			},
			giveKey: "0001_cap_reg",
			want:    ChangesetConfig{},
		},
		{
			name: "a changeset with OnlyLoadChainsFor option",
			setup: func(r *ChangesetsRegistry) {
				r.Add("0002_cap_reg", createNoopChangeset(t, nil), OnlyLoadChainsFor(1, 2))
			},
			giveKey: "0002_cap_reg",
			want: ChangesetConfig{
				ChainsToLoad: []uint64{1, 2},
				WithoutJD:    false,
			},
		},
		{
			name: "a changeset with WithoutJD option",
			setup: func(r *ChangesetsRegistry) {
				r.Add("0003_cap_reg", createNoopChangeset(t, nil), WithoutJD())
			},
			giveKey: "0003_cap_reg",
			want: ChangesetConfig{
				WithoutJD: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := NewChangesetsRegistry()
			tt.setup(r)

			got, err := r.GetChangesetOptions(tt.giveKey)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr, "should return expected error message")
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got, "should return expected changeset config")
			}
		})
	}
}

func Test_Changesets_InputChainOverrides(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(*ChangesetsRegistry)
		giveKey string
		want    []uint64
		wantErr string
	}{
		{
			name:    "an invalid key",
			setup:   func(r *ChangesetsRegistry) {},
			giveKey: "invalid_key",
			wantErr: "changeset 'invalid_key' not found",
		},
		{
			name: "a changeset without input chain overrides",
			setup: func(r *ChangesetsRegistry) {
				r.Add("0001_cap_reg", createNoopChangeset(t, nil))
			},
			giveKey: "0001_cap_reg",
			want:    nil,
		},
		{
			name: "a changeset with input chain overrides",
			setup: func(r *ChangesetsRegistry) {
				r.Add("0002_cap_reg", createNoopChangeset(t, []uint64{1, 2}))
			},
			giveKey: "0002_cap_reg",
			want:    []uint64{1, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := NewChangesetsRegistry()
			tt.setup(r)

			got, err := r.GetConfigurations(tt.giveKey)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr, "should return expected error message")
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got.InputChainOverrides, "should return expected input chain overrides")
			}
		})
	}
}

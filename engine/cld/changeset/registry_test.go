package changeset

import (
	"testing"

	"github.com/stretchr/testify/require"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

// noopChangeset is a changeset that does nothing.
//
// Used for testing.
type noopChangeset struct {
	chainOverrides []uint64
}

func (noopChangeset) noop() {}

func (noopChangeset) Apply(e fdeployment.Environment) (fdeployment.ChangesetOutput, error) {
	return fdeployment.ChangesetOutput{}, nil
}

func (n noopChangeset) Configurations() (Configurations, error) {
	return Configurations{
		InputChainOverrides: n.chainOverrides,
	}, nil
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
		applyChangeset = noopChangeset{}
	)

	tests := []struct {
		name    string
		giveKey string
		want    fdeployment.ChangesetOutput
		wantErr string
	}{
		{
			name:    "with a registered changeset",
			giveKey: applyKey,
			want:    fdeployment.ChangesetOutput{},
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

			got, err := r.Apply(tt.giveKey, fdeployment.Environment{})

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

	r.Add("0001_cap_reg", noopChangeset{})
	require.Equal(t, []string{"0001_cap_reg"}, r.keyHistory)

	r.Add("0002_cap_reg", noopChangeset{})
	require.Equal(t, []string{"0001_cap_reg", "0002_cap_reg"}, r.keyHistory)

	require.Panics(t, func() {
		r.Add("0002_same_index", noopChangeset{})
	}, "Add should panic when adding a key with the same index")

	require.Panics(t, func() {
		r.Add("0001_lower_index", noopChangeset{})
	}, "Add should panic when adding a key with lower index")

	require.Panics(t, func() {
		r.Add("xxxx_invalid_key", noopChangeset{})
	}, "Add should panic when adding a key with invalid format")

	require.Panics(t, func() {
		r.Add("InvalidChangesetKeyFormat", noopChangeset{})
	}, "Add should panic when adding an invalid changeset key format")

	r.SetValidate(false)
	require.NotPanics(t, func() {
		r.Add("0002_same_index", noopChangeset{})
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

	r.Add("0001_cap_reg", noopChangeset{})
	r.Add("0002_cap_reg", noopChangeset{})
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
				r.Add(key, noopChangeset{})
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
				r.Add("0001_cap_reg", noopChangeset{})
			},
			giveKey: "0001_cap_reg",
			want:    ChangesetConfig{},
		},
		{
			name: "a changeset with OnlyLoadChainsFor option",
			setup: func(r *ChangesetsRegistry) {
				r.Add("0002_cap_reg", noopChangeset{}, OnlyLoadChainsFor(1, 2))
			},
			giveKey: "0002_cap_reg",
			want: ChangesetConfig{
				ChainsToLoad: []uint64{1, 2},
				WithoutJD:    false,
			},
		},
		{
			name: "a changeset with OnlyLoadChainsFor empty (load no chains)",
			setup: func(r *ChangesetsRegistry) {
				r.Add("0003_cap_reg", noopChangeset{}, OnlyLoadChainsFor())
			},
			giveKey: "0003_cap_reg",
			want: ChangesetConfig{
				ChainsToLoad: []uint64{},
				WithoutJD:    false,
			},
		},
		{
			name: "a changeset with WithoutJD option",
			setup: func(r *ChangesetsRegistry) {
				r.Add("0003_cap_reg", noopChangeset{}, WithoutJD())
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
				r.Add("0001_cap_reg", noopChangeset{})
			},
			giveKey: "0001_cap_reg",
			want:    nil,
		},
		{
			name: "a changeset with input chain overrides",
			setup: func(r *ChangesetsRegistry) {
				r.Add("0002_cap_reg", noopChangeset{chainOverrides: []uint64{1, 2}})
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

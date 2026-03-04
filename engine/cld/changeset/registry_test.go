package changeset

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
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

// recordingChangeset tracks whether Apply was called and returns configurable output/error.
type recordingChangeset struct {
	applyCalled bool
	output      fdeployment.ChangesetOutput
	err         error
}

func (*recordingChangeset) noop() {}

func (r *recordingChangeset) Apply(_ fdeployment.Environment) (fdeployment.ChangesetOutput, error) {
	r.applyCalled = true
	return r.output, r.err
}

func (*recordingChangeset) Configurations() (Configurations, error) {
	return Configurations{}, nil
}

// orderRecordingChangeset records "apply" calls into its internal order slice for tests.
type orderRecordingChangeset struct {
	order []string
}

func (*orderRecordingChangeset) noop() {}

func (o *orderRecordingChangeset) Apply(_ fdeployment.Environment) (fdeployment.ChangesetOutput, error) {
	o.order = append(o.order, "apply")
	return fdeployment.ChangesetOutput{}, nil
}

func (*orderRecordingChangeset) Configurations() (Configurations, error) {
	return Configurations{}, nil
}

func hookTestEnv(t *testing.T) fdeployment.Environment {
	t.Helper()

	return fdeployment.Environment{
		Name:       "test-env",
		Logger:     logger.Test(t),
		GetContext: func() context.Context { return context.Background() },
	}
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

func Test_Changesets_AddGlobalPreHooks(t *testing.T) {
	t.Parallel()

	r := NewChangesetsRegistry()

	h1 := PreHook{HookDefinition: HookDefinition{Name: "h1"}}
	h2 := PreHook{HookDefinition: HookDefinition{Name: "h2"}}
	h3 := PreHook{HookDefinition: HookDefinition{Name: "h3"}}

	r.AddGlobalPreHooks(h1, h2)
	require.Len(t, r.globalPreHooks, 2)
	require.Equal(t, "h1", r.globalPreHooks[0].Name)
	require.Equal(t, "h2", r.globalPreHooks[1].Name)

	r.AddGlobalPreHooks(h3)
	require.Len(t, r.globalPreHooks, 3, "multiple calls should be additive")
	require.Equal(t, "h3", r.globalPreHooks[2].Name)
}

func Test_Changesets_AddGlobalPostHooks(t *testing.T) {
	t.Parallel()

	r := NewChangesetsRegistry()

	h1 := PostHook{HookDefinition: HookDefinition{Name: "h1"}}
	h2 := PostHook{HookDefinition: HookDefinition{Name: "h2"}}
	h3 := PostHook{HookDefinition: HookDefinition{Name: "h3"}}

	r.AddGlobalPostHooks(h1, h2)
	require.Len(t, r.globalPostHooks, 2)
	require.Equal(t, "h1", r.globalPostHooks[0].Name)
	require.Equal(t, "h2", r.globalPostHooks[1].Name)

	r.AddGlobalPostHooks(h3)
	require.Len(t, r.globalPostHooks, 3, "multiple calls should be additive")
	require.Equal(t, "h3", r.globalPostHooks[2].Name)
}

func Test_Changesets_NoGlobalHooks_Unchanged(t *testing.T) {
	t.Parallel()

	r := NewChangesetsRegistry()
	r.Add("0001_test", noopChangeset{})

	require.Nil(t, r.globalPreHooks)
	require.Nil(t, r.globalPostHooks)

	got, err := r.Apply("0001_test", fdeployment.Environment{})
	require.NoError(t, err)
	require.Equal(t, fdeployment.ChangesetOutput{}, got)
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

func Test_Apply_PreHookAbort_BlocksChangeset(t *testing.T) {
	t.Parallel()

	cs := &recordingChangeset{}

	r := NewChangesetsRegistry()
	r.entries["test-cs"] = registryEntry{
		changeset: cs,
		preHooks: []PreHook{{
			HookDefinition: HookDefinition{Name: "blocker", FailurePolicy: Abort},
			Func: func(_ context.Context, _ PreHookParams) error {
				return errors.New("service unhealthy")
			},
		}},
	}

	_, err := r.Apply("test-cs", hookTestEnv(t))
	require.Error(t, err)
	require.ErrorContains(t, err, "service unhealthy")
	assert.False(t, cs.applyCalled, "changeset should not have been called")
}

func Test_Apply_PreHookWarn_ContinuesExecution(t *testing.T) {
	t.Parallel()

	cs := &recordingChangeset{}

	r := NewChangesetsRegistry()
	r.entries["test-cs"] = registryEntry{
		changeset: cs,
		preHooks: []PreHook{{
			HookDefinition: HookDefinition{Name: "warner", FailurePolicy: Warn},
			Func: func(_ context.Context, _ PreHookParams) error {
				return errors.New("non-critical failure")
			},
		}},
	}

	_, err := r.Apply("test-cs", hookTestEnv(t))
	require.NoError(t, err)
	assert.True(t, cs.applyCalled, "changeset should still run after Warn hook failure")
}

func Test_Apply_PostHookReceivesOutputAndErr(t *testing.T) {
	t.Parallel()

	expectedOutput := fdeployment.ChangesetOutput{}
	expectedErr := errors.New("apply failed")
	cs := &recordingChangeset{
		output: expectedOutput,
		err:    expectedErr,
	}

	var receivedParams PostHookParams

	r := NewChangesetsRegistry()
	r.entries["test-cs"] = registryEntry{
		changeset: cs,
		postHooks: []PostHook{{
			HookDefinition: HookDefinition{Name: "observer", FailurePolicy: Warn},
			Func: func(_ context.Context, params PostHookParams) error {
				receivedParams = params
				return nil
			},
		}},
	}

	_, err := r.Apply("test-cs", hookTestEnv(t))
	require.Error(t, err)
	assert.Equal(t, expectedOutput, receivedParams.Output)
	assert.Equal(t, expectedErr, receivedParams.Err)
	assert.Equal(t, "test-cs", receivedParams.ChangesetKey)
	assert.Equal(t, "test-env", receivedParams.Env.Name)
}

func Test_Apply_PostHookAbort_AfterSuccessfulApply(t *testing.T) {
	t.Parallel()

	cs := recordingChangeset{}

	r := NewChangesetsRegistry()
	r.entries["test-cs"] = registryEntry{
		changeset: &cs,
		postHooks: []PostHook{{
			HookDefinition: HookDefinition{Name: "post-blocker", FailurePolicy: Abort},
			Func: func(_ context.Context, _ PostHookParams) error {
				return errors.New("post-hook failed")
			},
		}},
	}

	_, err := r.Apply("test-cs", hookTestEnv(t))
	require.Error(t, err)
	require.ErrorContains(t, err, "post-hook failed")
	assert.True(t, cs.applyCalled, "changeset should have been called before post-hook")
}

func Test_Apply_PostHookFailure_AfterFailedApply_ApplyErrorWins(t *testing.T) {
	t.Parallel()

	cs := recordingChangeset{
		err: errors.New("apply error"),
	}

	r := NewChangesetsRegistry()
	r.entries["test-cs"] = registryEntry{
		changeset: &cs,
		postHooks: []PostHook{{
			HookDefinition: HookDefinition{Name: "post-also-fails", FailurePolicy: Abort},
			Func: func(_ context.Context, _ PostHookParams) error {
				return errors.New("post-hook error")
			},
		}},
	}

	_, err := r.Apply("test-cs", hookTestEnv(t))
	require.Error(t, err)
	require.ErrorContains(t, err, "apply error", "Apply error should be returned, not post-hook error")
	assert.NotContains(t, err.Error(), "post-hook error")
}

func Test_Apply_ExecutionOrder(t *testing.T) {
	t.Parallel()

	cs := &orderRecordingChangeset{}

	r := NewChangesetsRegistry()
	r.SetValidate(false)

	r.AddGlobalPreHooks(PreHook{
		HookDefinition: HookDefinition{Name: "global-pre"},
		Func: func(_ context.Context, _ PreHookParams) error {
			cs.order = append(cs.order, "global-pre")
			return nil
		},
	})
	r.AddGlobalPostHooks(PostHook{
		HookDefinition: HookDefinition{Name: "global-post"},
		Func: func(_ context.Context, _ PostHookParams) error {
			cs.order = append(cs.order, "global-post")
			return nil
		},
	})

	r.Add("test-cs", noopChangeset{})

	r.entries["test-cs"] = registryEntry{
		changeset: cs,
		preHooks: []PreHook{{
			HookDefinition: HookDefinition{Name: "cs-pre"},
			Func: func(_ context.Context, _ PreHookParams) error {
				cs.order = append(cs.order, "cs-pre")
				return nil
			},
		}},
		postHooks: []PostHook{{
			HookDefinition: HookDefinition{Name: "cs-post"},
			Func: func(_ context.Context, _ PostHookParams) error {
				cs.order = append(cs.order, "cs-post")
				return nil
			},
		}},
	}

	_, err := r.Apply("test-cs", hookTestEnv(t))
	require.NoError(t, err)

	expected := []string{"global-pre", "cs-pre", "apply", "cs-post", "global-post"}
	assert.Equal(t, expected, cs.order)
}

func Test_Apply_GlobalPreHookAbort_BlocksChangeset(t *testing.T) {
	t.Parallel()

	cs := &recordingChangeset{}

	r := NewChangesetsRegistry()
	r.SetValidate(false)

	r.AddGlobalPreHooks(PreHook{
		HookDefinition: HookDefinition{Name: "global-blocker", FailurePolicy: Abort},
		Func: func(_ context.Context, _ PreHookParams) error {
			return errors.New("global pre-hook blocked")
		},
	})

	r.Add("test-cs", cs)

	_, err := r.Apply("test-cs", hookTestEnv(t))
	require.Error(t, err)
	require.ErrorContains(t, err, "global pre-hook blocked")
	assert.False(t, cs.applyCalled, "changeset should not run when global pre-hook aborts")
}

func Test_Apply_HookEnvConstruction(t *testing.T) {
	t.Parallel()

	cs := &recordingChangeset{}

	var receivedEnv HookEnv

	r := NewChangesetsRegistry()
	r.entries["test-cs"] = registryEntry{
		changeset: cs,
		preHooks: []PreHook{{
			HookDefinition: HookDefinition{Name: "env-checker", FailurePolicy: Warn},
			Func: func(_ context.Context, params PreHookParams) error {
				receivedEnv = params.Env
				return nil
			},
		}},
	}

	env := hookTestEnv(t)
	_, err := r.Apply("test-cs", env)
	require.NoError(t, err)

	assert.Equal(t, "test-env", receivedEnv.Name)
	assert.NotNil(t, receivedEnv.Logger)
}

func Test_WithPreHooks_Additive(t *testing.T) {
	t.Parallel()

	h1 := PreHook{HookDefinition: HookDefinition{Name: "h1"}, Func: func(_ context.Context, _ PreHookParams) error { return nil }}
	h2 := PreHook{HookDefinition: HookDefinition{Name: "h2"}, Func: func(_ context.Context, _ PreHookParams) error { return nil }}
	h3 := PreHook{HookDefinition: HookDefinition{Name: "h3"}, Func: func(_ context.Context, _ PreHookParams) error { return nil }}

	cs := Configure(MyChangeSet).With("cfg").
		WithPreHooks(h1, h2).
		WithPreHooks(h3)

	hooks := cs.(hookCarrier).getPreHooks()
	require.Len(t, hooks, 3)
	assert.Equal(t, "h1", hooks[0].Name)
	assert.Equal(t, "h2", hooks[1].Name)
	assert.Equal(t, "h3", hooks[2].Name)
}

func Test_WithPostHooks_Additive(t *testing.T) {
	t.Parallel()

	h1 := PostHook{HookDefinition: HookDefinition{Name: "h1"}, Func: func(_ context.Context, _ PostHookParams) error { return nil }}
	h2 := PostHook{HookDefinition: HookDefinition{Name: "h2"}, Func: func(_ context.Context, _ PostHookParams) error { return nil }}

	cs := Configure(MyChangeSet).With("cfg").
		WithPostHooks(h1).
		WithPostHooks(h2)

	hooks := cs.(hookCarrier).getPostHooks()
	require.Len(t, hooks, 2)
	assert.Equal(t, "h1", hooks[0].Name)
	assert.Equal(t, "h2", hooks[1].Name)
}

func Test_WithHooks_ThenWith_CarriesForward(t *testing.T) {
	t.Parallel()

	pre := PreHook{HookDefinition: HookDefinition{Name: "pre-before-then"}, Func: func(_ context.Context, _ PreHookParams) error { return nil }}
	post := PostHook{HookDefinition: HookDefinition{Name: "post-before-then"}, Func: func(_ context.Context, _ PostHookParams) error { return nil }}

	cs := Configure(MyChangeSet).With("cfg").
		WithPreHooks(pre).
		WithPostHooks(post).
		ThenWith(func(_ fdeployment.Environment, o fdeployment.ChangesetOutput) (fdeployment.ChangesetOutput, error) {
			return o, nil
		})

	carrier := cs.(hookCarrier)
	require.Len(t, carrier.getPreHooks(), 1)
	assert.Equal(t, "pre-before-then", carrier.getPreHooks()[0].Name)
	require.Len(t, carrier.getPostHooks(), 1)
	assert.Equal(t, "post-before-then", carrier.getPostHooks()[0].Name)
}

func Test_WithHooks_ThenWith_AdditiveAfterThenWith(t *testing.T) {
	t.Parallel()

	preBefore := PreHook{HookDefinition: HookDefinition{Name: "pre-before"}, Func: func(_ context.Context, _ PreHookParams) error { return nil }}
	preAfter := PreHook{HookDefinition: HookDefinition{Name: "pre-after"}, Func: func(_ context.Context, _ PreHookParams) error { return nil }}
	postAfter := PostHook{HookDefinition: HookDefinition{Name: "post-after"}, Func: func(_ context.Context, _ PostHookParams) error { return nil }}

	cs := Configure(MyChangeSet).With("cfg").
		WithPreHooks(preBefore).
		ThenWith(func(_ fdeployment.Environment, o fdeployment.ChangesetOutput) (fdeployment.ChangesetOutput, error) {
			return o, nil
		}).
		WithPreHooks(preAfter).
		WithPostHooks(postAfter)

	carrier := cs.(hookCarrier)
	preHooks := carrier.getPreHooks()
	require.Len(t, preHooks, 2)
	assert.Equal(t, "pre-before", preHooks[0].Name)
	assert.Equal(t, "pre-after", preHooks[1].Name)

	postHooks := carrier.getPostHooks()
	require.Len(t, postHooks, 1)
	assert.Equal(t, "post-after", postHooks[0].Name)
}

func Test_FluentAPI_HooksExtractedByAdd(t *testing.T) {
	t.Parallel()

	var order []string

	pre := PreHook{
		HookDefinition: HookDefinition{Name: "fluent-pre"},
		Func: func(_ context.Context, _ PreHookParams) error {
			order = append(order, "fluent-pre")
			return nil
		},
	}
	post := PostHook{
		HookDefinition: HookDefinition{Name: "fluent-post"},
		Func: func(_ context.Context, _ PostHookParams) error {
			order = append(order, "fluent-post")
			return nil
		},
	}

	cs := Configure(MyChangeSet).With("cfg").
		WithPreHooks(pre).
		WithPostHooks(post)

	r := NewChangesetsRegistry()
	r.SetValidate(false)
	r.Add("test-cs", cs)

	entry := r.entries["test-cs"]
	require.Len(t, entry.preHooks, 1, "Add should extract pre-hooks via hookCarrier")
	require.Len(t, entry.postHooks, 1, "Add should extract post-hooks via hookCarrier")

	_, err := r.Apply("test-cs", hookTestEnv(t))
	require.NoError(t, err)
	assert.Equal(t, []string{"fluent-pre", "fluent-post"}, order)
}

func Test_FluentAPI_ThenWith_HooksExtractedByAdd(t *testing.T) {
	t.Parallel()

	var order []string

	pre := PreHook{
		HookDefinition: HookDefinition{Name: "pp-pre"},
		Func: func(_ context.Context, _ PreHookParams) error {
			order = append(order, "pp-pre")
			return nil
		},
	}
	post := PostHook{
		HookDefinition: HookDefinition{Name: "pp-post"},
		Func: func(_ context.Context, _ PostHookParams) error {
			order = append(order, "pp-post")
			return nil
		},
	}

	cs := Configure(MyChangeSet).With("cfg").
		WithPreHooks(pre).
		ThenWith(func(_ fdeployment.Environment, o fdeployment.ChangesetOutput) (fdeployment.ChangesetOutput, error) {
			return o, nil
		}).
		WithPostHooks(post)

	r := NewChangesetsRegistry()
	r.SetValidate(false)
	r.Add("test-cs", cs)

	entry := r.entries["test-cs"]
	require.Len(t, entry.preHooks, 1)
	require.Len(t, entry.postHooks, 1)

	_, err := r.Apply("test-cs", hookTestEnv(t))
	require.NoError(t, err)
	assert.Equal(t, []string{"pp-pre", "pp-post"}, order)
}

func Test_WithHooks_SliceIsolation(t *testing.T) {
	t.Parallel()

	h1 := PreHook{HookDefinition: HookDefinition{Name: "h1"}, Func: func(_ context.Context, _ PreHookParams) error { return nil }}
	h2 := PreHook{HookDefinition: HookDefinition{Name: "h2"}, Func: func(_ context.Context, _ PreHookParams) error { return nil }}

	base := Configure(MyChangeSet).With("cfg").WithPreHooks(h1)
	branch := base.WithPreHooks(h2)

	baseHooks := base.(hookCarrier).getPreHooks()
	branchHooks := branch.(hookCarrier).getPreHooks()

	require.Len(t, baseHooks, 1, "base should be unaffected by branch append")
	require.Len(t, branchHooks, 2, "branch should have both hooks")
}

// Integration tests — full hook lifecycle through the registry provider pattern.

func Test_Integration_MultiChangeset_GlobalAndPerCSHooks(t *testing.T) {
	t.Parallel()

	var order []string

	globalPre := PreHook{
		HookDefinition: HookDefinition{Name: "global-pre"},
		Func: func(_ context.Context, p PreHookParams) error {
			order = append(order, "global-pre:"+p.ChangesetKey)
			return nil
		},
	}
	globalPost := PostHook{
		HookDefinition: HookDefinition{Name: "global-post"},
		Func: func(_ context.Context, p PostHookParams) error {
			order = append(order, "global-post:"+p.ChangesetKey)
			return nil
		},
	}

	csAPre := PreHook{
		HookDefinition: HookDefinition{Name: "csA-pre"},
		Func: func(_ context.Context, _ PreHookParams) error {
			order = append(order, "csA-pre")
			return nil
		},
	}
	csBPost := PostHook{
		HookDefinition: HookDefinition{Name: "csB-post"},
		Func: func(_ context.Context, _ PostHookParams) error {
			order = append(order, "csB-post")
			return nil
		},
	}

	csA := Configure(MyChangeSet).With("cfgA").WithPreHooks(csAPre)
	csB := Configure(MyChangeSet).With("cfgB").WithPostHooks(csBPost)

	r := NewChangesetsRegistry()
	r.SetValidate(false)
	r.AddGlobalPreHooks(globalPre)
	r.AddGlobalPostHooks(globalPost)
	r.Add("csA", csA)
	r.Add("csB", csB)

	_, err := r.Apply("csA", hookTestEnv(t))
	require.NoError(t, err)

	_, err = r.Apply("csB", hookTestEnv(t))
	require.NoError(t, err)

	expected := []string{
		"global-pre:csA", "csA-pre", "global-post:csA",
		"global-pre:csB", "csB-post", "global-post:csB",
	}
	assert.Equal(t, expected, order,
		"global hooks should run for every changeset; per-CS hooks only for their own")
}

func Test_Integration_MixedAbortWarn_GlobalAndPerCS(t *testing.T) {
	t.Parallel()

	var order []string

	r := NewChangesetsRegistry()
	r.SetValidate(false)

	r.AddGlobalPreHooks(PreHook{
		HookDefinition: HookDefinition{Name: "global-warn-pre", FailurePolicy: Warn},
		Func: func(_ context.Context, _ PreHookParams) error {
			order = append(order, "global-warn-pre")
			return errors.New("global warning")
		},
	})

	csWithAbort := Configure(MyChangeSet).With("cfg").
		WithPreHooks(PreHook{
			HookDefinition: HookDefinition{Name: "cs-abort-pre", FailurePolicy: Abort},
			Func: func(_ context.Context, _ PreHookParams) error {
				order = append(order, "cs-abort-pre")
				return errors.New("abort this")
			},
		})

	csClean := Configure(MyChangeSet).With("cfg").
		WithPostHooks(PostHook{
			HookDefinition: HookDefinition{Name: "cs-warn-post", FailurePolicy: Warn},
			Func: func(_ context.Context, _ PostHookParams) error {
				order = append(order, "cs-warn-post")
				return errors.New("post warning")
			},
		})

	r.Add("cs-abort", csWithAbort)
	r.Add("cs-clean", csClean)

	_, err := r.Apply("cs-abort", hookTestEnv(t))
	require.Error(t, err)
	require.ErrorContains(t, err, "abort this")

	assert.Equal(t, []string{"global-warn-pre", "cs-abort-pre"}, order,
		"global Warn pre-hook should run and be swallowed, then per-CS Abort should fail")

	order = nil
	_, err = r.Apply("cs-clean", hookTestEnv(t))
	require.NoError(t, err)

	assert.Equal(t, []string{"global-warn-pre", "cs-warn-post"}, order,
		"global Warn pre-hook swallowed, Apply succeeds, per-CS Warn post-hook swallowed")
}

func Test_Integration_HooksCoexistWithThenWith(t *testing.T) {
	t.Parallel()

	var order []string

	postProcessed := false

	cs := Configure(MyChangeSet).With("cfg").
		WithPreHooks(PreHook{
			HookDefinition: HookDefinition{Name: "pre"},
			Func: func(_ context.Context, _ PreHookParams) error {
				order = append(order, "pre")
				return nil
			},
		}).
		ThenWith(func(_ fdeployment.Environment, o fdeployment.ChangesetOutput) (fdeployment.ChangesetOutput, error) {
			postProcessed = true
			order = append(order, "post-processor")

			return o, nil
		}).
		WithPostHooks(PostHook{
			HookDefinition: HookDefinition{Name: "post"},
			Func: func(_ context.Context, _ PostHookParams) error {
				order = append(order, "post")
				return nil
			},
		})

	r := NewChangesetsRegistry()
	r.SetValidate(false)
	r.AddGlobalPreHooks(PreHook{
		HookDefinition: HookDefinition{Name: "global-pre"},
		Func: func(_ context.Context, _ PreHookParams) error {
			order = append(order, "global-pre")
			return nil
		},
	})
	r.AddGlobalPostHooks(PostHook{
		HookDefinition: HookDefinition{Name: "global-post"},
		Func: func(_ context.Context, _ PostHookParams) error {
			order = append(order, "global-post")
			return nil
		},
	})
	r.Add("cs-pp", cs)

	_, err := r.Apply("cs-pp", hookTestEnv(t))
	require.NoError(t, err)
	assert.True(t, postProcessed, "ThenWith post-processor should have run")

	expected := []string{"global-pre", "pre", "post-processor", "post", "global-post"}
	assert.Equal(t, expected, order,
		"hooks and ThenWith post-processor should coexist in the correct order")
}

func Test_Apply_HappyPath_WithHooks(t *testing.T) {
	t.Parallel()

	cs := recordingChangeset{}

	preHookRan := false
	postHookRan := false

	r := NewChangesetsRegistry()
	r.SetValidate(false)

	r.AddGlobalPreHooks(PreHook{
		HookDefinition: HookDefinition{Name: "global-pre"},
		Func: func(_ context.Context, _ PreHookParams) error {
			preHookRan = true
			return nil
		},
	})
	r.AddGlobalPostHooks(PostHook{
		HookDefinition: HookDefinition{Name: "global-post"},
		Func: func(_ context.Context, _ PostHookParams) error {
			postHookRan = true
			return nil
		},
	})

	r.Add("test-cs", &cs)

	got, err := r.Apply("test-cs", hookTestEnv(t))
	require.NoError(t, err)
	assert.Equal(t, fdeployment.ChangesetOutput{}, got)
	assert.True(t, cs.applyCalled, "changeset should have been called")
	assert.True(t, preHookRan, "pre-hook should have run")
	assert.True(t, postHookRan, "post-hook should have run")
}

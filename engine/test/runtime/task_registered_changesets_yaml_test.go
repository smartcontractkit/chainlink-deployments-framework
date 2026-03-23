package runtime

import (
	"context"
	"errors"
	"testing"

	"github.com/Masterminds/semver/v3"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/require"

	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
)

func TestRegisteredChangesetsTask(t *testing.T) {
	t.Parallel()

	t.Run("executes changesets in YAML order with per-entry input", func(t *testing.T) {
		t.Parallel()

		rt, err := New(t.Context())
		require.NoError(t, err)

		var calls []recordedCall
		registerCalls := 0
		provider := newTestRegistryProvider(func(registry *changeset.ChangesetsRegistry) {
			registerCalls++
			registry.Add(
				"first",
				changeset.Configure(makeChangeset("first", &calls)).WithEnvInput(),
			)
			registry.Add(
				"second",
				changeset.Configure(makeChangeset("second", &calls)).WithEnvInput(),
			)
		})

		inputYAML := []byte(`environment: testnet
domain: opdev
changesets:
  - first:
      payload:
        value: 1
  - second:
      payload:
        value: 2
`)

		task := RegisteredChangesetsTask(staticProviderFactory(provider), inputYAML)
		err = rt.Exec(task)
		require.NoError(t, err)
		require.Equal(t, []recordedCall{
			{Name: "first", Value: 1},
			{Name: "second", Value: 2},
		}, calls)
		require.Equal(t, 1, registerCalls)
		require.Len(t, rt.State().Outputs, 2)
		require.Contains(t, rt.State().Outputs, task.ID()+"-first-0")
		require.Contains(t, rt.State().Outputs, task.ID()+"-second-1")
	})

	t.Run("supports resolver-backed registered changesets", func(t *testing.T) {
		t.Parallel()

		rt, err := New(t.Context())
		require.NoError(t, err)

		var calls []recordedCall
		provider := newTestRegistryProvider(func(registry *changeset.ChangesetsRegistry) {
			resolver := func(input runtimeResolverInput) (runtimeYAMLInput, error) {
				return runtimeYAMLInput{Value: input.Base + 10}, nil
			}
			registry.Add(
				"resolved",
				changeset.Configure(makeChangeset("resolved", &calls)).
					WithConfigResolver(resolver),
			)
		})

		inputYAML := []byte(`environment: testnet
domain: opdev
changesets:
  - resolved:
      payload:
        base: 7
`)

		task := RegisteredChangesetsTask(staticProviderFactory(provider), inputYAML)
		err = rt.Exec(task)
		require.NoError(t, err)
		require.Equal(t, []recordedCall{
			{Name: "resolved", Value: 17},
		}, calls)
	})

	t.Run("propagates state to next changeset in YAML order", func(t *testing.T) {
		t.Parallel()

		rt, err := New(t.Context())
		require.NoError(t, err)

		const propagatedAddress = "0x9999999999999999999999999999999999999999"
		sawPropagatedState := false

		provider := newTestRegistryProvider(func(registry *changeset.ChangesetsRegistry) {
			registry.Add(
				"first",
				changeset.Configure(fdeployment.CreateChangeSet(
					func(e fdeployment.Environment, cfg runtimeYAMLInput) (fdeployment.ChangesetOutput, error) {
						ds := fdatastore.NewMemoryDataStore()
						addErr := ds.Addresses().Add(fdatastore.AddressRef{
							ChainSelector: chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector,
							Address:       propagatedAddress,
							Type:          "StatePropagationContract",
							Version:       semver.MustParse("1.0.0"),
						})
						require.NoError(t, addErr)

						return fdeployment.ChangesetOutput{DataStore: ds}, nil
					},
					func(e fdeployment.Environment, cfg runtimeYAMLInput) error {
						return nil
					},
				)).WithEnvInput(),
			)
			registry.Add(
				"second",
				changeset.Configure(fdeployment.CreateChangeSet(
					func(e fdeployment.Environment, cfg runtimeYAMLInput) (fdeployment.ChangesetOutput, error) {
						addrs, fetchErr := e.DataStore.Addresses().Fetch()
						require.NoError(t, fetchErr)

						for _, addr := range addrs {
							if addr.Address == propagatedAddress {
								sawPropagatedState = true
								break
							}
						}
						if !sawPropagatedState {
							return fdeployment.ChangesetOutput{}, errors.New("expected first changeset state in second changeset environment")
						}

						return fdeployment.ChangesetOutput{}, nil
					},
					func(e fdeployment.Environment, cfg runtimeYAMLInput) error {
						return nil
					},
				)).WithEnvInput(),
			)
		})

		err = rt.Exec(RegisteredChangesetsTask(staticProviderFactory(provider), []byte(`environment: testnet
domain: opdev
changesets:
  - first:
      payload:
        value: 1
  - second:
      payload:
        value: 2
`)))
		require.NoError(t, err)
		require.True(t, sawPropagatedState)
	})

	t.Run("skips pre and post hooks when applying registered changesets", func(t *testing.T) {
		t.Parallel()

		rt, err := New(t.Context())
		require.NoError(t, err)

		var calls []recordedCall
		provider := newTestRegistryProvider(func(registry *changeset.ChangesetsRegistry) {
			registry.AddGlobalPreHooks(changeset.PreHook{
				HookDefinition: changeset.HookDefinition{Name: "global-pre", FailurePolicy: changeset.Abort},
				Func: func(_ context.Context, _ changeset.PreHookParams) error {
					return errors.New("global pre hook should not run")
				},
			})
			registry.AddGlobalPostHooks(changeset.PostHook{
				HookDefinition: changeset.HookDefinition{Name: "global-post", FailurePolicy: changeset.Abort},
				Func: func(_ context.Context, _ changeset.PostHookParams) error {
					return errors.New("global post hook should not run")
				},
			})
			registry.Add(
				"first",
				changeset.Configure(makeChangeset("first", &calls)).
					WithEnvInput().
					WithPreHooks(changeset.PreHook{
						HookDefinition: changeset.HookDefinition{Name: "pre", FailurePolicy: changeset.Abort},
						Func: func(_ context.Context, _ changeset.PreHookParams) error {
							return errors.New("pre hook should not run")
						},
					}).
					WithPostHooks(changeset.PostHook{
						HookDefinition: changeset.HookDefinition{Name: "post", FailurePolicy: changeset.Abort},
						Func: func(_ context.Context, _ changeset.PostHookParams) error {
							return errors.New("post hook should not run")
						},
					}),
			)
		})

		err = rt.Exec(RegisteredChangesetsTask(staticProviderFactory(provider), []byte(`environment: testnet
domain: opdev
changesets:
  - first:
      payload:
        value: 3
`)))
		require.NoError(t, err)
		require.Equal(t, []recordedCall{{Name: "first", Value: 3}}, calls)
	})

	t.Run("executes hooks when WithExecuteHooks is set", func(t *testing.T) {
		t.Parallel()

		rt, err := New(t.Context())
		require.NoError(t, err)

		var calls []recordedCall
		globalPreCalls := 0
		globalPostCalls := 0
		preCalls := 0
		postCalls := 0
		provider := newTestRegistryProvider(func(registry *changeset.ChangesetsRegistry) {
			registry.AddGlobalPreHooks(changeset.PreHook{
				HookDefinition: changeset.HookDefinition{Name: "global-pre", FailurePolicy: changeset.Abort},
				Func: func(_ context.Context, _ changeset.PreHookParams) error {
					globalPreCalls++
					return nil
				},
			})
			registry.AddGlobalPostHooks(changeset.PostHook{
				HookDefinition: changeset.HookDefinition{Name: "global-post", FailurePolicy: changeset.Abort},
				Func: func(_ context.Context, _ changeset.PostHookParams) error {
					globalPostCalls++
					return nil
				},
			})
			registry.Add(
				"first",
				changeset.Configure(makeChangeset("first", &calls)).
					WithEnvInput().
					WithPreHooks(changeset.PreHook{
						HookDefinition: changeset.HookDefinition{Name: "pre", FailurePolicy: changeset.Abort},
						Func: func(_ context.Context, _ changeset.PreHookParams) error {
							preCalls++
							return nil
						},
					}).
					WithPostHooks(changeset.PostHook{
						HookDefinition: changeset.HookDefinition{Name: "post", FailurePolicy: changeset.Abort},
						Func: func(_ context.Context, _ changeset.PostHookParams) error {
							postCalls++
							return nil
						},
					}),
			)
		})

		err = rt.Exec(RegisteredChangesetsTask(staticProviderFactory(provider), []byte(`environment: testnet
domain: opdev
changesets:
  - first:
      payload:
        value: 5
`), WithExecuteHooks()))
		require.NoError(t, err)
		require.Equal(t, []recordedCall{{Name: "first", Value: 5}}, calls)
		require.Equal(t, 1, globalPreCalls)
		require.Equal(t, 1, globalPostCalls)
		require.Equal(t, 1, preCalls)
		require.Equal(t, 1, postCalls)
	})

	t.Run("can execute as an Executable task", func(t *testing.T) {
		t.Parallel()

		rt, err := New(t.Context())
		require.NoError(t, err)

		var calls []recordedCall
		provider := newTestRegistryProvider(func(registry *changeset.ChangesetsRegistry) {
			registry.Add(
				"first",
				changeset.Configure(makeChangeset("first", &calls)).WithEnvInput(),
			)
			registry.Add(
				"second",
				changeset.Configure(makeChangeset("second", &calls)).WithEnvInput(),
			)
		})

		inputYAML := []byte(`environment: testnet
domain: opdev
changesets:
  - first:
      payload:
        value: 11
  - second:
      payload:
        value: 22
`)

		task := RegisteredChangesetsTask(staticProviderFactory(provider), inputYAML)
		err = rt.Exec(task)
		require.NoError(t, err)
		require.Equal(t, []recordedCall{
			{Name: "first", Value: 11},
			{Name: "second", Value: 22},
		}, calls)
		require.Contains(t, rt.State().Outputs, task.ID()+"-first-0")
		require.Contains(t, rt.State().Outputs, task.ID()+"-second-1")
	})

	t.Run("returns error for invalid YAML changeset payload", func(t *testing.T) {
		t.Parallel()

		rt, err := New(t.Context())
		require.NoError(t, err)

		inputYAML := []byte(`environment: testnet
domain: opdev
changesets:
  - first:
      notPayload:
        value: 1
`)

		err = rt.Exec(RegisteredChangesetsTask(staticProviderFactory(newTestRegistryProvider(nil)), inputYAML))
		require.ErrorContains(t, err, "is missing required 'payload' field")
	})

	t.Run("returns error for invalid changesets array entry", func(t *testing.T) {
		t.Parallel()

		rt, err := New(t.Context())
		require.NoError(t, err)

		inputYAML := []byte(`environment: testnet
domain: opdev
changesets:
  - first:
      payload:
        value: 1
  - bad
`)

		err = rt.Exec(RegisteredChangesetsTask(staticProviderFactory(newTestRegistryProvider(nil)), inputYAML))
		require.ErrorContains(t, err, "invalid changesets array in input YAML")
		require.ErrorContains(t, err, "expected single-key object")
	})

	t.Run("returns error when provider factory is nil", func(t *testing.T) {
		t.Parallel()

		rt, err := New(t.Context())
		require.NoError(t, err)

		err = rt.Exec(RegisteredChangesetsTask(nil, []byte(`environment: testnet
domain: opdev
changesets:
  - first:
      payload:
        value: 1
`)))
		require.ErrorContains(t, err, "provider factory is required")
	})

	t.Run("returns error when provider factory returns nil provider", func(t *testing.T) {
		t.Parallel()

		rt, err := New(t.Context())
		require.NoError(t, err)

		err = rt.Exec(RegisteredChangesetsTask(
			func() changeset.RegistryProvider { return nil },
			[]byte(`environment: testnet
domain: opdev
changesets:
  - first:
      payload:
        value: 1
`),
		))
		require.ErrorContains(t, err, "provider is required")
	})

	t.Run("returns error for invalid YAML syntax", func(t *testing.T) {
		t.Parallel()

		rt, err := New(t.Context())
		require.NoError(t, err)

		err = rt.Exec(RegisteredChangesetsTask(
			staticProviderFactory(newTestRegistryProvider(nil)),
			[]byte("environment: [\n"),
		))
		require.ErrorContains(t, err, "failed to parse input YAML")
	})

	t.Run("returns error for empty changesets array", func(t *testing.T) {
		t.Parallel()

		rt, err := New(t.Context())
		require.NoError(t, err)

		err = rt.Exec(RegisteredChangesetsTask(
			staticProviderFactory(newTestRegistryProvider(nil)),
			[]byte(`environment: testnet
domain: opdev
changesets: []
`),
		))
		require.ErrorContains(t, err, "input YAML has empty 'changesets' array")
	})

	t.Run("returns apply error with changeset name and index", func(t *testing.T) {
		t.Parallel()

		rt, err := New(t.Context())
		require.NoError(t, err)
		var calls []recordedCall

		provider := newTestRegistryProvider(func(registry *changeset.ChangesetsRegistry) {
			registry.Add(
				"first",
				changeset.Configure(makeChangeset("first", &calls)).WithEnvInput(),
			)
			registry.Add(
				"second",
				changeset.Configure(fdeployment.CreateChangeSet(
					func(e fdeployment.Environment, cfg runtimeYAMLInput) (fdeployment.ChangesetOutput, error) {
						return fdeployment.ChangesetOutput{}, errors.New("boom")
					},
					func(e fdeployment.Environment, cfg runtimeYAMLInput) error {
						return nil
					},
				)).WithEnvInput(),
			)
		})

		err = rt.Exec(RegisteredChangesetsTask(staticProviderFactory(provider), []byte(`environment: testnet
domain: opdev
changesets:
  - first:
      payload:
        value: 1
  - second:
      payload:
        value: 2
`)))
		require.ErrorContains(t, err, `failed to apply changeset "second" at index 1`)
		require.ErrorContains(t, err, "boom")
	})
}

func TestRegisteredChangesetsTask_ExplicitYAMLInputTakesPrecedenceOverEnvVar(t *testing.T) {
	t.Setenv("DURABLE_PIPELINE_INPUT", `{"payload":{"value":999}}`)

	rt, err := New(t.Context())
	require.NoError(t, err)
	var calls []recordedCall

	provider := newTestRegistryProvider(func(registry *changeset.ChangesetsRegistry) {
		registry.Add(
			"first",
			changeset.Configure(makeChangeset("first", &calls)).WithEnvInput(),
		)
	})

	err = rt.Exec(RegisteredChangesetsTask(staticProviderFactory(provider), []byte(`environment: testnet
domain: opdev
changesets:
  - first:
      payload:
        value: 1
`)))
	require.NoError(t, err)
	require.Equal(t, []recordedCall{{Name: "first", Value: 1}}, calls)
}

type runtimeYAMLInput struct {
	Value int `json:"value"`
}

type runtimeResolverInput struct {
	Base int `json:"base"`
}

type recordedCall struct {
	Name  string
	Value int
}

type testRegistryProvider struct {
	*changeset.BaseRegistryProvider
	register func(registry *changeset.ChangesetsRegistry)
}

func newTestRegistryProvider(register func(registry *changeset.ChangesetsRegistry)) *testRegistryProvider {
	return &testRegistryProvider{
		BaseRegistryProvider: changeset.NewBaseRegistryProvider(),
		register:             register,
	}
}

func (p *testRegistryProvider) Init() error {
	if p.register != nil {
		p.register(p.Registry())
	}

	return nil
}

func (p *testRegistryProvider) Archive() {}

func staticProviderFactory(provider changeset.RegistryProvider) RegistryProviderFactory {
	return func() changeset.RegistryProvider {
		return provider
	}
}

func makeChangeset(name string, out *[]recordedCall) fdeployment.ChangeSetV2[runtimeYAMLInput] {
	return fdeployment.CreateChangeSet(
		func(e fdeployment.Environment, cfg runtimeYAMLInput) (fdeployment.ChangesetOutput, error) {
			*out = append(*out, recordedCall{Name: name, Value: cfg.Value})
			return fdeployment.ChangesetOutput{}, nil
		},
		func(e fdeployment.Environment, cfg runtimeYAMLInput) error {
			return nil
		},
	)
}

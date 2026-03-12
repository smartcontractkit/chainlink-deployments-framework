package runtime

import (
	"testing"

	"github.com/stretchr/testify/require"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
)

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

func makeInputChangeset(name string, out *[]recordedCall) fdeployment.ChangeSetV2[runtimeYAMLInput] {
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

func makeResolverChangeset(name string, out *[]recordedCall) fdeployment.ChangeSetV2[runtimeYAMLInput] {
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

//nolint:paralleltest // One subtest intentionally uses process env for precedence assertions.
func TestRuntime_ExecRegisteredChangesetsFromYAML(t *testing.T) {
	t.Run("executes changesets in YAML order with per-entry input", func(t *testing.T) {
		t.Parallel()

		rt, err := New(t.Context())
		require.NoError(t, err)

		var calls []recordedCall
		providerFactory := func() changeset.RegistryProvider {
			return newTestRegistryProvider(func(registry *changeset.ChangesetsRegistry) {
				registry.SetValidate(false)
				registry.Add(
					"first",
					changeset.Configure(makeInputChangeset("first", &calls)).WithEnvInput(),
				)
				registry.Add(
					"second",
					changeset.Configure(makeInputChangeset("second", &calls)).WithEnvInput(),
				)
			})
		}

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

		err = rt.ExecRegisteredChangesetsFromYAML(providerFactory, inputYAML)
		require.NoError(t, err)
		require.Equal(t, []recordedCall{
			{Name: "first", Value: 1},
			{Name: "second", Value: 2},
		}, calls)
		require.Len(t, rt.State().Outputs, 2)
	})

	t.Run("supports resolver-backed registered changesets", func(t *testing.T) {
		t.Parallel()

		rt, err := New(t.Context())
		require.NoError(t, err)

		var calls []recordedCall
		providerFactory := func() changeset.RegistryProvider {
			return newTestRegistryProvider(func(registry *changeset.ChangesetsRegistry) {
				registry.SetValidate(false)
				resolver := func(input runtimeResolverInput) (runtimeYAMLInput, error) {
					return runtimeYAMLInput{Value: input.Base + 10}, nil
				}
				registry.Add(
					"resolved",
					changeset.Configure(makeResolverChangeset("resolved", &calls)).
						WithConfigResolver(resolver),
				)
			})
		}

		inputYAML := []byte(`environment: testnet
domain: opdev
changesets:
  - resolved:
      payload:
        base: 7
`)

		err = rt.ExecRegisteredChangesetsFromYAML(providerFactory, inputYAML)
		require.NoError(t, err)
		require.Equal(t, []recordedCall{
			{Name: "resolved", Value: 17},
		}, calls)
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

		err = rt.ExecRegisteredChangesetsFromYAML(func() changeset.RegistryProvider {
			return newTestRegistryProvider(nil)
		}, inputYAML)
		require.ErrorContains(t, err, "is missing required 'payload' field")
	})

	t.Run("returns error when provider factory is nil", func(t *testing.T) {
		t.Parallel()

		rt, err := New(t.Context())
		require.NoError(t, err)

		err = rt.ExecRegisteredChangesetsFromYAML(nil, []byte(`environment: testnet
domain: opdev
changesets:
  - first:
      payload:
        value: 1
`))
		require.ErrorContains(t, err, "provider factory is required")
	})

	t.Run("explicit YAML input takes precedence over env var", func(t *testing.T) {
		t.Setenv("DURABLE_PIPELINE_INPUT", `{"payload":{"value":999}}`)

		rt, err := New(t.Context())
		require.NoError(t, err)
		var calls []recordedCall

		providerFactory := func() changeset.RegistryProvider {
			return newTestRegistryProvider(func(registry *changeset.ChangesetsRegistry) {
				registry.SetValidate(false)
				registry.Add(
					"first",
					changeset.Configure(makeInputChangeset("first", &calls)).WithEnvInput(),
				)
			})
		}

		err = rt.ExecRegisteredChangesetsFromYAML(providerFactory, []byte(`environment: testnet
domain: opdev
changesets:
  - first:
      payload:
        value: 1
`))
		require.NoError(t, err)
		require.Equal(t, []recordedCall{{Name: "first", Value: 1}}, calls)
	})
}

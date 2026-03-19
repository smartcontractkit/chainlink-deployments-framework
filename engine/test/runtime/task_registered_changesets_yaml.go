package runtime

import (
	"errors"
	"fmt"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/pipeline/input"
)

type registeredChangesetsConfig struct {
	executeHooks bool
}

// RegistryProviderFactory returns a registry provider instance.
type RegistryProviderFactory func() changeset.RegistryProvider

// RegisteredChangesetsOption configures RegisteredChangesetsTask behavior.
type RegisteredChangesetsOption func(*registeredChangesetsConfig)

// WithExecuteHooks enables pre/post hook execution while applying registered
// changesets. By default, hooks are not executed.
func WithExecuteHooks() RegisteredChangesetsOption {
	return func(cfg *registeredChangesetsConfig) {
		cfg.executeHooks = true
	}
}

var _ Executable = &registeredChangesetsTask{}

// RegisteredChangesetsTask creates an executable task that applies
// registered changesets from pipeline YAML input.
// By default, hooks are not executed.
//
// Example:
//
//	factory := func() changeset.RegistryProvider { return NewProvider() }
//	err := rt.Exec(RegisteredChangesetsTask(factory, inputYAML))
//	err := rt.Exec(RegisteredChangesetsTask(factory, inputYAML, WithExecuteHooks()))
func RegisteredChangesetsTask(
	providerFactory RegistryProviderFactory,
	inputYAML []byte,
	opts ...RegisteredChangesetsOption,
) registeredChangesetsTask {
	cfg := registeredChangesetsConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	return registeredChangesetsTask{
		baseTask:        newBaseTask(),
		providerFactory: providerFactory,
		inputYAML:       inputYAML,
		cfg:             cfg,
	}
}

type registeredChangesetsTask struct {
	*baseTask

	providerFactory RegistryProviderFactory
	inputYAML       []byte
	cfg             registeredChangesetsConfig
}

func (t registeredChangesetsTask) Run(e fdeployment.Environment, state *State) error {
	if t.providerFactory == nil {
		return errors.New("provider factory is required")
	}

	provider := t.providerFactory()
	if provider == nil {
		return errors.New("provider is required")
	}

	parsed, err := input.ParseYAMLBytes(t.inputYAML)
	if err != nil {
		return fmt.Errorf("failed to parse input YAML: %w", err)
	}

	ordered, err := input.GetAllChangesetsInOrder(parsed.Changesets)
	if err != nil {
		return fmt.Errorf("invalid changesets array in input YAML: %w", err)
	}
	if len(ordered) == 0 {
		return errors.New("input YAML has empty 'changesets' array")
	}

	provider.Registry().SetValidate(false)

	if initErr := provider.Init(); initErr != nil {
		return fmt.Errorf("failed to init registry provider: %w", initErr)
	}

	currentEnv := e
	for i, cs := range ordered {
		inputJSON, err := input.BuildChangesetInputJSON(cs.Name, cs.Data)
		if err != nil {
			return fmt.Errorf("failed to build input for changeset %q: %w", cs.Name, err)
		}

		var out fdeployment.ChangesetOutput
		if t.cfg.executeHooks {
			out, err = provider.Registry().Apply(cs.Name, currentEnv, changeset.WithInput(inputJSON))
		} else {
			out, err = provider.Registry().Apply(
				cs.Name,
				currentEnv,
				changeset.WithInput(inputJSON),
				changeset.WithoutHooks(),
			)
		}
		if err != nil {
			return fmt.Errorf("failed to apply changeset %q at index %d: %w", cs.Name, i, err)
		}

		if err := state.MergeChangesetOutput(
			fmt.Sprintf("%s-%s-%d", t.ID(), cs.Name, i),
			out,
		); err != nil {
			return fmt.Errorf("failed to merge output for changeset %q at index %d: %w", cs.Name, i, err)
		}

		currentEnv = newEnvFromState(currentEnv, state)
	}

	return nil
}

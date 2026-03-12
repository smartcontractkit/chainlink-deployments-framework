package runtime

import (
	"errors"
	"fmt"

	"github.com/segmentio/ksuid"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/durablepipeline"
)

// ExecRegisteredChangesetsFromYAML executes registered changesets from durable-pipeline YAML input.
//
// For each changeset entry in YAML order:
//  1. Build per-entry durable-pipeline JSON input from payload/chainOverrides.
//  2. Create and initialize a fresh registry provider.
//  3. Apply the named changeset against the current runtime environment with explicit input.
//  4. Merge output into runtime state and regenerate environment for the next step.
func (r *Runtime) ExecRegisteredChangesetsFromYAML(
	providerFactory func() changeset.RegistryProvider,
	inputYAML []byte,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if providerFactory == nil {
		return errors.New("provider factory is required")
	}

	parsed, err := durablepipeline.ParseYAMLBytes(inputYAML)
	if err != nil {
		return fmt.Errorf("failed to parse input file %s: %w", "runtime-input.yaml", err)
	}

	ordered, err := durablepipeline.GetAllChangesetsInOrder(parsed.Changesets)
	if err != nil {
		return fmt.Errorf("input file %s: %w", "runtime-input.yaml", err)
	}
	if len(ordered) == 0 {
		return fmt.Errorf("input file %s has empty 'changesets' array", "runtime-input.yaml")
	}

	for _, cs := range ordered {
		inputJSON, err := durablepipeline.BuildChangesetInputJSON(cs.Name, cs.Data)
		if err != nil {
			return fmt.Errorf("failed to build input for changeset %q in input file %s: %w", cs.Name, "runtime-input.yaml", err)
		}

		provider := providerFactory()
		if provider == nil {
			return errors.New("provider factory returned nil")
		}
		if initErr := provider.Init(); initErr != nil {
			return fmt.Errorf("failed to init registry provider: %w", initErr)
		}

		out, err := provider.Registry().ApplyWithInput(cs.Name, r.currentEnv, inputJSON)
		if err != nil {
			return err
		}

		if err := r.state.MergeChangesetOutput(
			fmt.Sprintf("registered-%s-%s", cs.Name, ksuid.New().String()),
			out,
		); err != nil {
			return err
		}

		r.currentEnv = r.generateNewEnvironment()
	}

	return nil
}

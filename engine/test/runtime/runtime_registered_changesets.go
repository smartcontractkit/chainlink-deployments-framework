package runtime

import (
	"errors"
	"fmt"
	"os"

	"github.com/segmentio/ksuid"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/durablepipeline"
)

// ExecRegisteredChangesetsFromYAML executes registered changesets from durable-pipeline YAML input.
//
// For each changeset entry in YAML order:
//  1. Set DURABLE_PIPELINE_INPUT from that entry's payload/chainOverrides.
//  2. Create and initialize a fresh registry provider.
//  3. Apply the named changeset against the current runtime environment.
//  4. Merge output into runtime state and regenerate environment for the next step.
//
// Do not run this in parallel. This is not thread-safe. It temporarily mutates the process-wide
// DURABLE_PIPELINE_INPUT environment variable while applying each changeset.
// Once we move the reliance on the environment variable in the implementation, we can remove this restriction.
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

	oldInput, hadInput := os.LookupEnv("DURABLE_PIPELINE_INPUT")
	defer func() {
		if hadInput {
			_ = os.Setenv("DURABLE_PIPELINE_INPUT", oldInput)
		} else {
			_ = os.Unsetenv("DURABLE_PIPELINE_INPUT")
		}
	}()

	for _, cs := range ordered {
		inputJSON, err := durablepipeline.BuildChangesetInputJSON(cs.Name, cs.Data)
		if err != nil {
			return fmt.Errorf("failed to build input for changeset %q in input file %s: %w", cs.Name, "runtime-input.yaml", err)
		}
		if setEnvErr := os.Setenv("DURABLE_PIPELINE_INPUT", inputJSON); setEnvErr != nil {
			return fmt.Errorf("failed to set DURABLE_PIPELINE_INPUT environment variable: %w", setEnvErr)
		}

		provider := providerFactory()
		if provider == nil {
			return errors.New("provider factory returned nil")
		}
		if initErr := provider.Init(); initErr != nil {
			return fmt.Errorf("failed to init registry provider: %w", initErr)
		}

		out, err := provider.Registry().Apply(cs.Name, r.currentEnv)
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

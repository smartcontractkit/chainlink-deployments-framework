package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
)

// NewStateCmds creates a new set of commands for state environment.
func (c Commands) NewStateCmds(dom domain.Domain, config StateConfig) *cobra.Command {
	stateCmd := &cobra.Command{
		Use:   "state",
		Short: "State commands",
	}
	stateCmd.AddCommand(c.newStateGenerate(dom, config))

	stateCmd.PersistentFlags().
		StringP("environment", "e", "", "Deployment environment (required)")
	_ = stateCmd.MarkPersistentFlagRequired("environment")

	return stateCmd
}

type StateConfig struct {
	ViewState cldf.ViewStateV2
}

func (c Commands) newStateGenerate(dom domain.Domain, cfg StateConfig) *cobra.Command {
	var (
		persist       bool
		outputPath    string
		prevStatePath string
	)

	cmd := cobra.Command{
		Use:   "generate",
		Short: "Generate latest state. Nodes must be present in the `nodes.json` to be included.",
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			envdir := dom.EnvDir(envKey)
			viewTimeout := 10 * time.Minute

			cmd.Printf("Generate latest state for %s in environment: %s\n", dom, envKey)
			cmd.Printf("This command may take a while to complete, please be patient. Timeout set to %v\n", viewTimeout)
			ctx, cancel := context.WithTimeout(cmd.Context(), viewTimeout)
			defer cancel()

			ctxFunc := func() context.Context {
				return ctx
			}

			env, err := environment.Load(ctxFunc, c.lggr, envKey, dom, true /*useRealBackends*/)
			if err != nil {
				return fmt.Errorf("failed to load environment %w", err)
			}

			prevState, err := envdir.LoadState()
			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to load previous state: %w", err)
			}

			state, err := cfg.ViewState(env, prevState)
			if err != nil {
				return fmt.Errorf("unable to snapshot state: %w", err)
			}

			b, err := state.MarshalJSON()
			if err != nil {
				return fmt.Errorf("unable to marshal state: %w", err)
			}

			if persist {
				// Save the state to the outputPath if defined, otherwise save it with the default
				// path in the product and environment directory with the default file name.
				if outputPath != "" {
					err = domain.SaveViewState(outputPath, state)
				} else {
					err = envdir.SaveViewState(state)
				}

				if err != nil {
					return fmt.Errorf("failed to save state: %w", err)
				}
			}

			cmd.Println(string(b))

			return nil
		},
	}

	cmd.Flags().BoolVarP(&persist, "persist", "p", false, "Persist state to disk")
	cmd.Flags().StringVarP(&outputPath, "outputPath", "o", "", "Output path. Default is <product>/<environment>/state.json")
	cmd.Flags().StringVarP(&prevStatePath, "previousState", "s", "", "Previous state's path. Default is <product>/<environment>/state.json")

	return &cmd
}

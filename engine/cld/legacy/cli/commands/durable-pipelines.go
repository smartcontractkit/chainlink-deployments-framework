package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	fresolvers "github.com/smartcontractkit/chainlink-deployments-framework/changeset/resolvers"
	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	cs "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	dom "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	cldenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/operations"

	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

// a temporary workaround to allow test to mock the LoadEnvironment function
var loadEnv = cldenv.Load

// TODO: envLoader needs to be refactored to an interface so we can mock it for testing
// to avoid using real backends
func (c Commands) NewDurablePipelineCmds(
	domain dom.Domain,
	loadMigration func(envName string) (*cs.ChangesetsRegistry, error),
	decodeProposalCtxProvider func(env fdeployment.Environment) (analyzer.ProposalContext, error),
	loadConfigResolvers *fresolvers.ConfigResolverManager) *cobra.Command {
	evmCmd := &cobra.Command{
		Use:   "durable-pipeline",
		Short: "Durable Pipeline commands",
	}

	evmCmd.AddCommand(
		c.newDurablePipelineRun(domain, loadMigration, decodeProposalCtxProvider, loadConfigResolvers),
		c.newDurablePipelineInputGenerate(domain, loadMigration, loadConfigResolvers),
		c.newDurablePipelineListBuild(domain, loadMigration, loadConfigResolvers),
		c.newDurablePipelineTemplateInput(domain, loadMigration, loadConfigResolvers))

	evmCmd.PersistentFlags().StringP("environment", "e", "", "Deployment environment (required)")
	_ = evmCmd.MarkPersistentFlagRequired("environment")

	return evmCmd
}

// newDurablePipelineExecute builds the run subcommand for running durable pipeline changesets
var (
	runLong = `
		Run a durable pipeline changeset.

		This command applies a changeset against the specified environment,
		resolves any timelock proposals, and persists artifacts.
`
	runExample = `
		# Dry-run of changeset 0001_test_changeset in testnet
		chainlink-deployments durable-pipeline run \
  		--environment testnet \
  		--changeset 0001_test_changeset \
  		--dry-run

		# Run changeset by name with input file
		chainlink-deployments durable-pipeline run \
  		--environment testnet \
  		--changeset 0001_test_changeset \
  		--input-file inputs.yaml

		# Run changeset by index position with array format input file.
		# Index 0 runs first changeset, index 1 runs second, etc.
		# Only works with array format YAML files, not object format.
		chainlink-deployments durable-pipeline run \
  		--environment testnet \
  		--input-file inputs.yaml \
  		--changeset-index 0
`
)

// newDurablePipelineRun builds the 'run' subcommand for executing durable pipelines
func (c Commands) newDurablePipelineRun(
	domain dom.Domain,
	loadMigration func(envName string) (*cs.ChangesetsRegistry, error),
	decodeProposalCtxProvider func(env fdeployment.Environment) (analyzer.ProposalContext, error),
	loadConfigResolvers *fresolvers.ConfigResolverManager,
) *cobra.Command {
	var (
		changesetStr   string
		dryRun         bool
		inputFile      string
		changesetIndex int
	)

	cmd := &cobra.Command{
		Use:     "run",
		Short:   "Run a durable pipeline changeset",
		Long:    runLong,
		Example: runExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			envdir := domain.EnvDir(envKey)
			artdir := envdir.ArtifactsDir()

			var actualChangesetName string

			// Determine which changeset to run
			if changesetStr != "" {
				// Mode 1: Run specific changeset by name
				actualChangesetName = changesetStr
				if err := setDurablePipelineInputFromYAML(inputFile, changesetStr, domain, envKey); err != nil {
					return fmt.Errorf("failed to parse input file: %w", err)
				}
			} else {
				// Mode 2: Run changeset at specific index position in input file
				var err error
				actualChangesetName, err = setDurablePipelineInputFromYAMLByIndex(inputFile, changesetIndex, domain, envKey)
				if err != nil {
					return fmt.Errorf("failed to get changeset at index %d: %w", changesetIndex, err)
				}
			}

			// Set artifacts directory for durable pipelines
			if err := artdir.SetDurablePipelines(strconv.FormatInt(time.Now().UnixNano(), 10)); err != nil {
				return err
			}

			migration, err := loadMigration(envKey)
			if err != nil {
				return err
			}

			envOptions, err := configureEnvironmentOptions(migration, actualChangesetName, dryRun, c.lggr)
			if err != nil {
				return err
			}

			// Verify that the resolver is registered
			cfg, err := migration.GetConfigurations(actualChangesetName)
			if err != nil {
				return fmt.Errorf("failed to get configurations for %s: %w", actualChangesetName, err)
			}

			if loadConfigResolvers != nil && cfg.ConfigResolver != nil {
				resolverName := loadConfigResolvers.NameOf(cfg.ConfigResolver)
				if resolverName == "" {
					return fmt.Errorf("resolver for %s is not registered", actualChangesetName)
				}
			}

			reports, err := artdir.LoadOperationsReports(actualChangesetName)
			if err != nil {
				return fmt.Errorf("failed to load operations report: %w", err)
			}

			originalReportsLen := len(reports)
			c.lggr.Infof("Loaded %d operations reports", originalReportsLen)
			reporter := operations.NewMemoryReporter(operations.WithReports(reports))

			envOptions = append(envOptions, cldenv.WithReporter(reporter))
			env, err := loadEnv(cmd.Context(), domain, envKey, envOptions...)
			if err != nil {
				return err
			}

			indexStr := ""
			if changesetStr == "" {
				indexStr = fmt.Sprintf(" (at index %d)", changesetIndex)
			}
			// We create the directory even before the attempt to run the changesets.
			// Some migrations may execute ChangeSet functions that only have side effects but not artifacts.
			// In that case we still want to create the directory. We include a .gitkeep file to ensure
			// the directory is not empty.
			c.lggr.Infof("Applying %s durable pipeline for changeset %s%s for environment: %s\n",
				domain, actualChangesetName, indexStr, envKey,
			)

			// Run the changeset for durable pipelines
			out, err := migration.Apply(actualChangesetName, env)
			// save reports first then handle above error
			// 2nd param is set to 0 for now as we are not loading any reports yet
			if saveErr := saveReports(reporter, originalReportsLen, c.lggr, artdir, actualChangesetName); saveErr != nil {
				return saveErr
			}
			if err != nil {
				return err
			}

			if len(out.DescribedTimelockProposals) == 0 && decodeProposalCtxProvider != nil {
				out.DescribedTimelockProposals = make([]string, len(out.MCMSTimelockProposals))
				proposalContext, err := decodeProposalCtxProvider(env)
				if err != nil {
					return err
				}
				for idx, proposal := range out.MCMSTimelockProposals {
					describedProposal, err := analyzer.DescribeTimelockProposal(proposalContext, &proposal)
					if err != nil {
						c.lggr.Errorf("failed to describe time lock proposal %d: %w", idx, err)
						continue
					}
					out.DescribedTimelockProposals[idx] = describedProposal
				}
			}

			// We probably need to save the MCMS proposal since it will need to be signed (generated by the changeset)
			// Probably also need to open a PR with the proposal if one is created
			if err := artdir.SaveChangesetOutput(actualChangesetName, out); err != nil {
				c.lggr.Errorf("failed to save changeset artifacts: %v", err)
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Use a read-only JD backend. WARNING: still uses real chain clients as of now!")
	cmd.Flags().StringVarP(&changesetStr, "changeset", "c", "", "changeset to apply by name")
	cmd.Flags().StringVarP(&inputFile, "input-file", "i", "", "YAML input file name. Not the full path, just the name")
	cmd.Flags().IntVarP(&changesetIndex, "changeset-index", "x", 0, "Index of changeset to run by position in array format input file")

	_ = cmd.MarkFlagRequired("input-file")
	cmd.MarkFlagsMutuallyExclusive("changeset", "changeset-index")
	cmd.MarkFlagsOneRequired("changeset", "changeset-index")

	return cmd
}

// Long and Example for 'input-generate' subcommand
var (
	inputGenerateLong = `
		Generate durable-pipeline input configurations.

		Reads an inputs file, resolves each changeset via registered config fresolvers,
		and outputs the resulting config in YAML or JSON.
`
	inputGenerateExample = `
		# Generate config from inputs.yaml and print
		chainlink-deployments durable-pipeline input-generate \
  			--environment testnet \
  			--inputs inputs.yaml

		# Write JSON output to file
		chainlink-deployments durable-pipeline input-generate \
		  --environment testnet \
		  --inputs inputs.yaml \
		  --json \
		  --output config.json
		`
)

// newDurablePipelineInputGenerate builds the config-generate subcommand for generating
// durable pipeline configurations using config resolvers
func (c Commands) newDurablePipelineInputGenerate(
	domain dom.Domain,
	loadMigrationsRegistry func(envName string) (*cs.ChangesetsRegistry, error),
	loadConfigResolvers *fresolvers.ConfigResolverManager,
) *cobra.Command {
	var (
		inputsFileName string
		outputPath     string
		formatAsJSON   bool
	)

	type durablePipelineFile struct {
		Environment string    `yaml:"environment" json:"environment"`
		Domain      string    `yaml:"domain"      json:"domain"`
		Changesets  yaml.Node `yaml:"changesets" json:"changesets"`
	}

	cmd := cobra.Command{
		Use:     "input-generate",
		Short:   "Generate durable-pipeline input using registered config fresolvers",
		Long:    inputGenerateLong,
		Example: inputGenerateExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			// Load the migrations registry
			registry, err := loadMigrationsRegistry(envKey)
			if err != nil {
				return fmt.Errorf("load migrations registry: %w", err)
			}

			// Read & parse the inputs file
			workspaceRoot, err := findWorkspaceRoot()
			if err != nil {
				return fmt.Errorf("find workspace root: %w", err)
			}
			inputsPath := filepath.Join(
				workspaceRoot, "domains", domain.String(),
				envKey, "durable_pipelines", "inputs", inputsFileName,
			)

			raw, err := os.ReadFile(inputsPath)
			if err != nil {
				return fmt.Errorf("read inputs file: %w", err)
			}

			var dpFile durablePipelineFile
			if err = yaml.Unmarshal(raw, &dpFile); err != nil {
				return fmt.Errorf("parse inputs file (yaml): %w", err)
			}

			// Build changeset to resolver map
			resolverByKey := make(map[string]fresolvers.ConfigResolver)
			for _, key := range registry.ListKeys() {
				var cfg cs.Configurations
				cfg, err = registry.GetConfigurations(key)
				if err != nil {
					return fmt.Errorf("get configurations for %s: %w", key, err)
				}
				res := cfg.ConfigResolver
				if res != nil {
					// Safety check: ensure the resolver is registered
					if loadConfigResolvers.NameOf(res) == "" {
						return fmt.Errorf("resolver for changeset %q is not registered with the resolver manager", key)
					}
					resolverByKey[key] = res
				}
			}

			// Resolve every changeset in the file
			var orderedChangesets []map[string]any // For both formats to preserve order and duplicates

			// Handle both object and array formats for changesets
			//nolint:exhaustive // Only handling MappingNode and SequenceNode cases for changesets
			switch dpFile.Changesets.Kind {
			case yaml.MappingNode:
				// Object format: changesets: { key1: {payload: ...}, key2: {payload: ...} }
				orderedChangesets = make([]map[string]any, 0, len(dpFile.Changesets.Content)/2)
				// yaml.Node for a mapping has Content with alternating key-value pairs
				for i := 0; i < len(dpFile.Changesets.Content); i += 2 {
					keyNode := dpFile.Changesets.Content[i]
					valueNode := dpFile.Changesets.Content[i+1]

					csName := keyNode.Value
					resolver, ok := resolverByKey[csName]
					if !ok {
						resolver = nil // No resolver registered for this changeset
					}

					resolvedCfg, err2 := resolveChangesetConfig(valueNode, csName, resolver)
					if err2 != nil {
						return err2
					}

					// For object format, store each changeset as a separate item (same as array format)
					changesetItem := map[string]any{
						csName: map[string]any{"payload": resolvedCfg},
					}
					orderedChangesets = append(orderedChangesets, changesetItem)
				}
			case yaml.SequenceNode:
				// Array format: changesets: [ { key1: {payload: ...} }, { key2: {payload: ...} } ]
				orderedChangesets = make([]map[string]any, 0, len(dpFile.Changesets.Content))
				for _, itemNode := range dpFile.Changesets.Content {
					if itemNode.Kind != yaml.MappingNode || len(itemNode.Content) < 2 {
						return errors.New("invalid changeset array item format - expected mapping with at least one key-value pair")
					}

					// Each item should be a mapping with one key-value pair (changeset name -> config)
					keyNode := itemNode.Content[0]
					valueNode := itemNode.Content[1]

					csName := keyNode.Value
					resolver, ok := resolverByKey[csName]
					if !ok {
						resolver = nil // No resolver registered for this changeset
					}

					resolvedCfg, err2 := resolveChangesetConfig(valueNode, csName, resolver)
					if err2 != nil {
						return err2
					}

					// For array format, store each changeset as a separate item
					changesetItem := map[string]any{
						csName: map[string]any{"payload": resolvedCfg},
					}
					orderedChangesets = append(orderedChangesets, changesetItem)
				}
			default:
				return fmt.Errorf("changesets must be either an object (mapping) or an array (sequence), got %v", dpFile.Changesets.Kind)
			}

			// Build ordered output structure using yaml.Node to preserve order and original format
			var changesetsNode *yaml.Node

			if dpFile.Changesets.Kind == yaml.MappingNode {
				// Object format: preserve as object
				changesetsNode = &yaml.Node{
					Kind: yaml.MappingNode,
				}

				for _, changesetItem := range orderedChangesets {
					// Each changesetItem has one key-value pair
					for csName, csConfig := range changesetItem {
						// Add key node
						keyNode := &yaml.Node{
							Kind:  yaml.ScalarNode,
							Value: csName,
						}
						changesetsNode.Content = append(changesetsNode.Content, keyNode)

						// Add value node
						valueNode := &yaml.Node{}
						err = valueNode.Encode(csConfig)
						if err != nil {
							return fmt.Errorf("encode changeset value for %s: %w", csName, err)
						}
						changesetsNode.Content = append(changesetsNode.Content, valueNode)

						break // Only one key-value pair per item
					}
				}
			} else {
				// Array format: preserve as array
				changesetsNode = &yaml.Node{
					Kind: yaml.SequenceNode,
				}

				for _, changesetItem := range orderedChangesets {
					// Create a mapping node for each changeset item
					itemNode := &yaml.Node{
						Kind: yaml.MappingNode,
					}

					// Each changesetItem has one key-value pair
					for csName, csConfig := range changesetItem {
						// Add key node
						keyNode := &yaml.Node{
							Kind:  yaml.ScalarNode,
							Value: csName,
						}
						itemNode.Content = append(itemNode.Content, keyNode)

						// Add value node
						valueNode := &yaml.Node{}
						err = valueNode.Encode(csConfig)
						if err != nil {
							return fmt.Errorf("encode changeset value for %s: %w", csName, err)
						}
						itemNode.Content = append(itemNode.Content, valueNode)

						break // Only one key-value pair per item
					}

					changesetsNode.Content = append(changesetsNode.Content, itemNode)
				}
			}

			// Create the final output structure
			finalOutputNode := &yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "environment"},
					{Kind: yaml.ScalarNode, Value: envKey},
					{Kind: yaml.ScalarNode, Value: "domain"},
					{Kind: yaml.ScalarNode, Value: domain.String()},
					{Kind: yaml.ScalarNode, Value: "changesets"},
					changesetsNode,
				},
			}

			// Encode & write
			var outBytes []byte
			if formatAsJSON {
				// For JSON, decode the node to a regular structure and marshal
				var finalOutput map[string]any
				err = finalOutputNode.Decode(&finalOutput)
				if err != nil {
					return fmt.Errorf("decode final output for JSON: %w", err)
				}
				outBytes, err = json.MarshalIndent(finalOutput, "", "  ")
			} else {
				outBytes, err = yaml.Marshal(finalOutputNode)
			}
			if err != nil {
				return fmt.Errorf("encode output: %w", err)
			}

			if outputPath != "" {
				if err := os.WriteFile(outputPath, outBytes, 0o644); err != nil { //nolint:gosec
					return fmt.Errorf("write output file: %w", err)
				}
				format := "YAML"
				if formatAsJSON {
					format = "JSON"
				}
				c.lggr.Infof("Generated %s config written to: %s", format, outputPath)
			} else {
				fmt.Print(string(outBytes))
			}

			return nil
		},
	}

	// CLI flags
	cmd.Flags().StringVarP(&inputsFileName, "inputs", "i", "", "Inputs file name (required)")
	cmd.Flags().BoolVarP(&formatAsJSON, "json", "j", false, "Emit JSON instead of YAML")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path (optional; prints to stdout if omitted)")

	_ = cmd.MarkFlagRequired("inputs")

	return &cmd
}

// Long and Example for 'list' subcommand
var (
	listLong = `
		List durable pipeline info.

		Displays registered changesets (static vs dynamic) and available fresolvers
		for the given environment.
		`
	listExample = `
		# List durable pipeline info for testnet
		chainlink-deployments durable-pipeline list --environment testnet
`
)

// newDurablePipelineListBuild builds the list subcommand for listing durable pipeline info including registered changesets and config fresolvers
func (Commands) newDurablePipelineListBuild(domain dom.Domain, loadMigrationsRegistry func(envName string) (*cs.ChangesetsRegistry, error), loadConfigResolvers *fresolvers.ConfigResolverManager) *cobra.Command {
	cmd := cobra.Command{
		Use:     "list",
		Short:   "List durable pipeline info",
		Long:    listLong,
		Example: listExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			registry, err := loadMigrationsRegistry(envKey)
			if err != nil {
				return fmt.Errorf("failed to load migrations registry: %w", err)
			}

			changesets := registry.ListKeys()

			// Use cmd.OutOrStdout() instead of direct stdout access
			out := cmd.OutOrStdout()

			fmt.Fprintf(out, "\n=== Durable Pipeline Info for %s ===\n", domain.String())

			// Legend
			fmt.Fprintf(out, "\nLegend: DYNAMIC = config resolver | STATIC = YAML input | ERROR = misconfigured\n")

			// Create table writer using command's output
			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)

			fmt.Fprintf(w, "\nRegistered Changesets:\n")
			fmt.Fprintf(w, "TYPE\tNAME\tCONFIG SOURCE\n")
			fmt.Fprintf(w, "----\t----\t-------------\n")

			for _, changeset := range changesets {
				cfg, err := registry.GetConfigurations(changeset)
				if err != nil {
					return fmt.Errorf("get configurations for %s: %w", changeset, err)
				}
				res := cfg.ConfigResolver

				if res == nil {
					fmt.Fprintf(w, "STATIC\t%s\tYAML input file\n", changeset)
				} else {
					resolverName := loadConfigResolvers.NameOf(res)
					if resolverName == "" {
						fmt.Fprintf(w, "ERROR\t%s\tResolver not registered\n", changeset)
					} else {
						parts := strings.Split(resolverName, ".")
						shortName := parts[len(parts)-1]
						fmt.Fprintf(w, "DYNAMIC\t%s\t%s\n", changeset, shortName)
					}
				}
			}

			w.Flush()

			// Available fresolvers
			allResolvers := loadConfigResolvers.ListResolvers()
			fmt.Fprintf(out, "\nAvailable Config Resolvers:\n")
			for _, resolver := range allResolvers {
				parts := strings.Split(resolver, ".")
				shortName := parts[len(parts)-1]
				fmt.Fprintf(out, "  • %s\n", shortName)
			}

			return nil
		},
	}

	return &cmd
}

// Long and Example for 'template-input' subcommand
var (
	longDescription = `
		Generate YAML input templates from Changeset input Go struct types.

		This command helps create YAML input files by analyzing Go struct types
		from changesets and generating properly formatted YAML templates with
		example values and comments.
`
	example = `
		# Generate YAML template for a single changeset
		chainlink-deployments durable-pipeline template-input \
		  --environment testnet \
		  --changeset test_migration_dynamic_inputs

		# Generate YAML template for multiple changesets
		chainlink-deployments durable-pipeline template-input \
		  --environment testnet \
		  --changeset changeset1,changeset2,changeset3

		# Configure depth limit for nested structures
		chainlink-deployments durable-pipeline template-input \
		  --environment testnet \
		  --changeset test_migration_dynamic_inputs \
		  --depth 3

		# Save output to file
		chainlink-deployments durable-pipeline template-input \
		  --environment testnet \
		  --changeset test_migration_dynamic_inputs > example.yaml
		`
)

// newDurablePipelineTemplateInput builds the template-input subcommand for generating
// YAML input templates from Go struct types
func (c Commands) newDurablePipelineTemplateInput(
	domain dom.Domain,
	loadRegistry func(envName string) (*cs.ChangesetsRegistry, error),
	loadConfigResolvers *fresolvers.ConfigResolverManager,
) *cobra.Command {
	var (
		changesetList string
		depthLimit    int
	)

	cmd := cobra.Command{
		Use:     "template-input",
		Short:   "Generate YAML input templates from Changesets",
		Long:    longDescription,
		Example: example,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")

			registry, err := loadRegistry(envKey)
			if err != nil {
				return fmt.Errorf("load registry: %w", err)
			}

			// Parse changeset names (comma-separated)
			changesetNames := strings.Split(strings.TrimSpace(changesetList), ",")
			for i, name := range changesetNames {
				changesetNames[i] = strings.TrimSpace(name)
			}

			yamlTemplate, err := generateMultiChangesetYAMLTemplate(domain.String(), envKey, changesetNames, registry, loadConfigResolvers, depthLimit)
			if err != nil {
				return fmt.Errorf("generate YAML template: %w", err)
			}

			// Use fmt.Fprint with cmd.OutOrStdout() to ensure output goes to stdout
			// and can be properly redirected to files and captured in tests
			fmt.Fprint(cmd.OutOrStdout(), yamlTemplate)

			return nil
		},
	}

	cmd.Flags().StringVarP(&changesetList, "changeset", "c", "", "Changeset name(s) to generate YAML template for - comma-separated for multiple (required)")
	cmd.Flags().IntVarP(&depthLimit, "depth", "d", 5, "Maximum recursion depth generation for nested struct, configure this based on your struct complexity")

	_ = cmd.MarkFlagRequired("changeset")

	return &cmd
}

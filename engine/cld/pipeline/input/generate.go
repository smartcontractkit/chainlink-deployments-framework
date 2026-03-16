package input

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/smartcontractkit/chainlink-deployments-framework/changeset/resolvers"
	cs "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

type durablePipelineFile struct {
	Environment string    `yaml:"environment" json:"environment"`
	Domain      string    `yaml:"domain"      json:"domain"`
	Changesets  yaml.Node `yaml:"changesets" json:"changesets"`
}

// GenerateOptions configures the input-generate operation.
type GenerateOptions struct {
	InputsFileName  string
	Domain          domain.Domain
	EnvKey          string
	Registry        *cs.ChangesetsRegistry
	ResolverManager *resolvers.ConfigResolverManager
	FormatAsJSON    bool
	OutputPath      string // empty = print to stdout
}

// Generate resolves the inputs file and outputs the result.
func Generate(opts GenerateOptions) (string, error) {
	workspaceRoot, err := FindWorkspaceRoot()
	if err != nil {
		return "", fmt.Errorf("find workspace root: %w", err)
	}

	inputsPath := filepath.Join(
		workspaceRoot, "domains", opts.Domain.String(),
		opts.EnvKey, "durable_pipelines", "inputs", opts.InputsFileName,
	)

	raw, err := os.ReadFile(inputsPath)
	if err != nil {
		return "", fmt.Errorf("read inputs file: %w", err)
	}

	var dpFile durablePipelineFile
	if err = yaml.Unmarshal(raw, &dpFile); err != nil {
		return "", fmt.Errorf("parse inputs file (yaml): %w", err)
	}

	resolverByKey := make(map[string]resolvers.ConfigResolver)
	for _, key := range opts.Registry.ListKeys() {
		cfg, getErr := opts.Registry.GetConfigurations(key)
		if getErr != nil {
			return "", fmt.Errorf("get configurations for %s: %w", key, getErr)
		}
		if cfg.ConfigResolver != nil {
			if opts.ResolverManager.NameOf(cfg.ConfigResolver) == "" {
				return "", fmt.Errorf("resolver for changeset %q is not registered with the resolver manager", key)
			}
			resolverByKey[key] = cfg.ConfigResolver
		}
	}

	var orderedChangesets []map[string]any

	//nolint:exhaustive // Only MappingNode and SequenceNode are valid for changesets
	switch dpFile.Changesets.Kind {
	case yaml.MappingNode:
		orderedChangesets = make([]map[string]any, 0, len(dpFile.Changesets.Content)/2)
		for i := 0; i < len(dpFile.Changesets.Content); i += 2 {
			keyNode := dpFile.Changesets.Content[i]
			valueNode := dpFile.Changesets.Content[i+1]

			csName := keyNode.Value
			resolver := resolverByKey[csName]

			resolvedCfg, resolveErr := ResolveChangesetConfig(valueNode, csName, resolver)
			if resolveErr != nil {
				return "", resolveErr
			}

			changesetItem := map[string]any{
				csName: map[string]any{"payload": resolvedCfg},
			}
			orderedChangesets = append(orderedChangesets, changesetItem)
		}
	case yaml.SequenceNode:
		orderedChangesets = make([]map[string]any, 0, len(dpFile.Changesets.Content))
		for _, itemNode := range dpFile.Changesets.Content {
			if itemNode.Kind != yaml.MappingNode || len(itemNode.Content) < 2 {
				return "", errors.New("invalid changeset array item format - expected mapping with at least one key-value pair")
			}

			keyNode := itemNode.Content[0]
			valueNode := itemNode.Content[1]

			csName := keyNode.Value
			resolver := resolverByKey[csName]

			resolvedCfg, resolveErr := ResolveChangesetConfig(valueNode, csName, resolver)
			if resolveErr != nil {
				return "", resolveErr
			}

			changesetItem := map[string]any{
				csName: map[string]any{"payload": resolvedCfg},
			}
			orderedChangesets = append(orderedChangesets, changesetItem)
		}
	default:
		return "", fmt.Errorf("changesets must be either an object (mapping) or an array (sequence), got %v", dpFile.Changesets.Kind)
	}

	var changesetsNode *yaml.Node

	if dpFile.Changesets.Kind == yaml.MappingNode {
		changesetsNode = &yaml.Node{Kind: yaml.MappingNode}
		for _, changesetItem := range orderedChangesets {
			for csName, csConfig := range changesetItem {
				keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: csName}
				changesetsNode.Content = append(changesetsNode.Content, keyNode)

				valueNode, nodeErr := anyToYAMLNode(csConfig)
				if nodeErr != nil {
					return "", fmt.Errorf("encode changeset value for %s: %w", csName, nodeErr)
				}
				changesetsNode.Content = append(changesetsNode.Content, valueNode)

				break
			}
		}
	} else {
		changesetsNode = &yaml.Node{Kind: yaml.SequenceNode}
		for _, changesetItem := range orderedChangesets {
			itemNode := &yaml.Node{Kind: yaml.MappingNode}
			for csName, csConfig := range changesetItem {
				keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: csName}
				itemNode.Content = append(itemNode.Content, keyNode)

				valueNode, nodeErr := anyToYAMLNode(csConfig)
				if nodeErr != nil {
					return "", fmt.Errorf("encode changeset value for %s: %w", csName, nodeErr)
				}
				itemNode.Content = append(itemNode.Content, valueNode)

				break
			}
			changesetsNode.Content = append(changesetsNode.Content, itemNode)
		}
	}

	finalOutputNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "environment"},
			{Kind: yaml.ScalarNode, Value: opts.EnvKey},
			{Kind: yaml.ScalarNode, Value: "domain"},
			{Kind: yaml.ScalarNode, Value: opts.Domain.String()},
			{Kind: yaml.ScalarNode, Value: "changesets"},
			changesetsNode,
		},
	}

	var outBytes []byte
	if opts.FormatAsJSON {
		var finalOutput map[string]any
		if decodeErr := finalOutputNode.Decode(&finalOutput); decodeErr != nil {
			return "", fmt.Errorf("decode final output for JSON: %w", decodeErr)
		}
		outBytes, err = json.MarshalIndent(finalOutput, "", "  ")
	} else {
		outBytes, err = yaml.Marshal(finalOutputNode)
	}
	if err != nil {
		return "", fmt.Errorf("encode output: %w", err)
	}

	output := string(outBytes)

	if opts.OutputPath != "" {
		if err := os.WriteFile(opts.OutputPath, outBytes, 0o644); err != nil { //nolint:gosec
			return "", fmt.Errorf("write output file: %w", err)
		}
	}

	return output, nil
}

// anyToYAMLNode converts dynamic resolver output into a YAML node tree while
// preserving numeric semantics for json.Number values.
//
// yaml.v3 treats json.Number as a string-like value by default and emits quoted
// scalars (e.g. "1"), which changes number fields into strings in generated
// inputs. We map json.Number to scalar nodes with numeric tags so values remain
// numbers in YAML/JSON output.
func anyToYAMLNode(v any) (*yaml.Node, error) {
	switch typed := v.(type) {
	case map[string]any:
		node := &yaml.Node{Kind: yaml.MappingNode}
		keys := make([]string, 0, len(typed))
		for k := range typed {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			valueNode, err := anyToYAMLNode(typed[k])
			if err != nil {
				return nil, err
			}
			node.Content = append(node.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: k},
				valueNode,
			)
		}

		return node, nil
	case []any:
		node := &yaml.Node{Kind: yaml.SequenceNode}
		for _, elem := range typed {
			elemNode, err := anyToYAMLNode(elem)
			if err != nil {
				return nil, err
			}
			node.Content = append(node.Content, elemNode)
		}

		return node, nil
	case json.Number:
		value := typed.String()
		tag := "!!float"
		if decimalInteger.MatchString(value) {
			tag = "!!int"
		}

		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   tag,
			Value: value,
		}, nil
	default:
		node := &yaml.Node{}
		if err := node.Encode(v); err != nil {
			return nil, err
		}

		return node, nil
	}
}

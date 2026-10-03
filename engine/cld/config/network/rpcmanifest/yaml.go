package rpcmanifest

import (
	"cmp"
	"errors"
	"io"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
)

// WriteConfigYAML writes a network config in the standard CLD networks YAML format.
func WriteConfigYAML(cfg *cfgnet.Config, w io.Writer) error {
	if cfg == nil {
		return errors.New("network config is nil")
	}
	if w == nil {
		return errors.New("writer is nil")
	}

	networks := cfg.Networks()
	slices.SortFunc(networks, func(a, b cfgnet.Network) int {
		return cmp.Compare(a.ChainSelector, b.ChainSelector)
	})

	yamlOut, err := buildYAMLWithAnchor(networks)
	if err != nil {
		return err
	}

	_, err = w.Write(yamlOut)

	return err
}

func buildYAMLWithAnchor(networks []cfgnet.Network) ([]byte, error) {
	preferred := &yaml.Node{
		Kind:   yaml.ScalarNode,
		Value:  preferredTransportScheme,
		Anchor: "preferred_url_scheme",
		Tag:    "!!str",
		// Double-quote to match the scaffolded network YAML templates (e.g.
		// engine/cld/scaffold/templates/testnet.yaml.tmpl:2), so generated output is consistent
		// with hand-authored network yaml.
		Style: yaml.DoubleQuotedStyle,
	}
	networksYaml, err := yaml.Marshal(networks)
	if err != nil {
		return nil, err
	}
	var networksNode yaml.Node
	if err := yaml.Unmarshal(networksYaml, &networksNode); err != nil {
		return nil, err
	}
	networksValue := sequenceNodeFromYAML(&networksNode)
	aliasPreferredURLSchemes(networksValue, preferred)

	root := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "preferred_url_scheme"},
			preferred,
			{Kind: yaml.ScalarNode, Value: "networks"},
			networksValue,
		},
	}
	var bld strings.Builder
	enc := yaml.NewEncoder(&bld)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}

	return []byte(bld.String()), nil
}

// aliasPreferredURLSchemes rewrites every rpcs[].preferred_url_scheme scalar value in the given
// networks sequence node to an alias referencing anchor, matching the scaffolded network YAML
// templates (e.g. engine/cld/scaffold/templates/testnet.yaml.tmpl) instead of repeating the
// literal scheme string on every RPC entry.
func aliasPreferredURLSchemes(networksSeq *yaml.Node, anchor *yaml.Node) {
	if networksSeq.Kind != yaml.SequenceNode {
		return
	}

	for _, networkNode := range networksSeq.Content {
		rpcsNode := mappingValue(networkNode, "rpcs")
		if rpcsNode == nil || rpcsNode.Kind != yaml.SequenceNode {
			continue
		}

		for _, rpcNode := range rpcsNode.Content {
			schemeNode := mappingValue(rpcNode, "preferred_url_scheme")
			if schemeNode == nil || schemeNode.Kind != yaml.ScalarNode {
				continue
			}
			if schemeNode.Value != anchor.Value {
				continue
			}

			schemeNode.Kind = yaml.AliasNode
			schemeNode.Value = anchor.Anchor
			schemeNode.Tag = ""
			schemeNode.Alias = anchor
		}
	}
}

// mappingValue returns the value node for a key in a YAML mapping node, or nil if not found.
func mappingValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}

	return nil
}

func sequenceNodeFromYAML(node *yaml.Node) *yaml.Node {
	if node == nil {
		return &yaml.Node{Kind: yaml.SequenceNode}
	}
	if node.Kind == yaml.SequenceNode {
		return node
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 && node.Content[0].Kind == yaml.SequenceNode {
		return node.Content[0]
	}

	return &yaml.Node{Kind: yaml.SequenceNode}
}

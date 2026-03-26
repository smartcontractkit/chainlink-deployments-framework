package input

import (
	"encoding/json"
	"fmt"

	"github.com/suzuki-shunsuke/go-convmap/convmap"
	"gopkg.in/yaml.v3"

	"github.com/smartcontractkit/chainlink-deployments-framework/changeset/resolvers"
)

// ResolveChangesetConfig resolves the configuration for a changeset using either
// a registered resolver or keeping the original payload.
func ResolveChangesetConfig(valueNode *yaml.Node, csName string, resolver resolvers.ConfigResolver) (any, error) {
	changesetMap, ok := YamlNodeToAny(valueNode).(map[string]any)
	if !ok {
		return nil, fmt.Errorf("decode changeset data for %s: expected mapping node", csName)
	}
	// Parse payload directly from yaml.Node conversion to preserve integer literals as json.Number.
	payload, payloadExists := changesetMap["payload"]
	if !payloadExists {
		return nil, fmt.Errorf("decode changeset data for %s: missing required 'payload' field", csName)
	}

	if resolver != nil {
		jsonSafePayload, err := convmap.Convert(payload, nil)
		if err != nil {
			return nil, fmt.Errorf("convert payload for %s: %w", csName, err)
		}
		raw, err := json.Marshal(jsonSafePayload)
		if err != nil {
			return nil, fmt.Errorf("marshal payload for %s: %w", csName, err)
		}

		resolvedCfg, err := resolvers.CallResolver[any](resolver, raw)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve config for changeset %q: %w", csName, err)
		}

		return resolvedCfg, nil
	}

	return payload, nil
}

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
	var changesetData struct {
		Payload any `yaml:"payload"`
	}

	if err := valueNode.Decode(&changesetData); err != nil {
		return nil, fmt.Errorf("decode changeset data for %s: %w", csName, err)
	}

	if resolver != nil {
		jsonSafePayload, err := convmap.Convert(changesetData.Payload, nil)
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

	return changesetData.Payload, nil
}

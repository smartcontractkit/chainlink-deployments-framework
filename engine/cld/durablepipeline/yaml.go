package durablepipeline

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var decimalInteger = regexp.MustCompile(`^-?(0|[1-9][0-9]*)$`)

// ParsedYAML represents the structure of a durable pipeline YAML input file.
type ParsedYAML struct {
	Environment string `yaml:"environment"`
	Domain      string `yaml:"domain"`
	Changesets  any    `yaml:"changesets"` // Must be []any (array format)
}

// NamedChangeset is a changeset name with its corresponding YAML data.
type NamedChangeset struct {
	Name string
	Data any
}

// ParseYAMLBytes parses and validates durable pipeline YAML content.
func ParseYAMLBytes(yamlData []byte) (*ParsedYAML, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(yamlData, &root); err != nil {
		return nil, fmt.Errorf("failed to parse YAML bytes: %w", err)
	}

	rootMap, ok := yamlNodeToAny(&root).(map[string]any)
	if !ok {
		return nil, errors.New("expected a YAML object at the root")
	}

	envRaw, hasEnv := rootMap["environment"]
	domainRaw, hasDomain := rootMap["domain"]
	changesetsRaw, hasChangesets := rootMap["changesets"]

	dpYAML := &ParsedYAML{
		Changesets: changesetsRaw,
	}
	if envStr, ok := envRaw.(string); ok {
		dpYAML.Environment = envStr
	}
	if domainStr, ok := domainRaw.(string); ok {
		dpYAML.Domain = domainStr
	}

	if !hasEnv || dpYAML.Environment == "" {
		return nil, errors.New("missing required 'environment' field")
	}
	if !hasDomain || dpYAML.Domain == "" {
		return nil, errors.New("missing required 'domain' field")
	}
	if !hasChangesets || dpYAML.Changesets == nil {
		return nil, errors.New("missing required 'changesets' field")
	}

	return dpYAML, nil
}

// FindChangesetInData finds a changeset in array format.
func FindChangesetInData(changesets any, changesetName string) (any, error) {
	data, ok := changesets.([]any)
	if !ok {
		return nil, errors.New("invalid 'changesets' format, expected array format")
	}

	if len(data) == 0 {
		return nil, errors.New("empty 'changesets' array")
	}

	for _, item := range data {
		if itemMap, ok := item.(map[string]any); ok {
			if changesetData, exists := itemMap[changesetName]; exists {
				return changesetData, nil
			}
		}
	}

	return nil, fmt.Errorf("changeset '%s' not found", changesetName)
}

// GetAllChangesetsInOrder returns all changesets in order from array format changesets data.
func GetAllChangesetsInOrder(changesets any) ([]NamedChangeset, error) {
	var result []NamedChangeset

	data, ok := changesets.([]any)
	if !ok {
		return nil, errors.New("invalid 'changesets' format for index access, expected array format")
	}

	for _, item := range data {
		if itemMap, ok := item.(map[string]any); ok {
			for name, changesetData := range itemMap {
				result = append(result, NamedChangeset{
					Name: name,
					Data: changesetData,
				})
			}
		}
	}

	return result, nil
}

// SetChangesetEnvironmentVariable sets DURABLE_PIPELINE_INPUT from changeset data.
func SetChangesetEnvironmentVariable(changesetName string, changesetData any, inputName string) error {
	inputJSON, err := BuildChangesetInputJSON(changesetName, changesetData)
	if err != nil {
		return fmt.Errorf("failed to build input for changeset %q in input file %s: %w", changesetName, inputName, err)
	}

	if err := os.Setenv("DURABLE_PIPELINE_INPUT", inputJSON); err != nil {
		return fmt.Errorf("failed to set DURABLE_PIPELINE_INPUT environment variable: %w", err)
	}

	return nil
}

// BuildChangesetInputJSON returns the DURABLE_PIPELINE_INPUT JSON string for changeset data.
func BuildChangesetInputJSON(changesetName string, changesetData any) (string, error) {
	changesetMap, ok := changesetData.(map[string]any)
	if !ok {
		return "", fmt.Errorf("changeset %q is not a valid object", changesetName)
	}

	payload, payloadExists := changesetMap["payload"]
	if !payloadExists {
		return "", fmt.Errorf("changeset %q is missing required 'payload' field", changesetName)
	}

	jsonSafePayload, err := convertToJSONSafe(payload)
	if err != nil {
		return "", fmt.Errorf("failed to convert payload to JSON-safe format: %w", err)
	}

	chainOverridesRaw, exists := changesetMap["chainOverrides"]
	if exists && chainOverridesRaw != nil {
		if chainOverridesList, ok := chainOverridesRaw.([]any); ok {
			for _, override := range chainOverridesList {
				switch v := override.(type) {
				case int:
					if v < 0 {
						return "", fmt.Errorf("chain override value must be non-negative, got: %d", v)
					}
				case int64:
					if v < 0 {
						return "", fmt.Errorf("chain override value must be non-negative, got: %d", v)
					}
				case uint64:
				case json.Number:
					n, ok := new(big.Int).SetString(v.String(), 10)
					if !ok {
						return "", fmt.Errorf("chain override value must be an integer, got type %T with value: %v", override, override)
					}
					if n.Sign() < 0 {
						return "", fmt.Errorf("chain override value must be non-negative, got: %s", v.String())
					}
				default:
					return "", fmt.Errorf("chain override value must be an integer, got type %T with value: %v", override, override)
				}
			}
		}
	}

	input := map[string]any{
		"payload": jsonSafePayload,
	}
	if exists {
		input["chainOverrides"] = chainOverridesRaw
	}

	jsonData, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload to JSON: %w", err)
	}

	return string(jsonData), nil
}

// convertToJSONSafe recursively converts YAML-decoded objects to JSON-safe structures.
func convertToJSONSafe(data any) (any, error) {
	switch v := data.(type) {
	case map[interface{}]interface{}:
		result := make(map[string]any)
		for key, value := range v {
			var keyStr string
			switch k := key.(type) {
			case string:
				keyStr = k
			case int:
				keyStr = strconv.Itoa(k)
			case int64:
				keyStr = strconv.FormatInt(k, 10)
			case uint64:
				keyStr = strconv.FormatUint(k, 10)
			case float64:
				keyStr = strconv.FormatFloat(k, 'f', -1, 64)
			default:
				keyStr = fmt.Sprintf("%v", k)
			}

			convertedValue, err := convertToJSONSafe(value)
			if err != nil {
				return nil, err
			}
			result[keyStr] = convertedValue
		}

		return result, nil
	case map[string]any:
		result := make(map[string]any)
		for key, value := range v {
			convertedValue, err := convertToJSONSafe(value)
			if err != nil {
				return nil, err
			}
			result[key] = convertedValue
		}

		return result, nil
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			convertedItem, err := convertToJSONSafe(item)
			if err != nil {
				return nil, err
			}
			result[i] = convertedItem
		}

		return result, nil
	case float64:
		if (v >= 1e15 || v <= -1e15) && v == math.Trunc(v) {
			formatted := strconv.FormatFloat(v, 'f', 0, 64)
			return json.Number(formatted), nil
		}

		return v, nil
	default:
		return v, nil
	}
}

func yamlNodeToAny(node *yaml.Node) any {
	if node == nil {
		return nil
	}

	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) == 0 {
			return nil
		}

		return yamlNodeToAny(node.Content[0])
	case yaml.MappingNode:
		out := make(map[string]any, len(node.Content)/2)
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i]
			value := node.Content[i+1]
			out[key.Value] = yamlNodeToAny(value)
		}

		return out
	case yaml.SequenceNode:
		out := make([]any, 0, len(node.Content))
		for _, elem := range node.Content {
			out = append(out, yamlNodeToAny(elem))
		}

		return out
	case yaml.ScalarNode:
		if node.Style == 0 && decimalInteger.MatchString(node.Value) {
			return json.Number(node.Value)
		}

		switch node.Tag {
		case "!!int":
			if decimalInteger.MatchString(node.Value) {
				return json.Number(node.Value)
			}
			if n, ok := new(big.Int).SetString(strings.ReplaceAll(node.Value, "_", ""), 0); ok {
				return json.Number(n.String())
			}

			return node.Value
		case "!!float":
			f, err := strconv.ParseFloat(node.Value, 64)
			if err != nil {
				return node.Value
			}

			return f
		case "!!null":
			return nil
		case "!!bool":
			return strings.EqualFold(node.Value, "true")
		default:
			return node.Value
		}
	case yaml.AliasNode:
		return yamlNodeToAny(node.Alias)
	default:
		return nil
	}
}

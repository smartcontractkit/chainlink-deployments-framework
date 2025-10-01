package commands

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
)

// durablePipelineYAML represents the structure of a durable pipeline YAML input file
type durablePipelineYAML struct {
	Environment string `yaml:"environment"`
	Domain      string `yaml:"domain"`
	Changesets  any    `yaml:"changesets"` // Can be either map[string]any or []any
}

// setDurablePipelineInputFromYAML reads a YAML file, extracts the payload for the specified changeset,
// and sets it as the DURABLE_PIPELINE_INPUT environment variable in JSON format.
// If inputFileName is just a filename (no path separators), it will be resolved relative to the
// appropriate durable_pipelines/inputs directory based on the domain and environment.
func setDurablePipelineInputFromYAML(inputFileName, changesetName string, domain domain.Domain, envKey string) error {
	resolvedPath, err := resolveDurablePipelineYamlPath(inputFileName, domain, envKey)
	if err != nil {
		return fmt.Errorf("failed to resolve input file path: %w", err)
	}

	yamlData, err := os.ReadFile(resolvedPath)
	if err != nil {
		return fmt.Errorf("failed to read input file %s: %w", resolvedPath, err)
	}

	var dpYAML durablePipelineYAML
	if err = yaml.Unmarshal(yamlData, &dpYAML); err != nil {
		return fmt.Errorf("failed to parse input file %s: %w", inputFileName, err)
	}

	if dpYAML.Environment == "" {
		return fmt.Errorf("input file %s is missing required 'environment' field", inputFileName)
	}
	if dpYAML.Domain == "" {
		return fmt.Errorf("input file %s is missing required 'domain' field", inputFileName)
	}
	if dpYAML.Changesets == nil {
		return fmt.Errorf("input file %s is missing required 'changesets' field", inputFileName)
	}

	// Find the changeset - handle both object and array formats
	changesetData, err := findChangesetInData(dpYAML.Changesets, changesetName, inputFileName)
	if err != nil {
		return err
	}

	// Convert changeset data to map to access fields
	changesetMap, ok := changesetData.(map[string]any)
	if !ok {
		return fmt.Errorf("changeset '%s' in input file %s is not a valid object", changesetName, inputFileName)
	}

	payload, payloadExists := changesetMap["payload"]
	if !payloadExists || payload == nil {
		return fmt.Errorf("changeset '%s' in input file %s is missing required 'payload' field", changesetName, inputFileName)
	}

	// Convert payload to JSON-safe format to handle map[interface{}]interface{} types
	jsonSafePayload, err := convertToJSONSafe(payload)
	if err != nil {
		return fmt.Errorf("failed to convert payload to JSON-safe format: %w", err)
	}

	chainOverridesRaw, exists := changesetMap["chainOverrides"]
	if exists && chainOverridesRaw != nil {
		if chainOverridesList, ok := chainOverridesRaw.([]any); ok {
			for _, override := range chainOverridesList {
				switch v := override.(type) {
				case int:
					if v < 0 {
						return fmt.Errorf("chain override value must be non-negative, got: %d", v)
					}
				case int64:
					if v < 0 {
						return fmt.Errorf("chain override value must be non-negative, got: %d", v)
					}
				case uint64:
					// no need to do any checks here
				default:
					return fmt.Errorf("chain override value must be an integer, got type %T with value: %v", override, override)
				}
			}
		}
	}

	// Create the JSON structure that WithEnvInput expects
	inputJSON := map[string]any{
		"payload": jsonSafePayload,
	}
	if exists {
		inputJSON["chainOverrides"] = chainOverridesRaw
	}

	// Convert to JSON
	jsonData, err := json.Marshal(inputJSON)
	if err != nil {
		return fmt.Errorf("failed to marshal payload to JSON: %w", err)
	}

	// Set the environment variable
	if err := os.Setenv("DURABLE_PIPELINE_INPUT", string(jsonData)); err != nil {
		return fmt.Errorf("failed to set DURABLE_PIPELINE_INPUT environment variable: %w", err)
	}

	return nil
}

// findChangesetInData finds a changeset in either object or array format
func findChangesetInData(changesets any, changesetName, inputFileName string) (any, error) {
	switch data := changesets.(type) {
	case map[string]any:
		// Object format: {"changeset1": {...}, "changeset2": {...}}
		if len(data) == 0 {
			return nil, fmt.Errorf("input file %s has empty 'changesets' object", inputFileName)
		}

		changesetData, exists := data[changesetName]
		if !exists {
			return nil, fmt.Errorf("changeset '%s' not found in input file %s", changesetName, inputFileName)
		}

		return changesetData, nil

	case []any:
		// Array format: [{"changeset1": {...}}, {"changeset2": {...}}]
		if len(data) == 0 {
			return nil, fmt.Errorf("input file %s has empty 'changesets' array", inputFileName)
		}

		// Search through array items for the changeset
		for _, item := range data {
			if itemMap, ok := item.(map[string]any); ok {
				if changesetData, exists := itemMap[changesetName]; exists {
					return changesetData, nil
				}
			}
		}

		return nil, fmt.Errorf("changeset '%s' not found in input file %s", changesetName, inputFileName)

	default:
		return nil, fmt.Errorf("input file %s has invalid 'changesets' format, expected object or array", inputFileName)
	}
}

// convertToJSONSafe recursively converts map[interface{}]interface{} to map[string]any
// and handles other YAML types that need conversion for JSON marshaling.
// This is because the JSON marshaling library does not support map[interface{}]interface{}.
func convertToJSONSafe(data any) (any, error) {
	switch v := data.(type) {
	case map[interface{}]interface{}:
		// Convert map[interface{}]interface{} to map[string]any
		result := make(map[string]any)
		for key, value := range v {
			// Convert key to string - handle both string and numeric keys
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

			// Recursively convert the value
			convertedValue, err := convertToJSONSafe(value)
			if err != nil {
				return nil, err
			}
			result[keyStr] = convertedValue
		}

		return result, nil

	case map[string]any:
		// Already the right type, but recursively convert values
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
		// Convert slice elements recursively
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
		// Convert large numbers that would become scientific notation to json.Number
		// as it can cause issues to big.Int when it tries to unmarshal it.
		// Only convert if it's actually an integer (no fractional part)
		if v >= 1e15 || v <= -1e15 {
			// Check if this is truly an integer (no fractional part)
			if v == math.Trunc(v) {
				// This is a large integer that would be in scientific notation
				// Convert to json.Number to preserve exact representation
				formatted := strconv.FormatFloat(v, 'f', 0, 64)
				return json.Number(formatted), nil
			}
		}

		return v, nil

	default:
		// For primitive types (string, int, bool, etc.), return as-is
		return v, nil
	}
}

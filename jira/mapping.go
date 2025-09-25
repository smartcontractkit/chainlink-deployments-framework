package jira

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	cfgjira "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/jira"
)

// mapFieldsToStruct maps JIRA fields to a target struct using the field mappings.
func mapFieldsToStruct[T any](issue *JiraIssue, config *cfgjira.Config) (T, error) {
	var result T

	if issue == nil {
		return result, errors.New("nil issue")
	}
	if config == nil {
		return result, errors.New("nil config")
	}

	remappedData := make(map[string]any, len(config.FieldMaps))

	for configFieldName, fieldMapping := range config.FieldMaps {
		path := strings.TrimSpace(fieldMapping.JiraField)
		if path == "" {
			return result, fmt.Errorf("field %q has empty jira_field", configFieldName)
		}

		value, ok := getByPath(issue, path)
		if !ok || value == nil {
			return result, fmt.Errorf("field %s (JIRA field: %s) not found in issue - expected field specified in schema", configFieldName, fieldMapping.JiraField)
		}

		remappedData[configFieldName] = value
	}

	// JSON round-trip lets Go handle type conversions into T.
	jsonBytes, err := json.Marshal(remappedData)
	if err != nil {
		return result, fmt.Errorf("failed to marshal remapped data: %w", err)
	}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return result, fmt.Errorf("failed to unmarshal to target struct (type mismatch - user specified wrong type): %w", err)
	}

	return result, nil
}

// getByPath resolves a dotted path against the Jira issue.
// If the path doesn't start with "fields." or "key", it's treated as "fields.<path>".
func getByPath(issue *JiraIssue, path string) (any, bool) {
	if issue == nil || path == "" {
		return nil, false
	}

	if !strings.HasPrefix(path, "fields.") && path != "key" && !strings.HasPrefix(path, "key.") {
		path = "fields." + path
	}

	root := map[string]any{
		"key":    issue.Key,
		"fields": issue.Fields,
	}

	cur := any(root)
	for _, seg := range strings.Split(path, ".") {
		switch node := cur.(type) {
		case map[string]any:
			v, ok := node[seg]
			if !ok {
				return nil, false
			}
			cur = v
		case []any:
			// numeric array index
			idx, ok := parseIndex(seg)
			if !ok || idx < 0 || idx >= len(node) {
				return nil, false
			}
			cur = node[idx]
		default:
			// trying to traverse deeper but current is a scalar
			return nil, false
		}
	}

	return cur, true
}

// parseIndex converts a dotted path segment to a non-negative int index.
func parseIndex(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}

	return n, true
}

package jira

import (
	"encoding/json"
	"fmt"
)

// mapFieldsToStruct maps JIRA fields to a target struct using the field mappings
func mapFieldsToStruct[T any](issue *JiraIssue, config *JiraConfig) (T, error) {
	var result T

	// Create a remapped JSON object based on the schema
	remappedData := make(map[string]interface{})

	// Get all the struct field names we need to populate
	// We'll iterate through the schema field_maps to build the remapped data
	for configFieldName, fieldMapping := range config.FieldMaps {
		// Extract value from JIRA issue
		value, exists := issue.Fields[fieldMapping.JiraField]
		if !exists || value == nil {
			return result, fmt.Errorf("field %s (JIRA field: %s) not found in issue - expected field specified in schema", configFieldName, fieldMapping.JiraField)
		}

		// Map the JIRA field to the config field name
		remappedData[configFieldName] = value
	}
	fmt.Println("remappedData", remappedData)

	// Convert to JSON and back to let Go handle all the type conversions
	jsonBytes, err := json.Marshal(remappedData)
	if err != nil {
		return result, fmt.Errorf("failed to marshal remapped data: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return result, fmt.Errorf("failed to unmarshal to target struct (type mismatch - user specified wrong type): %w", err)
	}
	fmt.Println("result", result)

	return result, nil
}

package jira

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cfgjira "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/jira"
)

func TestParseIndex(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected int
		valid    bool
	}{
		{
			name:     "valid single digit",
			input:    "0",
			expected: 0,
			valid:    true,
		},
		{
			name:     "valid multi-digit",
			input:    "123",
			expected: 123,
			valid:    true,
		},
		{
			name:     "valid large number",
			input:    "999",
			expected: 999,
			valid:    true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
			valid:    false,
		},
		{
			name:     "non-numeric string",
			input:    "abc",
			expected: 0,
			valid:    false,
		},
		{
			name:     "mixed alphanumeric",
			input:    "12a",
			expected: 0,
			valid:    false,
		},
		{
			name:     "negative number",
			input:    "-1",
			expected: 0,
			valid:    false,
		},
		{
			name:     "decimal number",
			input:    "1.5",
			expected: 0,
			valid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, valid := parseIndex(tt.input)

			assert.Equal(t, tt.valid, valid)
			if valid {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGetByPath(t *testing.T) {
	t.Parallel()
	issue := &JiraIssue{
		Key: "TEST-123",
		Fields: map[string]any{
			"summary": "Test Issue",
			"status": map[string]any{
				"name": "To Do",
				"id":   "1",
			},
			"assignee": map[string]any{
				"displayName":  "John Doe",
				"emailAddress": "john@example.com",
			},
			"labels": []any{
				"bug",
				"urgent",
				"frontend",
			},
			"customfield_10001": "Custom Value",
		},
	}

	tests := []struct {
		name     string
		path     string
		expected any
		found    bool
	}{
		{
			name:     "get issue key",
			path:     "key",
			expected: "TEST-123",
			found:    true,
		},
		{
			name:     "get simple field",
			path:     "summary",
			expected: "Test Issue",
			found:    true,
		},
		{
			name:     "get nested field",
			path:     "status.name",
			expected: "To Do",
			found:    true,
		},
		{
			name:     "get deeply nested field",
			path:     "assignee.displayName",
			expected: "John Doe",
			found:    true,
		},
		{
			name:     "get array element",
			path:     "labels.0",
			expected: "bug",
			found:    true,
		},
		{
			name:     "get array element by index",
			path:     "labels.1",
			expected: "urgent",
			found:    true,
		},
		{
			name:     "get custom field",
			path:     "customfield_10001",
			expected: "Custom Value",
			found:    true,
		},
		{
			name:     "explicit fields prefix",
			path:     "fields.summary",
			expected: "Test Issue",
			found:    true,
		},
		{
			name:     "explicit fields prefix with nested",
			path:     "fields.status.name",
			expected: "To Do",
			found:    true,
		},
		{
			name:     "non-existent field",
			path:     "nonexistent",
			expected: nil,
			found:    false,
		},
		{
			name:     "non-existent nested field",
			path:     "status.nonexistent",
			expected: nil,
			found:    false,
		},
		{
			name:     "array index out of bounds",
			path:     "labels.10",
			expected: nil,
			found:    false,
		},
		{
			name:     "negative array index",
			path:     "labels.-1",
			expected: nil,
			found:    false,
		},
		{
			name:     "invalid array index",
			path:     "labels.abc",
			expected: nil,
			found:    false,
		},
		{
			name:     "empty path",
			path:     "",
			expected: nil,
			found:    false,
		},
		{
			name:     "nil issue",
			path:     "summary",
			expected: nil,
			found:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var testIssue *JiraIssue
			if tt.name == "nil issue" {
				testIssue = nil
			} else {
				testIssue = issue
			}

			result, found := getByPath(testIssue, tt.path)

			assert.Equal(t, tt.found, found)
			if found {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestMapFieldsToStruct(t *testing.T) {
	t.Parallel()
	// Define test structs
	type SimpleStruct struct {
		Summary string `json:"summary"`
		Status  string `json:"status"`
	}

	type ArrayStruct struct {
		Summary string   `json:"summary"`
		Labels  []string `json:"labels"`
	}

	type CustomFieldStruct struct {
		Summary     string `json:"summary"`
		CustomField string `json:"custom_field"`
	}

	type MultipleCustomFieldsStruct struct {
		Summary     string `json:"summary"`
		StoryPoints string `json:"story_points"`
		Priority    string `json:"priority"`
		EpicLink    string `json:"epic_link"`
	}

	issue := &JiraIssue{
		Key: "TEST-123",
		Fields: map[string]any{
			"summary": "Test Issue",
			"status": map[string]any{
				"name": "To Do",
				"id":   "1",
			},
			"labels": []any{
				"bug",
				"urgent",
			},
			"customfield_10001": "Custom Value",
		},
	}

	// Issue with multiple custom fields for comprehensive testing
	issueWithMultipleCustomFields := &JiraIssue{
		Key: "TEST-456",
		Fields: map[string]any{
			"summary":           "Epic Story",
			"customfield_10028": "5",        // Story Points
			"customfield_10016": "High",     // Priority
			"customfield_10014": "EPIC-123", // Epic Link
		},
	}

	config := &cfgjira.Config{
		FieldMaps: map[string]cfgjira.FieldMapping{
			"summary": {JiraField: "summary"},
			"status":  {JiraField: "status"},
		},
	}

	tests := []struct {
		name          string
		issue         *JiraIssue
		config        *cfgjira.Config
		expectError   bool
		errorContains string
		validate      func(any) error
	}{
		{
			name:  "successful mapping to simple struct",
			issue: issue,
			config: &cfgjira.Config{
				FieldMaps: map[string]cfgjira.FieldMapping{
					"summary": {JiraField: "summary"},
					"status":  {JiraField: "status.name"},
				},
			},
			expectError: false,
			validate: func(result any) error {
				s, ok := result.(SimpleStruct)
				if !ok {
					return fmt.Errorf("Expected SimpleStruct, got %T", result)
				}
				if s.Summary != "Test Issue" {
					return fmt.Errorf("Expected Summary 'Test Issue', got '%s'", s.Summary)
				}
				if s.Status != "To Do" {
					return fmt.Errorf("Expected Status 'To Do', got '%s'", s.Status)
				}

				return nil
			},
		},
		{
			name:  "successful mapping with nested fields",
			issue: issue,
			config: &cfgjira.Config{
				FieldMaps: map[string]cfgjira.FieldMapping{
					"summary": {JiraField: "summary"},
					"status":  {JiraField: "status.name"},
				},
			},
			expectError: false,
			validate: func(result any) error {
				s, ok := result.(SimpleStruct)
				if !ok {
					return fmt.Errorf("Expected SimpleStruct, got %T", result)
				}
				if s.Summary != "Test Issue" {
					return fmt.Errorf("Expected Summary 'Test Issue', got '%s'", s.Summary)
				}
				if s.Status != "To Do" {
					return fmt.Errorf("Expected Status 'To Do', got '%s'", s.Status)
				}

				return nil
			},
		},
		{
			name:  "successful mapping with array fields",
			issue: issue,
			config: &cfgjira.Config{
				FieldMaps: map[string]cfgjira.FieldMapping{
					"summary": {JiraField: "summary"},
					"labels":  {JiraField: "labels"},
				},
			},
			expectError: false,
			validate: func(result any) error {
				s, ok := result.(ArrayStruct)
				if !ok {
					return fmt.Errorf("Expected ArrayStruct, got %T", result)
				}
				if s.Summary != "Test Issue" {
					return fmt.Errorf("Expected Summary 'Test Issue', got '%s'", s.Summary)
				}
				if len(s.Labels) != 2 {
					return fmt.Errorf("Expected 2 labels, got %d", len(s.Labels))
				}
				if s.Labels[0] != "bug" {
					return fmt.Errorf("Expected first label 'bug', got '%s'", s.Labels[0])
				}

				return nil
			},
		},
		{
			name:  "successful mapping with custom fields",
			issue: issue,
			config: &cfgjira.Config{
				FieldMaps: map[string]cfgjira.FieldMapping{
					"summary":      {JiraField: "summary"},
					"custom_field": {JiraField: "customfield_10001"},
				},
			},
			expectError: false,
			validate: func(result any) error {
				s, ok := result.(CustomFieldStruct)
				if !ok {
					return fmt.Errorf("Expected CustomFieldStruct, got %T", result)
				}
				if s.Summary != "Test Issue" {
					return fmt.Errorf("Expected Summary 'Test Issue', got '%s'", s.Summary)
				}
				if s.CustomField != "Custom Value" {
					return fmt.Errorf("Expected CustomField 'Custom Value', got '%s'", s.CustomField)
				}

				return nil
			},
		},
		{
			name:          "nil issue",
			issue:         nil,
			config:        config,
			expectError:   true,
			errorContains: "nil issue",
		},
		{
			name:          "nil config",
			issue:         issue,
			config:        nil,
			expectError:   true,
			errorContains: "nil config",
		},
		{
			name:  "missing field in issue",
			issue: issue,
			config: &cfgjira.Config{
				FieldMaps: map[string]cfgjira.FieldMapping{
					"summary": {JiraField: "summary"},
					"missing": {JiraField: "nonexistent_field"},
				},
			},
			expectError:   true,
			errorContains: "not found in issue",
		},
		{
			name:  "empty jira_field",
			issue: issue,
			config: &cfgjira.Config{
				FieldMaps: map[string]cfgjira.FieldMapping{
					"summary": {JiraField: "summary"},
					"empty":   {JiraField: ""},
				},
			},
			expectError:   true,
			errorContains: "empty jira_field",
		},
		{
			name:  "whitespace-only jira_field",
			issue: issue,
			config: &cfgjira.Config{
				FieldMaps: map[string]cfgjira.FieldMapping{
					"summary": {JiraField: "summary"},
					"empty":   {JiraField: "   "},
				},
			},
			expectError:   true,
			errorContains: "empty jira_field",
		},
		{
			name:  "custom field mapping",
			issue: issueWithMultipleCustomFields,
			config: &cfgjira.Config{
				FieldMaps: map[string]cfgjira.FieldMapping{
					"summary":      {JiraField: "summary"},
					"story_points": {JiraField: "customfield_10028"}, // Story Points field
					"priority":     {JiraField: "customfield_10016"}, // Priority field
					"epic_link":    {JiraField: "customfield_10014"}, // Epic Link field
				},
			},
			expectError: false,
			validate: func(result any) error {
				s, ok := result.(MultipleCustomFieldsStruct)
				if !ok {
					return fmt.Errorf("Expected MultipleCustomFieldsStruct, got %T", result)
				}
				if s.Summary != "Epic Story" {
					return fmt.Errorf("Expected Summary 'Epic Story', got '%s'", s.Summary)
				}
				if s.StoryPoints != "5" {
					return fmt.Errorf("Expected StoryPoints '5', got '%s'", s.StoryPoints)
				}
				if s.Priority != "High" {
					return fmt.Errorf("Expected Priority 'High', got '%s'", s.Priority)
				}
				if s.EpicLink != "EPIC-123" {
					return fmt.Errorf("Expected EpicLink 'EPIC-123', got '%s'", s.EpicLink)
				}

				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var result any
			var err error

			// Use the appropriate struct type based on the test
			switch tt.name {
			case "successful mapping with array fields":
				result, err = mapFieldsToStruct[ArrayStruct](tt.issue, tt.config)
			case "successful mapping with custom fields":
				result, err = mapFieldsToStruct[CustomFieldStruct](tt.issue, tt.config)
			case "custom field mapping":
				result, err = mapFieldsToStruct[MultipleCustomFieldsStruct](tt.issue, tt.config)
			default:
				result, err = mapFieldsToStruct[SimpleStruct](tt.issue, tt.config)
			}

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}

				return
			}

			require.NoError(t, err)

			if tt.validate != nil {
				require.NoError(t, tt.validate(result))
			}
		})
	}
}

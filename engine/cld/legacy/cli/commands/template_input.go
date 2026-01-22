package commands

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/smartcontractkit/chainlink-deployments-framework/changeset/resolvers"
	cs "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
)

// generateMultiChangesetYAMLTemplate creates a YAML template with multiple changesets
func generateMultiChangesetYAMLTemplate(
	domainName string,
	envKey string,
	changesetNames []string,
	registry *cs.ChangesetsRegistry,
	resolverManager *resolvers.ConfigResolverManager,
	depthLimit int,
) (string, error) {
	if len(changesetNames) == 0 {
		return "", errors.New("no changeset names provided")
	}

	// Start with header
	yamlTemplate := fmt.Sprintf(`# Generated via template-input command
environment: %s
domain: %s
changesets:
`, envKey, domainName)

	// Generate each changeset section
	var yamlTemplateSb34 strings.Builder
	for i, changesetName := range changesetNames {
		if changesetName == "" {
			continue
		}

		// Add separator between changesets
		if i > 0 {
			yamlTemplateSb34.WriteString("\n  # ----------------------------------------\n")
		}

		// Get changeset configuration
		cfg, err := registry.GetConfigurations(changesetName)
		if err != nil {
			return "", fmt.Errorf("get configurations for changeset %s: %w", changesetName, err)
		}

		// Generate changeset section
		changesetSection, err := generateChangesetSection(changesetName, cfg, resolverManager, "  ", depthLimit)
		if err != nil {
			return "", fmt.Errorf("generate section for changeset %s: %w", changesetName, err)
		}

		yamlTemplateSb34.WriteString(changesetSection)
	}
	yamlTemplate += yamlTemplateSb34.String()

	return yamlTemplate, nil
}

// generateChangesetSection generates a single changeset section within a multi-changeset YAML
func generateChangesetSection(
	changesetName string,
	cfg cs.Configurations,
	resolverManager *resolvers.ConfigResolverManager,
	indent string,
	depthLimit int,
) (string, error) {
	var section strings.Builder

	// Add changeset header comment
	if cfg.ConfigResolver != nil {
		resolverName := resolverManager.NameOf(cfg.ConfigResolver)
		if resolverName == "" {
			return "", fmt.Errorf("resolver for changeset %s is not registered", changesetName)
		}

		// Use reflection to get the input type of the resolver
		rf := reflect.TypeOf(cfg.ConfigResolver)
		if rf.Kind() != reflect.Func || rf.NumIn() != 1 {
			return "", fmt.Errorf("invalid resolver signature for %s", changesetName)
		}

		inputType := rf.In(0)

		section.WriteString(fmt.Sprintf("%s# Config Resolver: %s\n", indent, resolverName))
		section.WriteString(fmt.Sprintf("%s# Input type: %s\n", indent, inputType.String()))
		section.WriteString(fmt.Sprintf("%s- %s:\n", indent, changesetName))

		// Add chainOverrides at the changeset level (before payload)
		writeChainOverridesSection(&section, indent)

		section.WriteString(indent + "    payload:\n")

		// Generate the payload structure from the struct
		payloadYAML, err := generateStructYAMLWithDepthLimit(inputType, indent+"      ", 0, make(map[reflect.Type]bool), depthLimit)
		if err != nil {
			return "", fmt.Errorf("generate struct YAML for %s: %w", inputType.String(), err)
		}

		section.WriteString(payloadYAML)
	} else if cfg.InputType != nil {
		// We have type information - generate template based on it
		section.WriteString(fmt.Sprintf("%s# Input type: %s\n", indent, cfg.InputType.String()))
		section.WriteString(fmt.Sprintf("%s- %s:\n", indent, changesetName))

		// Add chainOverrides at the changeset level (before payload)
		writeChainOverridesSection(&section, indent)

		section.WriteString(indent + "    payload:\n")

		// Generate the payload structure from the struct
		payloadYAML, err := generateStructYAMLWithDepthLimit(cfg.InputType, indent+"      ", 0, make(map[reflect.Type]bool), depthLimit)
		if err != nil {
			return "", fmt.Errorf("generate struct YAML for %s: %w", cfg.InputType.String(), err)
		}

		section.WriteString(payloadYAML)
	}

	return section.String(), nil
}

// generateStructYAMLWithDepthLimit recursively generates YAML structure with user-configurable depth limiting
func generateStructYAMLWithDepthLimit(
	t reflect.Type,
	indent string,
	depth int,
	visited map[reflect.Type]bool,
	maxDepth int,
) (string, error) {
	if depth > maxDepth {
		return "", nil
	}

	// Handle pointers
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Check for cycles
	if visited[t] {
		return fmt.Sprintf("# ... (circular reference to %s)\n", t.String()), nil
	}

	switch t.Kind() { //nolint:exhaustive // default case handles unspecified types
	case reflect.Struct:
		// Mark this type as visited for cycle detection
		visited[t] = true
		defer func() { delete(visited, t) }()

		var result strings.Builder
		fieldCount := 0
		maxFields := 20 // Limit number of fields to show

		for i := 0; i < t.NumField() && fieldCount < maxFields; i++ {
			field := t.Field(i)

			// Skip unexported fields
			if !field.IsExported() {
				continue
			}

			// Skip fields with yaml:"-" or json:"-" tag
			if yamlTag := field.Tag.Get("yaml"); yamlTag == "-" {
				continue
			}
			if jsonTag := field.Tag.Get("json"); jsonTag == "-" {
				continue
			}

			// Get field name from yaml/json tags or use field name
			fieldName := getFieldName(field)
			fieldType := field.Type

			// Generate value based on field type
			fieldValue, err := generateFieldValueWithDepthLimit(fieldType, indent+"  ", depth+1, visited, maxDepth)
			if err != nil {
				return "", fmt.Errorf("generate field value for %s: %w", field.Name, err)
			}

			result.WriteString(fmt.Sprintf("%s%s:", indent, fieldName))
			result.WriteString(fieldValue)
			if !strings.HasSuffix(fieldValue, "\n") {
				result.WriteString("\n")
			}
			fieldCount++
		}

		if t.NumField() > maxFields {
			result.WriteString(fmt.Sprintf("%s# ... and %d more fields\n", indent, t.NumField()-maxFields))
		}

		return result.String(), nil

	case reflect.Slice, reflect.Array:
		elemType := t.Elem()
		result := fmt.Sprintf("%s# Array of %s\n%s- ", indent, elemType.String(), indent)

		elemValue, err := generateFieldValueWithDepthLimit(elemType, indent+"  ", depth+1, visited, maxDepth)
		if err != nil {
			return "", err
		}

		return result + strings.TrimSpace(elemValue) + "\n", nil

	case reflect.Map:
		keyType := t.Key()
		valueType := t.Elem()

		// Special handling for map[string]interface{} - show the interface{} type
		if keyType.Kind() == reflect.String && valueType.Kind() == reflect.Interface {
			result := fmt.Sprintf("%s# Map[%s]%s\n", indent, keyType.String(), valueType.String())
			result += fmt.Sprintf("%sexample_key: # %s\n", indent, valueType.String())

			return result, nil
		}

		result := fmt.Sprintf("%s# Map[%s]%s\n%sexample_key: ", indent, keyType.String(), valueType.String(), indent)

		valueStr, err := generateFieldValueWithDepthLimit(valueType, indent+"  ", depth+1, visited, maxDepth)
		if err != nil {
			return "", err
		}

		return result + strings.TrimSpace(valueStr) + "\n", nil

	default:
		return " # " + t.String(), nil
	}
}

// generateFieldValueWithDepthLimit generates an example value for a field based on its type with user-configurable depth limiting
func generateFieldValueWithDepthLimit(
	t reflect.Type,
	indent string,
	depth int,
	visited map[reflect.Type]bool,
	maxDepth int,
) (string, error) {
	if depth > maxDepth {
		return " ...", nil
	}

	// Handle pointers
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() { //nolint:exhaustive // default case handles unspecified types
	case reflect.String:
		return " # string", nil
	case reflect.Bool:
		return " # bool", nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return " # " + t.String(), nil
	case reflect.Slice, reflect.Array:
		// Special case: if it's a slice/array of uint8 (bytes), treat it as a string type
		if t.Elem().Kind() == reflect.Uint8 {
			return " # " + t.String(), nil
		}
		// Regular slice/array handling
		elemType := t.Elem()
		elemValue, err := generateFieldValueWithDepthLimit(elemType, indent+"  ", depth+1, visited, maxDepth)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("\n%s- %s", indent, strings.TrimSpace(elemValue)), nil
	case reflect.Struct:
		structYAML, err := generateStructYAMLWithDepthLimit(t, indent, depth+1, visited, maxDepth)
		if err != nil {
			return "", err
		}

		return "\n" + structYAML, nil
	case reflect.Map:
		keyType := t.Key()
		valueType := t.Elem()
		valueStr, err := generateFieldValueWithDepthLimit(valueType, indent+"  ", depth+1, visited, maxDepth)
		if err != nil {
			return "", err
		}

		var keyExample string
		switch keyType.Kind() { //nolint:exhaustive // default case handles unspecified types
		case reflect.String:
			keyExample = "example_key"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			keyExample = "123"
		default:
			keyExample = "example_key"
		}

		return fmt.Sprintf("\n%s%s: %s", indent, keyExample, strings.TrimSpace(valueStr)), nil
	case reflect.Interface:
		return `"interface{} - provide appropriate value"`, nil
	default:
		return fmt.Sprintf(`"unknown_type_%s"`, t.Kind().String()), nil
	}
}

// getFieldName extracts the field name from yaml or json tags, falling back to the struct field name
func getFieldName(field reflect.StructField) string {
	// Try yaml tag first
	if yamlTag := field.Tag.Get("yaml"); yamlTag != "" {
		if parts := strings.Split(yamlTag, ","); len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}

	// Try json tag
	if jsonTag := field.Tag.Get("json"); jsonTag != "" {
		if parts := strings.Split(jsonTag, ","); len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}

	// Fall back to field name in lowercase
	return strings.ToLower(field.Name)
}

// writeChainOverridesSection writes the common chain overrides comment section
func writeChainOverridesSection(section *strings.Builder, indent string) {
	section.WriteString(indent + "    # Optional: Chain overrides (uncomment if needed)\n")
	section.WriteString(indent + "    # chainOverrides:\n")
	section.WriteString(indent + "    #   - 1  # Chain selector 1\n")
	section.WriteString(indent + "    #   - 2  # Chain selector 2\n")
}

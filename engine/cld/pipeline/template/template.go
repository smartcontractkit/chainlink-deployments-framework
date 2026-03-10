package template

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/smartcontractkit/chainlink-deployments-framework/changeset/resolvers"
	cs "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
)

// GenerateMultiChangesetYAML creates a YAML template with multiple changesets.
func GenerateMultiChangesetYAML(
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

	yamlTemplate := fmt.Sprintf(`# Generated via template-input command
environment: %s
domain: %s
changesets:
`, envKey, domainName)

	var sb strings.Builder
	for i, changesetName := range changesetNames {
		if changesetName == "" {
			continue
		}

		if i > 0 {
			sb.WriteString("\n  # ----------------------------------------\n")
		}

		cfg, err := registry.GetConfigurations(changesetName)
		if err != nil {
			return "", fmt.Errorf("get configurations for changeset %s: %w", changesetName, err)
		}

		section, err := generateChangesetSection(changesetName, cfg, resolverManager, "  ", depthLimit)
		if err != nil {
			return "", fmt.Errorf("generate section for changeset %s: %w", changesetName, err)
		}

		sb.WriteString(section)
	}
	yamlTemplate += sb.String()

	return yamlTemplate, nil
}

func generateChangesetSection(
	changesetName string,
	cfg cs.Configurations,
	resolverManager *resolvers.ConfigResolverManager,
	indent string,
	depthLimit int,
) (string, error) {
	var section strings.Builder

	if cfg.ConfigResolver != nil {
		resolverName := resolverManager.NameOf(cfg.ConfigResolver)
		if resolverName == "" {
			return "", fmt.Errorf("resolver for changeset %s is not registered", changesetName)
		}

		rf := reflect.TypeOf(cfg.ConfigResolver)
		if rf.Kind() != reflect.Func || rf.NumIn() != 1 {
			return "", fmt.Errorf("invalid resolver signature for %s", changesetName)
		}

		inputType := rf.In(0)

		section.WriteString(fmt.Sprintf("%s# Config Resolver: %s\n", indent, resolverName))
		section.WriteString(fmt.Sprintf("%s# Input type: %s\n", indent, inputType.String()))
		section.WriteString(fmt.Sprintf("%s- %s:\n", indent, changesetName))

		writeChainOverridesSection(&section, indent)

		section.WriteString(indent + "    payload:\n")

		payloadYAML, err := GenerateStructYAMLWithDepthLimit(inputType, indent+"      ", 0, make(map[reflect.Type]bool), depthLimit)
		if err != nil {
			return "", fmt.Errorf("generate struct YAML for %s: %w", inputType.String(), err)
		}

		section.WriteString(payloadYAML)
	} else if cfg.InputType != nil {
		section.WriteString(fmt.Sprintf("%s# Input type: %s\n", indent, cfg.InputType.String()))
		section.WriteString(fmt.Sprintf("%s- %s:\n", indent, changesetName))

		writeChainOverridesSection(&section, indent)

		section.WriteString(indent + "    payload:\n")

		payloadYAML, err := GenerateStructYAMLWithDepthLimit(cfg.InputType, indent+"      ", 0, make(map[reflect.Type]bool), depthLimit)
		if err != nil {
			return "", fmt.Errorf("generate struct YAML for %s: %w", cfg.InputType.String(), err)
		}

		section.WriteString(payloadYAML)
	}

	return section.String(), nil
}

func writeChainOverridesSection(section *strings.Builder, indent string) {
	section.WriteString(indent + "    # Optional: Chain overrides (uncomment if needed)\n")
	section.WriteString(indent + "    # chainOverrides:\n")
	section.WriteString(indent + "    #   - 1  # Chain selector 1\n")
	section.WriteString(indent + "    #   - 2  # Chain selector 2\n")
}

// GenerateStructYAMLWithDepthLimit recursively generates YAML structure with depth limiting.
func GenerateStructYAMLWithDepthLimit(
	t reflect.Type,
	indent string,
	depth int,
	visited map[reflect.Type]bool,
	maxDepth int,
) (string, error) {
	if depth > maxDepth {
		return "", nil
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if visited[t] {
		return fmt.Sprintf("# ... (circular reference to %s)\n", t.String()), nil
	}

	switch t.Kind() { //nolint:exhaustive
	case reflect.Struct:
		visited[t] = true
		defer func() { delete(visited, t) }()

		var result strings.Builder
		fieldCount := 0
		maxFields := 20

		for i := 0; i < t.NumField() && fieldCount < maxFields; i++ {
			field := t.Field(i)

			if !field.IsExported() {
				continue
			}

			if field.Tag.Get("yaml") == "-" || field.Tag.Get("json") == "-" {
				continue
			}

			fieldName := GetFieldName(field)
			fieldType := field.Type

			fieldValue, err := GenerateFieldValueWithDepthLimit(fieldType, indent+"  ", depth+1, visited, maxDepth)
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

		elemValue, err := GenerateFieldValueWithDepthLimit(elemType, indent+"  ", depth+1, visited, maxDepth)
		if err != nil {
			return "", err
		}

		return result + strings.TrimSpace(elemValue) + "\n", nil

	case reflect.Map:
		keyType := t.Key()
		valueType := t.Elem()

		if keyType.Kind() == reflect.String && valueType.Kind() == reflect.Interface {
			return fmt.Sprintf("%s# Map[%s]%s\n%sexample_key: # %s\n", indent, keyType.String(), valueType.String(), indent, valueType.String()), nil
		}

		valueStr, err := GenerateFieldValueWithDepthLimit(valueType, indent+"  ", depth+1, visited, maxDepth)
		if err != nil {
			return "", err
		}

		if strings.HasPrefix(valueStr, "\n") {
			return fmt.Sprintf("%s# Map[%s]%s\n%sexample_key:%s\n", indent, keyType.String(), valueType.String(), indent, strings.TrimRight(valueStr, "\n")), nil
		}

		return fmt.Sprintf("%s# Map[%s]%s\n%sexample_key: ", indent, keyType.String(), valueType.String(), indent) + strings.TrimSpace(valueStr) + "\n", nil

	default:
		return " # " + t.String(), nil
	}
}

// GenerateFieldValueWithDepthLimit generates an example value for a field based on its type.
// Exported for testing.
func GenerateFieldValueWithDepthLimit(
	t reflect.Type,
	indent string,
	depth int,
	visited map[reflect.Type]bool,
	maxDepth int,
) (string, error) {
	if depth > maxDepth {
		return " ...", nil
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() { //nolint:exhaustive
	case reflect.String:
		return " # string", nil
	case reflect.Bool:
		return " # bool", nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return " # " + t.String(), nil
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			return " # " + t.String(), nil
		}
		elemType := t.Elem()
		elemValue, err := GenerateFieldValueWithDepthLimit(elemType, indent+"  ", depth+1, visited, maxDepth)
		if err != nil {
			return "", err
		}

		trimmedElem := strings.TrimSpace(elemValue)
		if strings.Contains(trimmedElem, "\n") {
			return formatMultilineSliceElement(elemValue, indent), nil
		}

		return fmt.Sprintf("\n%s- %s", indent, trimmedElem), nil
	case reflect.Struct:
		structYAML, err := GenerateStructYAMLWithDepthLimit(t, indent, depth+1, visited, maxDepth)
		if err != nil {
			return "", err
		}

		return "\n" + structYAML, nil
	case reflect.Map:
		keyType := t.Key()
		valueType := t.Elem()
		valueStr, err := GenerateFieldValueWithDepthLimit(valueType, indent+"  ", depth+1, visited, maxDepth)
		if err != nil {
			return "", err
		}

		var keyExample string
		switch keyType.Kind() { //nolint:exhaustive
		case reflect.String:
			keyExample = "example_key"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			keyExample = "123"
		default:
			keyExample = "example_key"
		}

		if strings.HasPrefix(valueStr, "\n") {
			return fmt.Sprintf("\n%s%s:%s", indent, keyExample, valueStr), nil
		}

		return fmt.Sprintf("\n%s%s: %s", indent, keyExample, strings.TrimSpace(valueStr)), nil
	case reflect.Interface:
		return `"interface{} - provide appropriate value"`, nil
	default:
		return fmt.Sprintf(`"unknown_type_%s"`, t.Kind().String()), nil
	}
}

func formatMultilineSliceElement(elemValue string, indent string) string {
	lines := trimBoundaryEmptyLines(strings.Split(strings.TrimRight(elemValue, " \t\n"), "\n"))
	if len(lines) == 0 {
		return fmt.Sprintf("\n%s- ", indent)
	}

	knownPrefix := indent + "  "
	var listValue strings.Builder
	listValue.WriteString(fmt.Sprintf("\n%s- %s", indent, stripKnownPrefix(lines[0], knownPrefix)))

	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}

		listValue.WriteString("\n")
		listValue.WriteString(indent)
		listValue.WriteString("  ")
		listValue.WriteString(stripKnownPrefix(line, knownPrefix))
	}

	return listValue.String()
}

func trimBoundaryEmptyLines(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	return lines
}

func stripKnownPrefix(line string, knownPrefix string) string {
	if strings.HasPrefix(line, knownPrefix) {
		return line[len(knownPrefix):]
	}

	return strings.TrimLeft(line, " ")
}

// GetFieldName extracts the field name from yaml or json tags, falling back to the struct field name.
// Exported for testing.
func GetFieldName(field reflect.StructField) string {
	if yamlTag := field.Tag.Get("yaml"); yamlTag != "" {
		if parts := strings.Split(yamlTag, ","); len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}

	if jsonTag := field.Tag.Get("json"); jsonTag != "" {
		if parts := strings.Split(jsonTag, ","); len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}

	return strings.ToLower(field.Name)
}

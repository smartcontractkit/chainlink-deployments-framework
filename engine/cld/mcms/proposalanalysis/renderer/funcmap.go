package renderer

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"text/template"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotation"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/format"
	experimentalanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

const nilValue = "<nil>"

// frameworkAnnotations lists annotation names that are consumed by the
// framework's built-in template logic.
var frameworkAnnotations = map[string]struct{}{
	annotation.AnnotationSeverityName: {},
	annotation.AnnotationRiskName:     {},
	annotation.AnnotationDiffName:     {},
}

func defaultFuncMap() template.FuncMap {
	return template.FuncMap{
		"annotation":            findAnnotationValue,
		"isFrameworkAnnotation": isFrameworkAnnotation,
		"hasDisplayAnnotations": hasDisplayAnnotations,
		"diffAnnotations":       diffAnnotations,
		"renderDiff":            renderDiff,
		"formatParam":           formatParam,
		"indentLines":           indentLines,
		"formatAnnotationValue": formatAnnotationValue,
		"resolveChainSelector":  resolveChainSelector,
		"severitySymbol":        severitySymbol,
		"riskSymbol":            riskSymbol,
		"add":                   func(a, b int) int { return a + b },
	}
}

// isFrameworkAnnotation reports whether the given annotation name is handled
// by dedicated template logic rather than the generic annotations section.
func isFrameworkAnnotation(name string) bool {
	_, ok := frameworkAnnotations[name]

	return ok
}

// hasDisplayAnnotations reports whether any annotations should be shown in the
// generic annotation list.
func hasDisplayAnnotations(anns annotation.Annotations) bool {
	for _, ann := range anns {
		if !isFrameworkAnnotation(ann.Name()) {
			return true
		}
	}

	return false
}

// findAnnotationValue returns the value of the first annotation matching name.
func findAnnotationValue(anns annotation.Annotations, name string) any {
	for _, ann := range anns {
		if ann.Name() == name {
			return ann.Value()
		}
	}

	return nil
}

// formatParam formats a parameter's value for display.
func formatParam(param analyzer.AnalyzedParameter) string {
	v := formatValue(param.Value())

	t := param.Type()
	if strings.Contains(t, "int") {
		return commaGrouped(v)
	}

	return v
}

func indentLines(s, prefix string) string {
	if s == "" {
		return ""
	}

	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = prefix + lines[i]
	}

	return strings.Join(lines, "\n")
}

// formatAnnotationValue formats an annotation's value for display.
func formatAnnotationValue(v any) string {
	return formatValue(v)
}

// formatValue produces a human-readable string for arbitrary parameter values.
func formatValue(v any) string {
	if v == nil {
		return nilValue
	}

	switch val := v.(type) {
	case experimentalanalyzer.AddressField:
		return val.GetValue()
	case experimentalanalyzer.BytesField:
		if len(val.GetValue()) == 0 {
			return "0x"
		}

		return "0x" + hex.EncodeToString(val.GetValue())
	case experimentalanalyzer.SimpleField:
		return val.GetValue()
	case experimentalanalyzer.ChainSelectorField:
		if name, ok := format.TryResolveChainName(val.GetValue()); ok {
			return fmt.Sprintf("%s (%d)", name, val.GetValue())
		}

		return strconv.FormatUint(val.GetValue(), 10)
	case experimentalanalyzer.YamlField:
		return val.GetValue()
	case experimentalanalyzer.ArrayField:
		return formatArrayField(val)
	case experimentalanalyzer.StructField:
		return formatStructField(val)
	case []byte:
		if len(val) == 0 {
			return "0x"
		}

		return "0x" + hex.EncodeToString(val)
	case *big.Int:
		if val == nil {
			return nilValue
		}

		return val.String()
	case fmt.Stringer:
		rv := reflect.ValueOf(val)
		if rv.Kind() == reflect.Ptr && rv.IsNil() {
			return nilValue
		}

		return val.String()
	default:
		if b, err := json.MarshalIndent(v, "", "  "); err == nil {
			s := string(b)
			if len(s) > 0 && (s[0] == '{' || s[0] == '[') {
				return s
			}
		}

		return fmt.Sprintf("%v", v)
	}
}

func formatArrayField(af experimentalanalyzer.ArrayField) string {
	elems := af.GetElements()
	if len(elems) == 0 {
		return "[]"
	}

	parts := make([]string, len(elems))
	for i, elem := range elems {
		parts[i] = formatValue(elem)
	}

	if len(elems) == 1 {
		return "[" + parts[0] + "]"
	}

	for i, part := range parts {
		parts[i] = "  " + part
	}

	return "[\n" + strings.Join(parts, ",\n") + "\n]"
}

func formatStructField(sf experimentalanalyzer.StructField) string {
	fields := sf.GetFields()
	if len(fields) == 0 {
		return "{}"
	}

	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		parts = append(parts, fmt.Sprintf("%s: %s", f.Name, formatValue(f.Value)))
	}

	return "{ " + strings.Join(parts, ", ") + " }"
}

// commaGrouped adds comma separators to a numeric value for readability.
func commaGrouped(v any) string {
	if n, ok := v.(json.Number); ok {
		v = string(n)
	}

	switch val := v.(type) {
	case *big.Int:
		if val == nil {
			return nilValue
		}

		return format.CommaGroupBigInt(val)
	case string:
		num, ok := new(big.Int).SetString(val, 10)
		if !ok {
			return val
		}

		return format.CommaGroupBigInt(num)
	default:
		rv := reflect.ValueOf(v)
		if rv.CanInt() {
			return format.CommaGroupBigInt(big.NewInt(rv.Int()))
		}

		if rv.CanUint() {
			return format.CommaGroupBigInt(new(big.Int).SetUint64(rv.Uint()))
		}

		if rv.CanFloat() {
			s := strconv.FormatFloat(rv.Float(), 'f', -1, 64)
			parts := strings.Split(s, ".")

			num, ok := new(big.Int).SetString(parts[0], 10)
			if !ok {
				return s
			}

			intPart := format.CommaGroupBigInt(num)
			if len(parts) == 2 {
				return intPart + "." + parts[1]
			}

			return intPart
		}

		return formatValue(v)
	}
}

func resolveChainSelector(sel uint64) string {
	if name, ok := format.TryResolveChainName(sel); ok {
		return name
	}

	return "Chain " + strconv.FormatUint(sel, 10)
}

func severitySymbol(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}

	switch annotation.Severity(s) {
	case annotation.SeverityError:
		return "✗"
	case annotation.SeverityWarning:
		return "⚠"
	case annotation.SeverityInfo:
		return "ℹ"
	case annotation.SeverityDebug:
		return "⚙"
	default:
		return ""
	}
}

func riskSymbol(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}

	switch annotation.Risk(s) {
	case annotation.RiskHigh:
		return "🔴"
	case annotation.RiskMedium:
		return "🟡"
	case annotation.RiskLow:
		return "🟢"
	default:
		return ""
	}
}

// diffAnnotations extracts all diff annotations as DiffValue structs.
func diffAnnotations(anns annotation.Annotations) []annotation.DiffValue {
	var diffs []annotation.DiffValue
	for _, ann := range anns {
		if ann.Name() == annotation.AnnotationDiffName {
			if dv, ok := ann.Value().(annotation.DiffValue); ok {
				diffs = append(diffs, dv)
			}
		}
	}

	return diffs
}

// renderDiff formats a DiffValue as a markdown diff string.
func renderDiff(dv annotation.DiffValue) string {
	oldStr := formatDiffSide(dv.Old, dv.ValueType)
	newStr := formatDiffSide(dv.New, dv.ValueType)

	if dv.Field != "" {
		return fmt.Sprintf("**%s:** ~~%s~~ -> **%s**", dv.Field, oldStr, newStr)
	}

	return fmt.Sprintf("~~%s~~ -> **%s**", oldStr, newStr)
}

func formatDiffSide(v any, valueType string) string {
	if strings.Contains(valueType, "int") {
		return commaGrouped(v)
	}

	return formatValue(v)
}

package renderer

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"text/template"

	chainutils "github.com/smartcontractkit/chainlink-deployments-framework/chain/utils"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotation"
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
		"truncateAddress":       truncateAddress,
		"resolveChainSelector":  resolveChainSelector,
		"severitySymbol":        severitySymbol,
		"riskSymbol":            riskSymbol,
		"add":                   func(a, b int) int { return a + b },
	}
}

// resolveChainSelector returns a human-readable chain name for a selector.
func resolveChainSelector(sel uint64) string {
	info, err := chainutils.ChainInfo(sel)
	if err != nil {
		return strconv.FormatUint(sel, 10)
	}

	return info.ChainName
}

// truncateAddress shortens a long address for display.
func truncateAddress(addr string) string {
	if strings.HasPrefix(addr, "0x") && len(addr) > 12 {
		return addr[:6] + ".." + addr[len(addr)-4:]
	}
	if len(addr) > 12 {
		return addr[:4] + ".." + addr[len(addr)-3:]
	}

	return addr
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
		name := resolveChainSelector(val.GetValue())

		return fmt.Sprintf("%s (%d)", name, val.GetValue())
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

// commaGrouped adds comma separators to a numeric string for readability.
func commaGrouped(v any) string {
	var num *big.Int
	switch val := v.(type) {
	case *big.Int:
		if val == nil {
			return nilValue
		}
		num = val
	case string:
		var ok bool
		num, ok = new(big.Int).SetString(val, 10)
		if !ok {
			return val
		}
	default:
		num = new(big.Int)
		if _, err := fmt.Sscan(fmt.Sprintf("%v", v), num); err != nil {
			return fmt.Sprintf("%v", v)
		}
	}

	s := num.String()
	sign := ""
	if strings.HasPrefix(s, "-") {
		sign = "-"
		s = strings.TrimPrefix(s, "-")
	}
	if len(s) <= 3 {
		return sign + s
	}

	var b strings.Builder
	if sign != "" {
		b.WriteString(sign)
	}
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteRune(',')
		}
		b.WriteRune(ch)
	}

	return b.String()
}

func severitySymbol(severity any) string {
	switch fmt.Sprintf("%v", severity) {
	case string(annotation.SeverityError):
		return "✗"
	case string(annotation.SeverityWarning):
		return "⚠"
	case string(annotation.SeverityInfo):
		return "ℹ"
	case string(annotation.SeverityDebug):
		return "⚙"
	default:
		return ""
	}
}

func riskSymbol(risk any) string {
	switch fmt.Sprintf("%v", risk) {
	case string(annotation.RiskHigh):
		return "🔴"
	case string(annotation.RiskMedium):
		return "🟡"
	case string(annotation.RiskLow):
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

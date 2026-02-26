package renderer

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"text/template"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotation"
)

const nilValue = "<nil>"

// frameworkAnnotations lists annotation names that are consumed by the
// framework's built-in template logic.
var frameworkAnnotations = map[string]struct{}{
	annotation.AnnotationSeverityName:  {},
	annotation.AnnotationRiskName:      {},
	annotation.AnnotationValueTypeName: {},
	annotation.AnnotationDiffName:      {},
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
		"severitySymbol":        severitySymbol,
		"riskSymbol":            riskSymbol,
		"add":                   func(a, b int) int { return a + b },
	}
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
	for _, ann := range param.Annotations() {
		if ann.Name() == annotation.AnnotationValueTypeName {
			if vt, ok := ann.Value().(string); ok && vt != "" {
				return formatByValueType(param.Value(), vt)
			}
		}
	}

	return formatValue(param.Value())
}

// formatByValueType formats a raw value according to a semantic value type.
func formatByValueType(v any, valueType string) string {
	if v == nil {
		return nilValue
	}

	switch valueType {
	case "ethereum.address":
		return formatEthereumAddress(v)
	case "ethereum.uint256":
		return formatEthereumUint256(v)
	case "hex":
		return formatAsHex(v)
	default:
		return formatValue(v)
	}
}

// formatValue produces a human-readable string for arbitrary parameter values.
func formatValue(v any) string {
	if v == nil {
		return nilValue
	}

	switch val := v.(type) {
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

func formatEthereumAddress(v any) string {
	s := fmt.Sprintf("%v", v)
	s = strings.TrimPrefix(s, "0x")
	s = strings.ToLower(s)
	if len(s) < 40 {
		s = strings.Repeat("0", 40-len(s)) + s
	}

	return "0x" + s
}

func formatEthereumUint256(v any) string {
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

func formatAsHex(v any) string {
	switch val := v.(type) {
	case []byte:
		return "0x" + hex.EncodeToString(val)
	case *big.Int:
		if val == nil {
			return nilValue
		}

		return "0x" + strings.ToLower(val.Text(16))
	case string:
		if strings.HasPrefix(val, "0x") {
			return val
		}

		return "0x" + val
	case uint8, uint16, uint32, uint64, uint, int8, int16, int32, int64, int:
		return fmt.Sprintf("0x%x", val)
	default:
		return formatValue(v)
	}
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
	if valueType != "" {
		return formatByValueType(v, valueType)
	}

	return formatValue(v)
}

package annotation

import "strings"

const BuiltinReportNamePrefix = "cld.builtin."

// IsBuiltinReportName reports whether name is a structured built-in analyzer report
// annotation consumed by dedicated renderer templates.
func IsBuiltinReportName(name string) bool {
	return strings.HasPrefix(name, BuiltinReportNamePrefix) && strings.HasSuffix(name, ".report")
}

// Package text provides text formatting utilities for CLI commands.
package text

import (
	"strings"
)

// Indentation is the standard indentation for CLI help text.
const Indentation = `  `

// LongDesc normalizes a command's long description to follow the conventions.
func LongDesc(s string) string {
	if len(s) == 0 {
		return s
	}

	return normalizer{s}.trim().string
}

// Examples normalizes a command's examples to follow the conventions.
func Examples(s string) string {
	if len(s) == 0 {
		return s
	}

	return normalizer{s}.trim().indent().string
}

type normalizer struct {
	string
}

func (s normalizer) trim() normalizer {
	s.string = strings.TrimSpace(s.string)

	return s
}

func (s normalizer) indent() normalizer {
	lines := strings.Split(s.string, "\n")
	indentedLines := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		indented := Indentation + trimmed
		indentedLines = append(indentedLines, indented)
	}
	s.string = strings.Join(indentedLines, "\n")

	return s
}

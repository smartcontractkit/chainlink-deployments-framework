package internal

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/analyzer"
)

//go:embed templates/*
var embeddedTemplates embed.FS

// RenderFormat specifies the output format for rendering
type RenderFormat string

const (
	// FormatText renders in plain text format with ASCII art
	FormatText RenderFormat = "text"
	// FormatHTML renders as HTML with styling
	FormatHTML RenderFormat = "html"
	// FormatMarkdown renders as Markdown
	FormatMarkdown RenderFormat = "markdown"
	// FormatJSON renders as JSON
	FormatJSON RenderFormat = "json"
)

// Renderer renders an AnalyzedProposal using Go templates.
type Renderer struct {
	format RenderFormat
	tmpl   *template.Template
}

// NewRenderer creates a new renderer with default text format templates.
// For backward compatibility, defaults to text format.
func NewRenderer() (*Renderer, error) {
	return NewRendererWithFormat(FormatText)
}

// NewRendererWithFormat creates a new renderer with the specified format.
// Templates are loaded from embedded files in the templates/<format>/ directory.
func NewRendererWithFormat(format RenderFormat) (*Renderer, error) {
	return newRendererFromEmbedded(format)
}

// NewRendererFromDirectory creates a renderer that loads templates from a filesystem directory.
// The directory should contain subdirectories for each format (e.g., text/, html/).
func NewRendererFromDirectory(format RenderFormat, templateDir string) (*Renderer, error) {
	tmpl, err := template.New("root").Funcs(templateFuncs()).Parse("")
	if err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	// Load templates from the format-specific subdirectory
	formatDir := filepath.Join(templateDir, string(format))
	pattern := filepath.Join(formatDir, "*.tmpl")

	tmpl, err = tmpl.ParseGlob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to load templates from %s: %w", pattern, err)
	}

	return &Renderer{format: format, tmpl: tmpl}, nil
}

// NewRendererWithTemplates creates a new renderer with custom in-memory templates.
// This is useful for testing or programmatic template generation.
func NewRendererWithTemplates(format RenderFormat, templates map[string]string) (*Renderer, error) {
	tmpl, err := template.New("root").Funcs(templateFuncs()).Parse("")
	if err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	for name, content := range templates {
		tmpl, err = tmpl.New(name).Parse(content)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template %q: %w", name, err)
		}
	}

	return &Renderer{format: format, tmpl: tmpl}, nil
}

// newRendererFromEmbedded creates a renderer using embedded template files
func newRendererFromEmbedded(format RenderFormat) (*Renderer, error) {
	tmpl, err := template.New("root").Funcs(templateFuncs()).Parse("")
	if err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	// Load templates from embedded filesystem
	formatDir := fmt.Sprintf("templates/%s", format)
	entries, err := embeddedTemplates.ReadDir(formatDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded templates for format %s: %w", format, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tmpl") {
			continue
		}

		path := filepath.Join(formatDir, entry.Name())
		content, err := embeddedTemplates.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read embedded template %s: %w", path, err)
		}

		// Use the base name without extension as template name
		name := strings.TrimSuffix(entry.Name(), ".tmpl")
		tmpl, err = tmpl.New(name).Parse(string(content))
		if err != nil {
			return nil, fmt.Errorf("failed to parse embedded template %s: %w", path, err)
		}
	}

	return &Renderer{format: format, tmpl: tmpl}, nil
}

// Render renders the analyzed proposal to a string.
func (r *Renderer) Render(proposal analyzer.AnalyzedProposal) (string, error) {
	var buf bytes.Buffer
	if err := r.RenderTo(&buf, proposal); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderTo renders the analyzed proposal to the given writer.
func (r *Renderer) RenderTo(w io.Writer, proposal analyzer.AnalyzedProposal) error {
	if err := r.tmpl.ExecuteTemplate(w, "proposal", proposal); err != nil {
		return fmt.Errorf("failed to render proposal: %w", err)
	}
	return nil
}

// RenderToFile renders the analyzed proposal to a file.
func (r *Renderer) RenderToFile(filePath string, proposal analyzer.AnalyzedProposal) error {
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	return r.RenderTo(f, proposal)
}

// Format returns the format this renderer uses.
func (r *Renderer) Format() RenderFormat {
	return r.format
}

// templateFuncs returns the template functions available in all templates.
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		// String manipulation
		"indent": func(spaces int, text string) string {
			indent := strings.Repeat(" ", spaces)
			lines := strings.Split(text, "\n")
			for i, line := range lines {
				if line != "" {
					lines[i] = indent + line
				}
			}
			return strings.Join(lines, "\n")
		},
		"trimRight": strings.TrimRight,
		"upper":     strings.ToUpper,
		"lower":     strings.ToLower,
		"title":     strings.Title,
		"join": func(sep string, items []string) string {
			return strings.Join(items, sep)
		},
		"repeat": strings.Repeat,

		// Annotation functions
		"hasAnnotations": func(annotated analyzer.Annotated) bool {
			return annotated != nil && len(annotated.Annotations()) > 0
		},
		"getAnnotation": func(annotated analyzer.Annotated, name string) analyzer.Annotation {
			if annotated == nil {
				return nil
			}
			for _, ann := range annotated.Annotations() {
				if ann.Name() == name {
					return ann
				}
			}
			return nil
		},
		"getAnnotationValue": func(annotated analyzer.Annotated, name string) interface{} {
			if annotated == nil {
				return nil
			}
			for _, ann := range annotated.Annotations() {
				if ann.Name() == name {
					return ann.Value()
				}
			}
			return nil
		},
		"hasAnnotation": func(annotated analyzer.Annotated, name string) bool {
			if annotated == nil {
				return false
			}
			for _, ann := range annotated.Annotations() {
				if ann.Name() == name {
					return true
				}
			}
			return false
		},

		// Severity and risk symbols
		"severitySymbol": func(severity string) string {
			switch severity {
			case "error":
				return "✗"
			case "warning":
				return "⚠"
			case "info":
				return "ℹ"
			case "debug":
				return "⚙"
			default:
				return "?"
			}
		},
		"riskSymbol": func(risk string) string {
			switch risk {
			case "high":
				return "🔴"
			case "medium":
				return "🟡"
			case "low":
				return "🟢"
			default:
				return "⚪"
			}
		},

		// Value formatting functions
		"formatValue": func(param analyzer.AnalyzedParameter, formatter string) string {
			return formatParameterValue(param, formatter)
		},
	}
}

// formatParameterValue applies custom formatting to a parameter's value based on the formatter type
func formatParameterValue(param analyzer.AnalyzedParameter, formatter string) string {
	value := param.Value()
	if value == nil {
		return "<nil>"
	}

	// Handle different formatter types
	parts := strings.SplitN(formatter, ":", 2)
	formatterType := parts[0]

	switch formatterType {
	case "ethereum.address":
		return formatEthereumAddress(value)
	case "ethereum.uint256":
		return formatEthereumUint256(value)
	case "hex":
		return formatAsHex(value)
	case "truncate":
		if len(parts) > 1 {
			if length, err := strconv.Atoi(parts[1]); err == nil {
				return truncateString(fmt.Sprintf("%v", value), length)
			}
		}
		return fmt.Sprintf("%v", value)
	default:
		return fmt.Sprintf("%v", value)
	}
}

// formatEthereumAddress formats a value as an Ethereum address with 0x prefix
func formatEthereumAddress(value interface{}) string {
	str := fmt.Sprintf("%v", value)
	// Remove existing 0x prefix if present
	str = strings.TrimPrefix(str, "0x")
	// Ensure it's lowercase hex
	str = strings.ToLower(str)
	// Pad to 40 characters if needed
	if len(str) < 40 {
		str = strings.Repeat("0", 40-len(str)) + str
	}
	return "0x" + str
}

// formatEthereumUint256 formats a large number with commas for readability
func formatEthereumUint256(value interface{}) string {
	// Try to parse as big.Int
	var num *big.Int
	switch v := value.(type) {
	case *big.Int:
		num = v
	case string:
		var ok bool
		num, ok = new(big.Int).SetString(v, 10)
		if !ok {
			return fmt.Sprintf("%v", value)
		}
	case int, int64, uint, uint64:
		num = big.NewInt(0)
		fmt.Sscan(fmt.Sprintf("%v", v), num)
	default:
		return fmt.Sprintf("%v", value)
	}

	// Format with commas
	str := num.String()
	if len(str) <= 3 {
		return str
	}

	// Add commas
	var result strings.Builder
	for i, digit := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(digit)
	}
	return result.String()
}

// formatAsHex formats a value as hexadecimal
func formatAsHex(value interface{}) string {
	switch v := value.(type) {
	case []byte:
		return "0x" + fmt.Sprintf("%x", v)
	case string:
		return "0x" + v
	case int, int64, uint, uint64:
		return fmt.Sprintf("0x%x", v)
	default:
		return fmt.Sprintf("%v", value)
	}
}

// truncateString truncates a string to the specified length, adding "..." if truncated
func truncateString(str string, length int) string {
	if len(str) <= length {
		return str
	}
	if length <= 3 {
		return str[:length]
	}
	return str[:length-3] + "..."
}

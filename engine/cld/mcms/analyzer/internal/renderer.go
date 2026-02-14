package internal

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/analyzer"
)

// Renderer renders an AnalyzedProposal using Go templates.
type Renderer struct {
	tmpl *template.Template
}

// NewRenderer creates a new renderer with default templates.
func NewRenderer() (*Renderer, error) {
	tmpl, err := template.New("root").Funcs(templateFuncs()).Parse("")
	if err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	// Parse all templates
	tmpl, err = tmpl.Parse(proposalTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse proposal template: %w", err)
	}

	tmpl, err = tmpl.Parse(batchOperationTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse batch operation template: %w", err)
	}

	tmpl, err = tmpl.Parse(callTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse call template: %w", err)
	}

	tmpl, err = tmpl.Parse(parameterTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse parameter template: %w", err)
	}

	tmpl, err = tmpl.Parse(annotationsTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse annotations template: %w", err)
	}

	return &Renderer{tmpl: tmpl}, nil
}

// NewRendererWithTemplates creates a new renderer with custom templates.
func NewRendererWithTemplates(templates map[string]string) (*Renderer, error) {
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

	return &Renderer{tmpl: tmpl}, nil
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

// templateFuncs returns the template functions available in all templates.
func templateFuncs() template.FuncMap {
	return template.FuncMap{
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
		"hasAnnotations": func(annotated analyzer.Annotated) bool {
			return annotated != nil && len(annotated.Annotations()) > 0
		},
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
	}
}

package renderer

import (
	"embed"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/analyzer"
)

//go:embed templates/*
var embeddedTemplates embed.FS

const IDMarkdown = "markdown"

// TemplateRenderer renders an AnalyzedProposal using Go text/template.
type TemplateRenderer struct {
	id   string
	tmpl *template.Template
}

var _ Renderer = (*TemplateRenderer)(nil)

func (r *TemplateRenderer) ID() string { return r.id }

func (r *TemplateRenderer) RenderTo(w io.Writer, req RenderRequest, proposal analyzer.AnalyzedProposal) error {
	ctx := templateRenderContext{
		Request:  req,
		Proposal: proposal,
	}
	if err := r.tmpl.ExecuteTemplate(w, "proposal", ctx); err != nil {
		return fmt.Errorf("failed to render proposal: %w", err)
	}

	return nil
}

// NewMarkdownRenderer creates a TemplateRenderer with embedded markdown templates.
func NewMarkdownRenderer(opts ...Option) (*TemplateRenderer, error) {
	return newTemplateRenderer(IDMarkdown, "markdown", opts...)
}

func newTemplateRenderer(id, format string, opts ...Option) (*TemplateRenderer, error) {
	cfg := applyOptions(opts...)

	funcs := defaultFuncMap()
	for k, v := range cfg.extraFuncs {
		funcs[k] = v
	}

	tmpl := template.New("root").Funcs(funcs)

	var err error
	switch {
	case cfg.templateDir != "":
		tmpl, err = loadTemplatesFromDir(tmpl, cfg.templateDir)
	case cfg.templates != nil:
		tmpl, err = loadTemplatesFromMap(tmpl, cfg.templates)
	default:
		tmpl, err = loadEmbeddedTemplates(tmpl, format)
	}
	if err != nil {
		return nil, err
	}
	if err := validateRequiredTemplates(tmpl); err != nil {
		return nil, err
	}

	return &TemplateRenderer{id: id, tmpl: tmpl}, nil
}

type templateRenderContext struct {
	Request  RenderRequest
	Proposal analyzer.AnalyzedProposal
}

func validateRequiredTemplates(tmpl *template.Template) error {
	required := []string{"proposal", "batchOperation", "call", "parameter", "annotations"}
	missing := make([]string, 0, len(required))
	for _, name := range required {
		if tmpl.Lookup(name) == nil {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		slices.Sort(missing)
		return fmt.Errorf("template set is missing required template definitions: %s", strings.Join(missing, ", "))
	}

	return nil
}

func loadEmbeddedTemplates(tmpl *template.Template, format string) (*template.Template, error) {
	dir := "templates/" + format
	entries, err := embeddedTemplates.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded templates for format %q: %w", format, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tmpl") {
			continue
		}
		content, readErr := embeddedTemplates.ReadFile(dir + "/" + entry.Name())
		if readErr != nil {
			return nil, fmt.Errorf("failed to read embedded template %s: %w", entry.Name(), readErr)
		}
		name := strings.TrimSuffix(entry.Name(), ".tmpl")
		tmpl, err = tmpl.New(name).Parse(string(content))
		if err != nil {
			return nil, fmt.Errorf("failed to parse embedded template %s: %w", entry.Name(), err)
		}
	}

	return tmpl, nil
}

func loadTemplatesFromDir(tmpl *template.Template, dir string) (*template.Template, error) {
	pattern := filepath.Join(dir, "*.tmpl")
	tmpl, err := tmpl.ParseGlob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to load templates from %s: %w", pattern, err)
	}

	return tmpl, nil
}

func loadTemplatesFromMap(tmpl *template.Template, templates map[string]string) (*template.Template, error) {
	for name, content := range templates {
		var err error
		tmpl, err = tmpl.New(name).Parse(content)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template %q: %w", name, err)
		}
	}

	return tmpl, nil
}

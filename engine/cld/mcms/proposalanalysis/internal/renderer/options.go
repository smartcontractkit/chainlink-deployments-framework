package renderer

import "text/template"

// Option configures a TemplateRenderer.
type Option func(*config)

type config struct {
	templateDir string
	templates   map[string]string
	extraFuncs  template.FuncMap
}

func applyOptions(opts ...Option) config {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	return cfg
}

// WithTemplateDir loads templates from a filesystem directory instead of the
// embedded defaults. The directory should contain *.tmpl files whose {{define}}
// names match the default set: proposal, batchOperation, call, parameter,
// annotations.
func WithTemplateDir(dir string) Option {
	return func(c *config) {
		c.templateDir = dir
	}
}

// WithTemplates provides in-memory template overrides keyed by any name.
// Each value must contain a valid {{define "name"}} block -- the key itself
// is used only as a template set label and does not need to match the define name.
// Useful for testing or programmatic template generation.
func WithTemplates(templates map[string]string) Option {
	return func(c *config) {
		c.templates = templates
	}
}

// WithTemplateFuncs adds extra template functions to the renderer's func map.
// These are merged with the built-in functions; caller-provided functions take
// precedence over built-ins if keys collide.
func WithTemplateFuncs(funcs template.FuncMap) Option {
	return func(c *config) {
		c.extraFuncs = funcs
	}
}

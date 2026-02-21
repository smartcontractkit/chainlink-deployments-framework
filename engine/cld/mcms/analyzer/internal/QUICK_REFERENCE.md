# Quick Reference: Enhanced Renderer

## Creating Renderers

```go
// Text format (default)
renderer, err := internal.NewRenderer()

// HTML format
htmlRenderer, err := internal.NewRendererWithFormat(internal.FormatHTML)

// Custom templates
customRenderer, err := internal.NewRendererWithTemplates(internal.FormatText, templates)

// Load from directory
fsRenderer, err := internal.NewRendererFromDirectory(internal.FormatHTML, "/path/to/templates")
```

## Rendering

```go
// To string
output, err := renderer.Render(analyzedProposal)

// To writer
err := renderer.RenderTo(writer, analyzedProposal)

// To file
err := renderer.RenderToFile("proposal.html", analyzedProposal)
```

## Rendering Annotations

```go
// Mark as important
entity.AddAnnotations(internal.ImportantAnnotation(true))

// Add emoji
entity.AddAnnotations(internal.EmojiAnnotation("💰"))

// Format value
param.AddAnnotations(internal.FormatterAnnotation("ethereum.address"))
param.AddAnnotations(internal.FormatterAnnotation("ethereum.uint256"))
param.AddAnnotations(internal.FormatterAnnotation("hex"))
param.AddAnnotations(internal.FormatterAnnotation("truncate:20"))

// Style (HTML)
entity.AddAnnotations(internal.StyleAnnotation("danger"))

// Custom template
entity.AddAnnotations(internal.TemplateAnnotation("customTemplate"))

// Hide from output
entity.AddAnnotations(internal.HideAnnotation(true))

// Tooltip (HTML)
entity.AddAnnotations(internal.TooltipAnnotation("Description..."))
```

## Analysis Annotations (Auto-rendered)

```go
// Severity with symbols
entity.AddAnnotations(internal.SeverityAnnotation("error"))    // ✗
entity.AddAnnotations(internal.SeverityAnnotation("warning"))  // ⚠
entity.AddAnnotations(internal.SeverityAnnotation("info"))     // ℹ
entity.AddAnnotations(internal.SeverityAnnotation("debug"))    // ⚙

// Risk with colored symbols
entity.AddAnnotations(internal.RiskAnnotation("high"))    // 🔴
entity.AddAnnotations(internal.RiskAnnotation("medium"))  // 🟡
entity.AddAnnotations(internal.RiskAnnotation("low"))     // 🟢
```

## Template Functions

```go
// In templates:
{{getAnnotation . "annotation.name"}}
{{getAnnotationValue . "annotation.name"}}
{{hasAnnotation . "annotation.name"}}
{{formatValue .Param "formatter"}}
{{severitySymbol "warning"}}
{{riskSymbol "high"}}
```

## Custom Template Example

```go
// Text template
{{define "call"}}
CALL: {{.Name}}{{if getAnnotation . "render.important"}} ⭐{{end}}
{{- $severity := getAnnotationValue . "cld.severity"}}
{{if $severity}}Severity: {{severitySymbol $severity}} {{$severity}}{{end}}
{{end}}

// HTML template
{{define "parameter"}}
{{- $formatter := getAnnotationValue . "render.formatter"}}
<strong>{{.Name}}</strong> ({{.Type}}): 
<code>{{if $formatter}}{{formatValue . $formatter}}{{else}}{{.Value}}{{end}}</code>
{{end}}
```

## Available Formatters

| Formatter | Input | Output | Use Case |
|-----------|-------|--------|----------|
| `ethereum.address` | `"1234..."` | `"0x1234..."` | Ethereum addresses |
| `ethereum.uint256` | `"1000000000"` | `"1,000,000,000"` | Large numbers |
| `hex` | `[]byte{0x12, 0x34}` | `"0x1234"` | Hex values |
| `truncate:N` | `"long string"` | `"long st..."` | Long strings |

## Template Directory Structure

```
templates/
  ├── text/
  │   ├── proposal.tmpl
  │   ├── batchOperation.tmpl
  │   ├── call.tmpl
  │   └── parameter.tmpl
  └── html/
      ├── proposal.tmpl
      ├── batchOperation.tmpl
      ├── call.tmpl
      └── parameter.tmpl
```

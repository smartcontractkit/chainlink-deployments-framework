# Enhanced Analyzer Renderer

The renderer component has been enhanced to support multiple output formats, file-based templates, and annotation-driven rendering.

## Overview

The enhanced renderer provides:

1. **Multiple Output Formats**: Text, HTML, Markdown, JSON (extensible)
2. **File-Based Templates**: Templates are organized in format-specific directories
3. **Annotation-Driven Rendering**: Annotations control how entities are displayed
4. **Embedded Templates**: Templates are embedded in the binary for easy deployment
5. **Custom Formatters**: Extensible value formatting system

## Architecture

### Format Support

The renderer supports multiple output formats:

```go
const (
    FormatText     RenderFormat = "text"     // Plain text with ASCII art
    FormatHTML     RenderFormat = "html"     // HTML with CSS styling
    FormatMarkdown RenderFormat = "markdown" // Markdown (future)
    FormatJSON     RenderFormat = "json"     // JSON (future)
)
```

### Template Organization

Templates are organized in format-specific directories:

```
internal/templates/
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

Each format has its own set of templates optimized for that output type.

## Usage

### Basic Usage

#### Text Format (Default)

```go
renderer, err := internal.NewRenderer()
if err != nil {
    return err
}

output, err := renderer.Render(analyzedProposal)
fmt.Println(output)
```

#### HTML Format

```go
renderer, err := internal.NewRendererWithFormat(internal.FormatHTML)
if err != nil {
    return err
}

// Render to string
htmlOutput, err := renderer.Render(analyzedProposal)

// Or render to file
err = renderer.RenderToFile("proposal.html", analyzedProposal)
```

### Custom Templates

You can provide custom templates programmatically:

```go
customTemplates := map[string]string{
    "proposal": `{{define "proposal"}}
Custom Proposal Format
=====================
{{range .BatchOperations}}{{template "batchOperation" .}}{{end}}
{{end}}`,
    // ... more templates
}

renderer, err := internal.NewRendererWithTemplates(internal.FormatText, customTemplates)
```

### Loading Templates from Filesystem

Load templates from a custom directory:

```go
renderer, err := internal.NewRendererFromDirectory(internal.FormatText, "/path/to/templates")
```

## Annotation-Driven Rendering

The key enhancement is that annotations now control rendering behavior instead of being displayed as separate entities.

### Rendering Annotations

#### `render.important`

Marks an entity as important, causing it to be highlighted:

```go
param.AddAnnotations(internal.ImportantAnnotation(true))
// Text: ⭐ parameter_name
// HTML: <span class="important">parameter_name</span>
```

#### `render.emoji`

Adds an emoji decoration:

```go
param.AddAnnotations(internal.EmojiAnnotation("💰"))
// Output: 💰 amount (uint256): 1000
```

#### `render.formatter`

Specifies a custom value formatter:

```go
param.AddAnnotations(internal.FormatterAnnotation("ethereum.address"))
// Input:  "1234567890abcdef..."
// Output: "0x1234567890abcdef..."

param.AddAnnotations(internal.FormatterAnnotation("ethereum.uint256"))
// Input:  "1000000000"
// Output: "1,000,000,000"

param.AddAnnotations(internal.FormatterAnnotation("truncate:20"))
// Input:  "very long string..."
// Output: "very long string..."
```

#### `render.style`

Provides styling hints (mainly for HTML):

```go
call.AddAnnotations(internal.StyleAnnotation("danger"))
// HTML: applies danger styling
```

#### `render.template`

Specifies a custom template to use:

```go
call.AddAnnotations(internal.TemplateAnnotation("customCall"))
// Uses customCall.tmpl instead of call.tmpl
```

#### `render.hide`

Hides an entity from output:

```go
param.AddAnnotations(internal.HideAnnotation(true))
// Entity will not be rendered
```

#### `render.tooltip`

Adds tooltip text (HTML format):

```go
param.AddAnnotations(internal.TooltipAnnotation("This parameter controls..."))
// HTML: adds title attribute with tooltip text
```

### Built-in Analysis Annotations

These annotations from analyzers also affect rendering:

#### `cld.severity`

Severity levels are displayed with symbols:

```go
call.AddAnnotations(internal.SeverityAnnotation("warning"))
// Text: ⚠ warning
// HTML: <span class="severity-warning">⚠ warning</span>
```

Symbols:
- `error`: ✗
- `warning`: ⚠
- `info`: ℹ
- `debug`: ⚙

#### `cld.risk`

Risk levels are displayed with colored symbols:

```go
batchOp.AddAnnotations(internal.RiskAnnotation("high"))
// Text: 🔴 high
// HTML: <span class="risk-high">🔴 high</span>
```

Symbols:
- `high`: 🔴
- `medium`: 🟡
- `low`: 🟢

## Custom Value Formatters

The renderer includes several built-in formatters:

### Ethereum Address

```go
FormatterAnnotation("ethereum.address")
```

- Adds `0x` prefix
- Converts to lowercase hex
- Pads to 40 characters

### Ethereum Uint256

```go
FormatterAnnotation("ethereum.uint256")
```

- Formats large numbers with commas: `1,000,000,000`

### Hexadecimal

```go
FormatterAnnotation("hex")
```

- Formats values as `0x...` hex strings

### Truncation

```go
FormatterAnnotation("truncate:20")
```

- Truncates strings to specified length
- Adds `...` if truncated

## Template Functions

Templates have access to these functions:

### Annotation Functions

- `getAnnotation .Entity "name"` - Gets annotation by name
- `getAnnotationValue .Entity "name"` - Gets annotation value
- `hasAnnotation .Entity "name"` - Checks if annotation exists
- `hasAnnotations .Entity` - Checks if entity has any annotations

### Formatting Functions

- `formatValue .Param "formatter"` - Applies custom formatter
- `severitySymbol "level"` - Returns severity symbol
- `riskSymbol "level"` - Returns risk symbol

### String Functions

- `indent spaces text` - Indents text
- `upper text` - Uppercase
- `lower text` - Lowercase
- `title text` - Title case
- `trimRight text` - Trim right whitespace
- `join sep items` - Join strings
- `repeat count text` - Repeat text

## Template Examples

### Using Annotations in Text Templates

```go
{{define "call"}}
  ┌─ CALL: {{.Name}}{{if getAnnotation . "render.important"}} ⭐{{end}}
  {{- $severity := getAnnotationValue . "cld.severity"}}
  {{- if $severity}}
  │ Severity: {{severitySymbol $severity}} {{$severity}}
  {{- end}}
  └─────────────────────────
{{end}}
```

### Using Annotations in HTML Templates

```html
{{define "parameter"}}
{{- $important := getAnnotation . "render.important"}}
{{- $emoji := getAnnotationValue . "render.emoji"}}
{{- $formatter := getAnnotationValue . "render.formatter"}}
{{- if $important}}<span class="important">{{end}}
{{- if $emoji}}{{$emoji}} {{end}}
<strong>{{.Name}}</strong> ({{.Type}}): 
<code>{{if $formatter}}{{formatValue . $formatter}}{{else}}{{.Value}}{{end}}</code>
{{- if $important}}</span>{{end}}
{{end}}
```

## Extending the Renderer

### Adding New Formats

1. Create a new format constant:
   ```go
   const FormatMarkdown RenderFormat = "markdown"
   ```

2. Create template directory:
   ```
   internal/templates/markdown/
   ```

3. Create format-specific templates:
   ```
   proposal.tmpl
   batchOperation.tmpl
   call.tmpl
   parameter.tmpl
   ```

### Adding New Formatters

Add formatter logic to `formatParameterValue`:

```go
case "my.custom.formatter":
    return formatMyCustom(value)
```

### Adding New Template Functions

Add functions to `templateFuncs()`:

```go
func templateFuncs() template.FuncMap {
    return template.FuncMap{
        "myFunc": func(arg string) string {
            // implementation
        },
    }
}
```

## Migration from Old Renderer

The new renderer is backward compatible:

```go
// Old code (still works)
renderer, err := internal.NewRenderer()

// New code with explicit format
renderer, err := internal.NewRendererWithFormat(internal.FormatText)
```

The main difference is that annotations are no longer rendered as a separate section. They now control rendering behavior.

## Performance Considerations

- Templates are parsed once during renderer creation
- Templates are embedded in the binary (no filesystem I/O at runtime)
- Large proposals can be rendered directly to a writer to avoid memory allocation:
  ```go
  file, _ := os.Create("output.html")
  renderer.RenderTo(file, proposal)
  ```

## Best Practices

1. **Use annotations to guide rendering** - Don't add annotations just for display
2. **Choose appropriate formats** - Text for CLI, HTML for reports
3. **Leverage formatters** - Use built-in formatters for common types
4. **Custom templates for special cases** - Use templates for domain-specific needs
5. **Stream large outputs** - Use `RenderTo()` for large proposals

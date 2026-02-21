# Analyzer Renderer Component

The renderer component provides a flexible, template-based system for displaying `AnalyzedProposal` instances.

## Overview

The renderer uses Go's `text/template` package to render analyzed MCMS proposals in a hierarchical, human-readable format. It leverages template composition to render nested structures:

```
AnalyzedProposal
  └─ AnalyzedBatchOperation(s)
      └─ AnalyzedCall(s)
          └─ AnalyzedParameter(s)
```

Each level can have annotations that are also rendered.

## Architecture

### Template Hierarchy

The renderer implements a hierarchical template structure:

1. **`proposal`** - Top-level template for the entire proposal
2. **`batchOperation`** - Template for each batch operation within a proposal
3. **`call`** - Template for each call within a batch operation
4. **`parameter`** - Template for each parameter (input/output) within a call
5. **`annotations`** - Shared template for rendering annotations at any level

### Template Composition

Templates use the `{{template "name" .}}` action to embed other templates, creating a composition structure:

```go
// In the proposal template:
{{range .BatchOperations}}
  {{template "batchOperation" .}}
{{end}}

// In the batchOperation template:
{{range .Calls}}
  {{template "call" .}}
{{end}}

// And so on...
```

This approach allows each template to focus on rendering its own level while delegating to child templates for nested structures.

## Usage

### Basic Usage with Default Templates

```go
import "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/analyzer/internal"

// Create renderer with default templates
renderer, err := internal.NewRenderer()
if err != nil {
    return err
}

// Render an analyzed proposal
output, err := renderer.Render(analyzedProposal)
if err != nil {
    return err
}

fmt.Println(output)
```

### Custom Templates

You can provide custom templates to change the output format:

```go
customTemplates := map[string]string{
    "proposal": `{{define "proposal"}}
=== My Custom Proposal Format ===
Total Batch Operations: {{len .BatchOperations}}
{{range .BatchOperations}}{{template "batchOperation" .}}{{end}}
{{end}}`,
    
    "batchOperation": `{{define "batchOperation"}}
--- Batch Operation ---
Calls: {{len .Calls}}
{{range .Calls}}{{template "call" .}}{{end}}
{{end}}`,
    
    // ... more templates ...
}

renderer, err := internal.NewRendererWithTemplates(customTemplates)
```

### Rendering to a Writer

For better performance with large proposals, render directly to a writer:

```go
var buf bytes.Buffer
err := renderer.RenderTo(&buf, analyzedProposal)
if err != nil {
    return err
}
```

## Template Functions

The renderer provides several helper functions available in templates:

- **`indent <spaces> <text>`** - Indents each line of text by the specified number of spaces
- **`trimRight <text>`** - Trims whitespace from the right side
- **`upper <text>`** - Converts text to uppercase
- **`lower <text>`** - Converts text to lowercase
- **`title <text>`** - Converts text to title case
- **`join <separator> <items>`** - Joins string items with a separator
- **`repeat <count> <text>`** - Repeats text N times
- **`hasAnnotations <annotated>`** - Returns true if the object has annotations
- **`severitySymbol <severity>`** - Returns a symbol for severity levels (✗, ⚠, ℹ, ⚙)
- **`riskSymbol <risk>`** - Returns a symbol for risk levels (🔴, 🟡, 🟢)

Example usage in templates:

```go
{{if hasAnnotations .}}
  {{severitySymbol "warning"}} Annotations present
{{end}}
```

## Default Output Format

The default templates produce output like:

```
╔═══════════════════════════════════════════════════════════════════════════════
║ ANALYZED PROPOSAL
╚═══════════════════════════════════════════════════════════════════════════════
Annotations:
  - proposal.id [string]: PROP-001

Batch Operations: 1

─────────────────────────────────────────────────────────────────────────────
 BATCH OPERATION
─────────────────────────────────────────────────────────────────────────────
Annotations:
  - cld.risk [enum]: low

Calls: 1

  ┌───────────────────────────────────────────────────────────────────────────
  │ CALL: transfer
  └───────────────────────────────────────────────────────────────────────────
  Annotations:
    - cld.severity [enum]: info

  Inputs (2):
    • recipient (address): 0x1234567890abcdef
      Annotations:
        - param.note [string]: important parameter
    • amount (uint256): 1000000000000000000

  Outputs (1):
    • success (bool): true
```

## Extending the Renderer

### Adding New Template Functions

To add custom template functions, modify the `templateFuncs()` function in `renderer.go`:

```go
func templateFuncs() template.FuncMap {
    return template.FuncMap{
        // ... existing functions ...
        "myCustomFunc": func(arg string) string {
            // Custom logic
            return result
        },
    }
}
```

### Creating Format-Specific Renderers

You can create specialized renderers for different output formats:

```go
// JSON renderer
func NewJSONRenderer() (*Renderer, error) {
    templates := map[string]string{
        "proposal": `{{define "proposal"}}{"batchOperations": [{{range .BatchOperations}}{{template "batchOperation" .}}{{end}}]}{{end}}`,
        // ... more JSON templates ...
    }
    return NewRendererWithTemplates(templates)
}

// Markdown renderer
func NewMarkdownRenderer() (*Renderer, error) {
    templates := map[string]string{
        "proposal": `{{define "proposal"}}# Analyzed Proposal\n\n{{range .BatchOperations}}{{template "batchOperation" .}}{{end}}{{end}}`,
        // ... more Markdown templates ...
    }
    return NewRendererWithTemplates(templates)
}
```

## Testing

The renderer includes comprehensive tests for:

- Empty proposals
- Proposals with annotations
- Complete proposals with nested structures
- Multiple batch operations
- Custom templates
- Template functions

Run tests with:

```bash
go test ./engine/cld/mcms/analyzer/internal -v -run TestRenderer
```

## Future Enhancements

Potential improvements for the renderer:

1. **Format-specific renderers** - Pre-built renderers for JSON, Markdown, HTML, etc.
2. **Colorization** - Support for terminal color codes in text output
3. **Truncation options** - Ability to truncate large values or limit nesting depth
4. **Template validation** - Pre-validation of custom templates before use
5. **Streaming support** - Render large proposals in chunks
6. **Template library** - Collection of reusable template snippets

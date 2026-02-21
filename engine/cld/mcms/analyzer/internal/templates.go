package internal

// Deprecated: These templates are kept for backward compatibility with tests.
// New code should use file-based templates from the templates/ directory.
// The renderer now loads templates from templates/<format>/*.tmpl files.

// proposalTemplate is the main template for rendering an AnalyzedProposal.
const proposalTemplate = `{{define "proposal" -}}
╔═══════════════════════════════════════════════════════════════════════════════
║ ANALYZED PROPOSAL
╚═══════════════════════════════════════════════════════════════════════════════
{{if hasAnnotations . -}}
{{template "annotations" .}}
{{end -}}
{{- $batchOps := .BatchOperations -}}
{{if $batchOps -}}

Batch Operations: {{len $batchOps}}
{{range $i, $batchOp := $batchOps -}}
{{template "batchOperation" $batchOp}}
{{end -}}
{{else -}}

No batch operations found.
{{end -}}
{{end}}`

// batchOperationTemplate is the template for rendering an AnalyzedBatchOperation.
const batchOperationTemplate = `{{define "batchOperation" -}}

─────────────────────────────────────────────────────────────────────────────
 BATCH OPERATION
─────────────────────────────────────────────────────────────────────────────
{{if hasAnnotations . -}}
{{template "annotations" .}}
{{end -}}
{{- $calls := .Calls -}}
{{if $calls -}}

Calls: {{len $calls}}
{{range $i, $call := $calls -}}
{{template "call" $call}}
{{end -}}
{{else -}}

No calls found.
{{end -}}
{{end}}`

// callTemplate is the template for rendering an AnalyzedCall.
const callTemplate = `{{define "call" -}}

  ┌───────────────────────────────────────────────────────────────────────────
  │ CALL: {{.Name}}
  └───────────────────────────────────────────────────────────────────────────
{{if hasAnnotations . -}}
  {{template "annotations" .}}
{{end -}}
{{- $inputs := .Inputs -}}
{{if $inputs -}}

  Inputs ({{len $inputs}}):
{{range $i, $input := $inputs -}}
    {{template "parameter" $input}}
{{end -}}
{{else -}}

  No inputs.
{{end -}}
{{- $outputs := .Outputs -}}
{{if $outputs -}}

  Outputs ({{len $outputs}}):
{{range $i, $output := $outputs -}}
    {{template "parameter" $output}}
{{end -}}
{{else -}}

  No outputs.
{{end -}}
{{end}}`

// parameterTemplate is the template for rendering an AnalyzedParameter.
const parameterTemplate = `{{define "parameter" -}}
• {{.Name}} ({{.Type}}): {{.Value}}
{{- if hasAnnotations .}}
  {{template "annotations" .}}
{{- end}}
{{- end}}`

// annotationsTemplate is the template for rendering annotations.
const annotationsTemplate = `{{define "annotations" -}}
{{$annotations := .Annotations -}}
Annotations:
{{range $i, $annotation := $annotations -}}
  - {{$annotation.Name}} [{{$annotation.Type}}]: {{$annotation.Value}}
{{end -}}
{{end}}`

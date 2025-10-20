package analyzer

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// MarkdownRenderer renders ProposalReport and Descriptors as Markdown using templates.
type MarkdownRenderer struct {
	proposalTmpl    *template.Template
	timelockTmpl    *template.Template
	decodedCallTmpl *template.Template
	detailsTmpl     *template.Template
	summaryTmpl     *template.Template
}

const (
	Indent = "    "

	// Length thresholds for different rendering decisions
	MaxSummaryLength      = 80  // Maximum length for inline summaries
	MaxSimpleValueLength  = 50  // Maximum length for values considered "simple"
	MaxLongValueLength    = 120 // Threshold for values that need details section
	MaxHexPreviewBytes    = 16  // Maximum bytes to show in hex preview
	MaxCompactValueLength = 24  // Maximum length for compact value previews
	MaxCompactHexBytes    = 4   // Maximum bytes for compact hex previews
)

// Template constants for Markdown rendering
const (
	proposalTemplate = `{{range $i, $op := .Operations}}Operation #{{$i}}
Chain selector: {{$op.ChainSelector}} ({{$op.ChainName}})
{{range $call := $op.Calls}}{{indent (renderCall $call $.Context)}}{{end}}
{{end}}`

	timelockTemplate = `{{range $i, $batch := .Batches}}### Batch {{$i}}
**Chain selector:** ` + "`{{$batch.ChainSelector}}`" + ` ({{$batch.ChainName}})

{{range $j, $op := $batch.Operations}}#### Operation {{$j}}
{{range $call := $op.Calls}}{{renderCall $call $.Context}}

{{end}}{{end}}---

{{end}}`

	decodedCallTemplate = `**Address:** ` + "`{{.Address}}`" + `{{if .AddressAnnotation}} <sub><i>{{.AddressAnnotation}}</i></sub>{{end}}
**Method:** ` + "`{{.Method}}`" + `

{{if .Inputs}}**Inputs:**

{{range .Inputs}}- ` + "`{{.Name}}`" + `: {{.Summary}}{{if .Annotation}} <sub><i>{{.Annotation}}</i></sub>{{end}}
{{end}}
{{range .InputDetails}}{{.}}
{{end}}{{end}}{{if .Outputs}}**Outputs:**

{{range .Outputs}}- ` + "`{{.Name}}`" + `: {{.Summary}}{{if .Annotation}} <sub><i>{{.Annotation}}</i></sub>{{end}}
{{end}}
{{range .OutputDetails}}{{.}}
{{end}}{{end}}`

	detailsTemplate = `<details><summary>{{.Name}}</summary>

` + "```" + `
{{.Content}}
` + "```" + `
</details>
`

	summaryTemplate = `{{if .Type}}` +
		`{{.Type}}{{if .Length}}(len={{.Length}}){{end}}` +
		`{{if .Preview}}{{if hasPrefix .Preview ":"}}{{.Preview}}{{else}}: {{.Preview}}{{end}}{{end}}` +
		`{{else}}` + "`{{.Value}}`" + `{{end}}`
)

type ProposalTemplateData struct {
	Operations []OperationTemplateData
	Context    *DescriptorContext
}

type OperationTemplateData struct {
	ChainSelector uint64
	ChainName     string
	Calls         []*DecodedCall
}

type TimelockTemplateData struct {
	Batches []BatchTemplateData
	Context *DescriptorContext
}

type BatchTemplateData struct {
	ChainSelector uint64
	ChainName     string
	Operations    []OperationTemplateData
}

type DecodedCallTemplateData struct {
	Address           string
	AddressAnnotation string
	Method            string
	Inputs            []ArgumentTemplateData
	InputDetails      []string
	Outputs           []ArgumentTemplateData
	OutputDetails     []string
}

type ArgumentTemplateData struct {
	Name       string
	Summary    string
	Annotation string
	Details    string
}

type SummaryTemplateData struct {
	Type    string
	Length  int
	Preview string
	Value   string
}

type DetailsTemplateData struct {
	Name    string
	Content string
}

// NewMarkdownRenderer creates a new MarkdownRenderer with compiled templates.
func NewMarkdownRenderer() *MarkdownRenderer {
	r := &MarkdownRenderer{}
	r.initTemplates()

	return r
}

// RenderProposal renders a ProposalReport as Markdown.
func (r *MarkdownRenderer) RenderProposal(rep *ProposalReport, ctx *DescriptorContext) string {
	data := ProposalTemplateData{
		Operations: make([]OperationTemplateData, len(rep.Operations)),
		Context:    ctx,
	}

	for i, op := range rep.Operations {
		data.Operations[i] = OperationTemplateData{
			ChainSelector: op.ChainSelector,
			ChainName:     op.ChainName,
			Calls:         op.Calls,
		}
	}

	var buf bytes.Buffer
	if err := r.proposalTmpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("Error rendering proposal: %v", err)
	}

	return buf.String()
}

// RenderTimelock renders a Timelock ProposalReport as Markdown.
func (r *MarkdownRenderer) RenderTimelock(rep *ProposalReport, ctx *DescriptorContext) string {
	data := TimelockTemplateData{
		Batches: make([]BatchTemplateData, len(rep.Batches)),
		Context: ctx,
	}

	for i, batch := range rep.Batches {
		operations := make([]OperationTemplateData, len(batch.Operations))
		for j, op := range batch.Operations {
			operations[j] = OperationTemplateData{
				ChainSelector: op.ChainSelector,
				ChainName:     op.ChainName,
				Calls:         op.Calls,
			}
		}

		data.Batches[i] = BatchTemplateData{
			ChainSelector: batch.ChainSelector,
			ChainName:     batch.ChainName,
			Operations:    operations,
		}
	}

	var buf bytes.Buffer
	if err := r.timelockTmpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("Error rendering timelock: %v", err)
	}

	return buf.String()
}

// RenderDecodedCall renders a DecodedCall as Markdown.
func (r *MarkdownRenderer) RenderDecodedCall(d *DecodedCall, ctx *DescriptorContext) string {
	addrAnn := AddressDescriptor{Value: d.Address}.Annotation(ctx)

	data := DecodedCallTemplateData{
		Address:           d.Address,
		AddressAnnotation: addrAnn,
		Method:            d.Method,
		Inputs:            make([]ArgumentTemplateData, len(d.Inputs)),
		InputDetails:      []string{},
		Outputs:           make([]ArgumentTemplateData, len(d.Outputs)),
		OutputDetails:     []string{},
	}

	// Process inputs
	for i, input := range d.Inputs {
		summary, details := r.summarizeDescriptor(input.Name, input.Value, ctx)
		annotation := ""
		if addr, ok := input.Value.(AddressDescriptor); ok {
			annotation = addr.Annotation(ctx)
		}

		data.Inputs[i] = ArgumentTemplateData{
			Name:       input.Name,
			Summary:    summary,
			Annotation: annotation,
			Details:    details,
		}

		if details != "" {
			data.InputDetails = append(data.InputDetails, details)
		}
	}

	// Process outputs
	for i, output := range d.Outputs {
		summary, details := r.summarizeDescriptor(output.Name, output.Value, ctx)
		annotation := ""
		if addr, ok := output.Value.(AddressDescriptor); ok {
			annotation = addr.Annotation(ctx)
		}

		data.Outputs[i] = ArgumentTemplateData{
			Name:       output.Name,
			Summary:    summary,
			Annotation: annotation,
			Details:    details,
		}

		if details != "" {
			data.OutputDetails = append(data.OutputDetails, details)
		}
	}

	var buf bytes.Buffer
	if err := r.decodedCallTmpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("Error rendering decoded call: %v", err)
	}

	return buf.String()
}

// Helper functions for Markdown rendering

// initTemplates compiles all templates with their helper functions.
func (r *MarkdownRenderer) initTemplates() {
	funcMap := template.FuncMap{
		"indent":         indentString,
		"renderCall":     r.renderCallHelper,
		"hexPreview":     hexPreview,
		"compactValue":   compactValue,
		"truncateMiddle": truncateMiddle,
		"hasPrefix":      func(s, prefix string) bool { return strings.HasPrefix(s, prefix) },
	}

	r.proposalTmpl = template.Must(template.New("proposal").Funcs(funcMap).Parse(proposalTemplate))
	r.timelockTmpl = template.Must(template.New("timelock").Funcs(funcMap).Parse(timelockTemplate))
	r.decodedCallTmpl = template.Must(template.New("decodedCall").Funcs(funcMap).Parse(decodedCallTemplate))
	r.detailsTmpl = template.Must(template.New("details").Funcs(funcMap).Parse(detailsTemplate))
	r.summaryTmpl = template.Must(template.New("summary").Funcs(funcMap).Parse(summaryTemplate))
}

// renderCallHelper is a template helper function to render a DecodedCall.
func (r *MarkdownRenderer) renderCallHelper(call *DecodedCall, ctx *DescriptorContext) string {
	return r.RenderDecodedCall(call, ctx)
}

// renderSummary renders a summary using the summary template.
func (r *MarkdownRenderer) renderSummary(data SummaryTemplateData) string {
	var buf bytes.Buffer
	if err := r.summaryTmpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("Error rendering summary: %v", err)
	}

	return buf.String()
}

// renderDetails renders details HTML using the details template.
func (r *MarkdownRenderer) renderDetails(name, content string) string {
	data := DetailsTemplateData{
		Name:    name,
		Content: content,
	}
	var buf bytes.Buffer
	if err := r.detailsTmpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("Error rendering details: %v", err)
	}

	return buf.String()
}

// summarizeDescriptor produces a short summary and optional detailed description for an argument.
func (r *MarkdownRenderer) summarizeDescriptor(name string, descriptor Descriptor, ctx *DescriptorContext) (summary string, details string) {
	switch v := descriptor.(type) {
	case AddressDescriptor:
		return fmt.Sprintf("`%s`", v.Value), ""
	case ChainSelectorDescriptor:
		return fmt.Sprintf("`%s`", v.Describe(ctx)), ""
	case BytesDescriptor:
		n := len(v.Value)
		preview := hexPreview(v.Value, MaxHexPreviewBytes)
		summary = fmt.Sprintf("bytes(len=%d): %s", n, preview)
		details = r.renderDetails(name, hexutil.Encode(v.Value))

		return summary, details
	case ArrayDescriptor:
		n := len(v.Elements)
		if n == 0 {
			return "[]", ""
		}
		preview := arrayPreview(v.Elements, ctx)
		summaryData := SummaryTemplateData{
			Type:    fmt.Sprintf("array[%d]", n),
			Preview: preview,
		}
		summary = r.renderSummary(summaryData)
		details = r.renderDetails(name, v.Describe(ctx))

		return summary, details
	case StructDescriptor:
		summaryData := SummaryTemplateData{
			Type: fmt.Sprintf("struct{%d fields}", len(v.Fields)),
		}
		summary = r.renderSummary(summaryData)
		details = r.renderDetails(name, v.Describe(ctx))

		return summary, details
	case SimpleDescriptor:
		s := v.Value
		if len(s) > MaxSummaryLength {
			summary = fmt.Sprintf("`%s` (len=%d)", truncateMiddle(s, MaxSummaryLength), len(s))
			details = r.renderDetails(name, s)

			return summary, details
		}

		return fmt.Sprintf("`%s`", s), ""
	case YamlDescriptor:
		// YamlDescriptor values should always generate details since they're complex YAML-marshaled values
		s := v.Describe(ctx)
		s = strings.TrimRight(s, " \t\n\r")
		if strings.Contains(s, "\n") || len(s) > MaxLongValueLength {
			summary = fmt.Sprintf("`%s`", truncateMiddle(strings.ReplaceAll(s, "\n", " "), MaxSummaryLength))
			details = r.renderDetails(name, s)

			return summary, details
		}
		// Check if this is a simple value that should be displayed without backticks
		if r.isSimpleValue(s) {
			summary = s
			details = r.renderDetails(name, s)

			return summary, details
		}
		// Complex value - display with backticks and details
		summary = fmt.Sprintf("`%s`", s)
		details = r.renderDetails(name, s)

		return summary, details
	default:
		// Mixed case - template for details, fmt.Sprintf for summary
		s := descriptor.Describe(ctx)
		// Trim trailing whitespace
		s = strings.TrimRight(s, " \t\n\r")

		if strings.Contains(s, "\n") || len(s) > MaxLongValueLength {
			summary = fmt.Sprintf("`%s`", truncateMiddle(strings.ReplaceAll(s, "\n", " "), MaxSummaryLength))
			details = r.renderDetails(name, s)

			return summary, details
		}

		// Determine if this is a simple value that should be displayed without backticks
		if r.isSimpleValue(s) {
			summary = s
			details = r.renderDetails(name, s)

			return summary, details
		}

		// Complex value - display with backticks, no details for default case
		return fmt.Sprintf("`%s`", s), ""
	}
}

// isSimpleValue determines if a string represents a simple value that should be displayed without backticks.
// Simple values are typically:
// - Numbers (123, 0x1234)
// - Empty arrays/objects ([])
// - Short, clean values without complex formatting
// - Values that don't contain descriptive text or colons
func (r *MarkdownRenderer) isSimpleValue(s string) bool {
	// Empty string is not simple (should be quoted)
	if s == "" {
		return false
	}

	// Contains backticks - definitely not simple
	if strings.Contains(s, "`") {
		return false
	}

	// Contains newlines - not simple
	if strings.Contains(s, "\n") {
		return false
	}

	// Very long strings are not simple
	if len(s) > MaxSimpleValueLength {
		return false
	}

	// Contains descriptive text patterns (name: value, etc.)
	if strings.Contains(s, ": ") {
		return false
	}

	// Contains multiple words separated by spaces (likely descriptive text)
	words := strings.Fields(s)
	if len(words) > 1 {
		return false
	}

	// Long hex values (like 0x6d636d0000000000000000000000000000000000000000000000000000000000)
	// should not be considered simple - they need details
	if strings.HasPrefix(s, "0x") && len(s) > 20 {
		return false
	}

	// Single word or simple patterns
	return true
}

// indentString indents each line of the input string with the default indent.
func indentString(s string) string {
	return indentStringWith(s, Indent)
}

// indentStringWith indents each line of the input string with the specified indent.
func indentStringWith(s string, indent string) string {
	result := &strings.Builder{}
	components := strings.Split(s, "\n")
	for i, component := range components {
		result.WriteString(indent)
		result.WriteString(component)
		if i < len(components)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

func arrayPreview(elems []Descriptor, ctx *DescriptorContext) string {
	n := len(elems)
	if n == 0 {
		return ""
	}
	maxVal := 3
	if n < maxVal {
		maxVal = n
	}
	parts := make([]string, 0, maxVal)
	for i := range maxVal {
		parts = append(parts, compactValue(elems[i], ctx))
	}
	more := ""
	if n > maxVal {
		more = fmt.Sprintf(", … (+%d)", n-maxVal)
	}

	return fmt.Sprintf(": [%s%s]", strings.Join(parts, ", "), more)
}

// compactValue produces a very short representation for an argument, suitable for inline previews.
func compactValue(arg Descriptor, ctx *DescriptorContext) string {
	switch v := arg.(type) {
	case AddressDescriptor:
		return v.Value
	case ChainSelectorDescriptor:
		return v.Describe(ctx)
	case BytesDescriptor:
		return hexPreview(v.Value, MaxCompactHexBytes)
	case SimpleDescriptor:
		return truncateMiddle(v.Value, MaxCompactValueLength)
	case StructDescriptor:
		return "struct"
	case ArrayDescriptor:
		return fmt.Sprintf("array[%d]", len(v.Elements))
	default:
		s := arg.Describe(ctx)
		return truncateMiddle(strings.ReplaceAll(s, "\n", " "), MaxCompactValueLength)
	}
}

// hexPreview returns a hex string for the first maxBytes of b (or full if shorter),
// using middle truncation when needed and always including the 0x prefix.
func hexPreview(b []byte, maxBytes int) string {
	if len(b) <= maxBytes {
		return hexutil.Encode(b)
	}
	// encode both ends for clearer preview when data is large
	head := hexutil.Encode(b[:maxBytes])
	tailLen := maxBytes
	if len(b) >= maxBytes {
		tail := hexutil.Encode(b[len(b)-tailLen:])
		return fmt.Sprintf("%s…%s", head, tail)
	}

	return head + "…"
}

// truncateMiddle shortens a string to at most max characters by keeping the start and end.
func truncateMiddle(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	// split budget roughly in half around an ellipsis
	keep := (maxLen - 1) / 2
	left := s[:keep]
	right := s[len(s)-keep:]

	return left + "…" + right
}

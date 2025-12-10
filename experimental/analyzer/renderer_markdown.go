package analyzer

import (
	"bytes"
	"embed"
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"gopkg.in/yaml.v3"
)

//go:embed templates/markdown/*.tmpl
var templateFS embed.FS

// Verify MarkdownRenderer implements Renderer interface
var _ Renderer = (*MarkdownRenderer)(nil)

// MarkdownRenderer extends TextRenderer with markdown-specific formatting
type MarkdownRenderer struct {
	*TextRenderer
	proposalTmpl     *template.Template
	timelockTmpl     *template.Template
	decodedCallTmpl  *template.Template
	detailsTmpl      *template.Template
	summaryTmpl      *template.Template
	addressFieldTmpl *template.Template
	bytesFieldTmpl   *template.Template
	arrayFieldTmpl   *template.Template
	structFieldTmpl  *template.Template
	simpleFieldTmpl  *template.Template
	yamlFieldTmpl    *template.Template
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

type ProposalTemplateData struct {
	Operations []OperationTemplateData
	Context    *FieldContext
}

type OperationTemplateData struct {
	ChainSelector uint64
	ChainName     string
	Calls         []*DecodedCall
}

type TimelockTemplateData struct {
	Batches []BatchTemplateData
	Context *FieldContext
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
	Outputs           []ArgumentTemplateData
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

// Field-specific template data structures
type AddressFieldData struct {
	Value string
}

type BytesFieldData struct {
	Value  []byte
	Length int
}

type ArrayFieldData struct {
	Elements []FieldValue
	Length   int
	Context  *FieldContext
}

type StructFieldData struct {
	FieldCount int
}

type SimpleFieldData struct {
	Value string
}

type YamlFieldData struct {
	Value string
}

// NewMarkdownRenderer creates a new MarkdownRenderer with compiled templates
func NewMarkdownRenderer() *MarkdownRenderer {
	r := &MarkdownRenderer{
		TextRenderer: NewTextRenderer(),
	}
	r.initTemplates()

	return r
}

// RenderProposal renders a ProposalReport as Markdown.
func (r *MarkdownRenderer) RenderProposal(rep *ProposalReport, ctx *FieldContext) string {
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
func (r *MarkdownRenderer) RenderTimelockProposal(rep *ProposalReport, ctx *FieldContext) string {
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

// RenderDecodedCall renders a DecodedCall as Markdown using templates.
func (r *MarkdownRenderer) RenderDecodedCall(d *DecodedCall, ctx *FieldContext) string {
	addrAnn := AddressField{Value: d.Address}.Annotation(ctx)

	data := DecodedCallTemplateData{
		Address:           d.Address,
		AddressAnnotation: addrAnn,
		Method:            d.Method,
		Inputs:            make([]ArgumentTemplateData, len(d.Inputs)),
		Outputs:           make([]ArgumentTemplateData, len(d.Outputs)),
	}

	// Process inputs
	for i, input := range d.Inputs {
		summary, details := r.summarizeField(input.Name, input.Value, ctx)
		annotation := ""
		if addr, ok := input.Value.(AddressField); ok {
			annotation = addr.Annotation(ctx)
		}

		data.Inputs[i] = ArgumentTemplateData{
			Name:       input.Name,
			Summary:    summary,
			Annotation: annotation,
			Details:    details,
		}
	}

	// Process outputs
	for i, output := range d.Outputs {
		summary, details := r.summarizeField(output.Name, output.Value, ctx)
		annotation := ""
		if addr, ok := output.Value.(AddressField); ok {
			annotation = addr.Annotation(ctx)
		}

		data.Outputs[i] = ArgumentTemplateData{
			Name:       output.Name,
			Summary:    summary,
			Annotation: annotation,
			Details:    details,
		}
	}

	var buf bytes.Buffer
	if err := r.decodedCallTmpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("Error rendering decoded call: %v", err)
	}

	return buf.String()
}

// Helper functions for Markdown rendering

// initTemplates compiles all templates with their helper functions
func (r *MarkdownRenderer) initTemplates() {
	funcMap := template.FuncMap{
		"indent":         indentString,
		"renderCall":     r.renderCallHelper,
		"renderField":    r.renderFieldHelper,
		"hexPreview":     hexPreview,
		"compactValue":   compactValue,
		"truncateMiddle": truncateMiddle,
		"hasPrefix":      func(s, prefix string) bool { return strings.HasPrefix(s, prefix) },
		"contains":       strings.Contains,
		"replace":        strings.ReplaceAll,
		"len":            func(s string) int { return len(s) },
		"gt":             func(a, b int) bool { return a > b },
		"sub":            func(a, b int) int { return a - b },
		"isSimpleValue":  r.isSimpleValue,
	}

	// Load templates from filesystem
	r.proposalTmpl = template.Must(template.New("proposal.tmpl").Funcs(funcMap).ParseFS(templateFS, "templates/markdown/proposal.tmpl"))
	r.timelockTmpl = template.Must(template.New("timelock_proposal.tmpl").Funcs(funcMap).ParseFS(templateFS, "templates/markdown/timelock_proposal.tmpl"))
	r.decodedCallTmpl = template.Must(template.New("decoded_call.tmpl").Funcs(funcMap).ParseFS(templateFS, "templates/markdown/decoded_call.tmpl"))
	r.detailsTmpl = template.Must(template.New("details.tmpl").Funcs(funcMap).ParseFS(templateFS, "templates/markdown/details.tmpl"))
	r.summaryTmpl = template.Must(template.New("summary.tmpl").Funcs(funcMap).ParseFS(templateFS, "templates/markdown/summary.tmpl"))

	// Field-specific templates
	r.addressFieldTmpl = template.Must(template.New("address_field.tmpl").Funcs(funcMap).ParseFS(templateFS, "templates/markdown/address_field.tmpl"))
	r.bytesFieldTmpl = template.Must(template.New("bytes_field.tmpl").Funcs(funcMap).ParseFS(templateFS, "templates/markdown/bytes_field.tmpl"))
	r.arrayFieldTmpl = template.Must(template.New("array_field.tmpl").Funcs(funcMap).ParseFS(templateFS, "templates/markdown/array_field.tmpl"))
	r.structFieldTmpl = template.Must(template.New("struct_field.tmpl").Funcs(funcMap).ParseFS(templateFS, "templates/markdown/struct_field.tmpl"))
	r.simpleFieldTmpl = template.Must(template.New("simple_field.tmpl").Funcs(funcMap).ParseFS(templateFS, "templates/markdown/simple_field.tmpl"))
	r.yamlFieldTmpl = template.Must(template.New("yaml_field.tmpl").Funcs(funcMap).ParseFS(templateFS, "templates/markdown/yaml_field.tmpl"))
}

// renderCallHelper is a template helper function to render a DecodedCall
func (r *MarkdownRenderer) renderCallHelper(call *DecodedCall, ctx *FieldContext) string {
	return r.RenderDecodedCall(call, ctx)
}

// renderFieldHelper is a template helper function to render a FieldValue
func (r *MarkdownRenderer) renderFieldHelper(field FieldValue, ctx *FieldContext) string {
	summary, _ := r.summarizeField("", field, ctx)
	return summary
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

// summarizeField uses templates for rendering different field types
func (r *MarkdownRenderer) summarizeField(name string, field FieldValue, ctx *FieldContext) (summary string, details string) {
	switch f := field.(type) {
	case AddressField:
		return r.summarizeAddressField(f)

	case BytesField:
		return r.summarizeBytesField(name, f)

	case ArrayField:
		return r.summarizeArrayField(name, f, ctx)

	case StructField:
		return r.summarizeStructField(name, f)

	case SimpleField:
		return r.summarizeSimpleField(name, f)

	case YamlField:
		return r.summarizeYamlField(name, f)

	case NamedField:
		return r.summarizeNamedField(f, ctx)

	default:
		// Fallback to text renderer
		summary = r.renderFieldValueDirect(field, ctx)
		return summary, ""
	}
}

// Helper methods for field summarization

func (r *MarkdownRenderer) summarizeAddressField(f AddressField) (string, string) {
	data := AddressFieldData{Value: f.GetValue()}
	summary := r.renderTemplate(r.addressFieldTmpl, data)

	return summary, ""
}

func (r *MarkdownRenderer) summarizeBytesField(name string, f BytesField) (string, string) {
	data := BytesFieldData{Value: f.GetValue(), Length: f.GetLength()}
	summary := r.renderTemplate(r.bytesFieldTmpl, data)
	details := r.renderDetails(name, hexutil.Encode(f.GetValue()))

	return summary, details
}

func (r *MarkdownRenderer) summarizeArrayField(name string, f ArrayField, ctx *FieldContext) (string, string) {
	data := ArrayFieldData{
		Elements: f.GetElements(),
		Length:   f.GetLength(),
		Context:  ctx,
	}
	summary := r.renderTemplate(r.arrayFieldTmpl, data)
	var details string
	if f.GetLength() > 0 {
		details = r.renderDetails(name, r.renderFieldDetails(f, ""))
	}

	return summary, details
}

func (r *MarkdownRenderer) summarizeStructField(name string, f StructField) (string, string) {
	data := StructFieldData{FieldCount: f.GetFieldCount()}
	summary := r.renderTemplate(r.structFieldTmpl, data)
	details := r.renderDetails(name, r.renderFieldDetails(f, ""))

	return summary, details
}

func (r *MarkdownRenderer) summarizeSimpleField(name string, f SimpleField) (string, string) {
	data := SimpleFieldData{Value: f.GetValue()}
	summary := r.renderTemplate(r.simpleFieldTmpl, data)
	var details string
	if len(f.GetValue()) > MaxSummaryLength {
		details = r.renderDetails(name, f.GetValue())
	}

	return summary, details
}

func (r *MarkdownRenderer) summarizeYamlField(name string, f YamlField) (string, string) {
	data := YamlFieldData{Value: f.GetValue()}
	summary := r.renderTemplate(r.yamlFieldTmpl, data)
	details := r.renderDetails(name, r.renderFieldDetails(f, ""))

	return summary, details
}

func (r *MarkdownRenderer) summarizeNamedField(f NamedField, ctx *FieldContext) (string, string) {
	valueStr := r.renderFieldValueDirect(f.Value, ctx)
	if strings.HasPrefix(valueStr, "`") && strings.HasSuffix(valueStr, "`") && len(valueStr) > 1 {
		valueStr = valueStr[1 : len(valueStr)-1]
	}
	summary := fmt.Sprintf("`%s: %s`", f.Name, valueStr)

	return summary, ""
}

// renderFieldDetails renders the full content for details sections with proper formatting
func (r *MarkdownRenderer) renderFieldDetails(field FieldValue, indent string) string {
	switch f := field.(type) {
	case ArrayField:
		return r.renderArrayFieldDetails(f, indent)

	case StructField:
		return r.renderStructFieldDetails(f, indent)

	case BytesField:
		return r.renderBytesFieldDetails(f)

	case SimpleField:
		return r.renderSimpleFieldDetails(f)

	case YamlField:
		return r.renderYamlFieldDetails(f)

	case AddressField:
		return r.renderAddressFieldDetails(f)

	case ChainSelectorField:
		return r.renderChainSelectorFieldDetails(f)

	case NamedField:
		return r.renderNamedFieldDetails(f, indent)

	default:
		// Fallback to string representation without backticks
		return fmt.Sprintf("%v", field)
	}
}

// Helper methods for field details rendering

func (r *MarkdownRenderer) renderArrayFieldDetails(f ArrayField, indent string) string {
	if f.GetLength() == 0 {
		return "[]"
	}
	parts := make([]string, 0, f.GetLength())
	for i, elem := range f.GetElements() {
		elemStr := r.renderFieldDetails(elem, indent+"  ")
		parts = append(parts, fmt.Sprintf("%s%d: %s", indent+"  ", i, elemStr))
	}

	return fmt.Sprintf("[\n%s\n%s]", strings.Join(parts, "\n"), indent)
}

func (r *MarkdownRenderer) renderStructFieldDetails(f StructField, indent string) string {
	fields := f.GetFields()
	if len(fields) == 0 {
		return fmt.Sprintf("struct with %d fields (no field data available)", f.GetFieldCount())
	}
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		valueStr := r.renderFieldDetails(field.Value, indent+"  ")
		parts = append(parts, fmt.Sprintf("%s%s: %s", indent+"  ", field.Name, valueStr))
	}

	return fmt.Sprintf("{\n%s\n%s}", strings.Join(parts, "\n"), indent)
}

func (r *MarkdownRenderer) renderBytesFieldDetails(f BytesField) string {
	return hexutil.Encode(f.GetValue())
}

func (r *MarkdownRenderer) renderSimpleFieldDetails(f SimpleField) string {
	return f.GetValue()
}

func (r *MarkdownRenderer) renderYamlFieldDetails(f YamlField) string {
	if str, ok := f.Value.(string); ok {
		var data interface{}
		if err := yaml.Unmarshal([]byte(str), &data); err == nil {
			if pretty, err := yaml.Marshal(data); err == nil {
				content := strings.TrimRight(string(pretty), "\n")
				content = strings.ReplaceAll(content, "- ", "&#45; ")

				return content
			}
		}

		return str
	}

	if pretty, err := yaml.Marshal(f.Value); err == nil {
		content := strings.TrimRight(string(pretty), "\n")
		content = strings.ReplaceAll(content, "- ", "&#45; ")

		return content
	}

	return f.GetValue()
}

func (r *MarkdownRenderer) renderAddressFieldDetails(f AddressField) string {
	return f.GetValue()
}

func (r *MarkdownRenderer) renderChainSelectorFieldDetails(f ChainSelectorField) string {
	chainName, err := GetChainNameBySelector(f.GetValue())
	if err != nil || chainName == "" {
		return fmt.Sprintf("%d (<chain unknown>)", f.GetValue())
	}

	return fmt.Sprintf("%d (%s)", f.GetValue(), chainName)
}

func (r *MarkdownRenderer) renderNamedFieldDetails(f NamedField, indent string) string {
	valueStr := r.renderFieldDetails(f.Value, indent+"  ")
	return fmt.Sprintf("%s: %s", f.Name, valueStr)
}

// renderFieldValueDirect renders a field value directly without causing recursion
func (r *MarkdownRenderer) renderFieldValueDirect(field FieldValue, ctx *FieldContext) string {
	switch f := field.(type) {
	case AddressField:
		return r.renderAddressFieldDirect(f)

	case BytesField:
		return r.renderBytesFieldDirect(f)

	case ArrayField:
		return r.renderArrayFieldDirect(f, ctx)

	case StructField:
		return r.renderStructFieldDirect(f)

	case SimpleField:
		return r.renderSimpleFieldDirect(f)

	case ChainSelectorField:
		return r.renderChainSelectorFieldDirect(f)

	case YamlField:
		return r.renderYamlFieldDirect(f)

	case NamedField:
		return r.renderNamedFieldDirect(f, ctx)

	default:
		// Fallback to string representation
		return fmt.Sprintf("`%v`", field)
	}
}

// Helper methods for direct field value rendering

func (r *MarkdownRenderer) renderAddressFieldDirect(f AddressField) string {
	data := AddressFieldData{Value: f.GetValue()}
	return r.renderTemplate(r.addressFieldTmpl, data)
}

func (r *MarkdownRenderer) renderBytesFieldDirect(f BytesField) string {
	data := BytesFieldData{Value: f.GetValue(), Length: f.GetLength()}
	return r.renderTemplate(r.bytesFieldTmpl, data)
}

func (r *MarkdownRenderer) renderArrayFieldDirect(f ArrayField, ctx *FieldContext) string {
	data := ArrayFieldData{
		Elements: f.GetElements(),
		Length:   f.GetLength(),
		Context:  ctx,
	}

	return r.renderTemplate(r.arrayFieldTmpl, data)
}

func (r *MarkdownRenderer) renderStructFieldDirect(f StructField) string {
	data := StructFieldData{FieldCount: f.GetFieldCount()}
	return r.renderTemplate(r.structFieldTmpl, data)
}

func (r *MarkdownRenderer) renderSimpleFieldDirect(f SimpleField) string {
	data := SimpleFieldData{Value: f.GetValue()}
	return r.renderTemplate(r.simpleFieldTmpl, data)
}

func (r *MarkdownRenderer) renderChainSelectorFieldDirect(f ChainSelectorField) string {
	chainName, err := GetChainNameBySelector(f.GetValue())
	if err != nil || chainName == "" {
		return fmt.Sprintf("`%d (<chain unknown>)`", f.GetValue())
	}

	return fmt.Sprintf("`%d (%s)`", f.GetValue(), chainName)
}

func (r *MarkdownRenderer) renderYamlFieldDirect(f YamlField) string {
	return fmt.Sprintf("`%s`", f.GetValue())
}

func (r *MarkdownRenderer) renderNamedFieldDirect(f NamedField, ctx *FieldContext) string {
	valueStr := r.renderFieldValueDirect(f.Value, ctx)
	if strings.HasPrefix(valueStr, "`") && strings.HasSuffix(valueStr, "`") && len(valueStr) > 1 {
		valueStr = valueStr[1 : len(valueStr)-1]
	}

	return fmt.Sprintf("`%s: %s`", f.Name, valueStr)
}

// renderTemplate is a helper to execute a template with data
func (r *MarkdownRenderer) renderTemplate(tmpl *template.Template, data interface{}) string {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("Error rendering template: %v", err)
	}

	out := buf.String()
	if !strings.HasSuffix(out, "\n\n") { // assuming double newline means intentional spacing
		out = strings.TrimSuffix(out, "\n")
	}

	return out
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

func arrayPreview(elems []FieldValue, ctx *FieldContext) string {
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

// compactValue produces a very short representation for a field, suitable for inline previews
func compactValue(field FieldValue, ctx *FieldContext) string {
	switch f := field.(type) {
	case AddressField:
		return f.GetValue()
	case ChainSelectorField:
		chainName, err := GetChainNameBySelector(f.GetValue())
		if err != nil || chainName == "" {
			return strconv.FormatUint(f.GetValue(), 10)
		}

		return chainName
	case BytesField:
		return hexPreview(f.GetValue(), MaxCompactHexBytes)
	case SimpleField:
		return truncateMiddle(f.GetValue(), MaxCompactValueLength)
	case StructField:
		return "struct"
	case ArrayField:
		return fmt.Sprintf("array[%d]", f.GetLength())
	default:
		return truncateMiddle(fmt.Sprintf("<%s>", f.GetType()), MaxCompactValueLength)
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

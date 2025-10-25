package analyzer

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

//go:embed templates/text/*.tmpl
var textTemplateFS embed.FS

// Verify TextRenderer implements Renderer interface
var _ Renderer = (*TextRenderer)(nil)

// TextRenderer provides basic text rendering functionality using templates
type TextRenderer struct {
	indent                 string
	proposalTmpl           *template.Template
	timelockTmpl           *template.Template
	decodedCallTmpl        *template.Template
	simpleFieldTmpl        *template.Template
	addressFieldTmpl       *template.Template
	bytesFieldTmpl         *template.Template
	chainSelectorFieldTmpl *template.Template
	yamlFieldTmpl          *template.Template
	arrayFieldTmpl         *template.Template
	structFieldTmpl        *template.Template
	namedFieldTmpl         *template.Template
}

// NewTextRenderer creates a new TextRenderer
func NewTextRenderer() *TextRenderer {
	r := &TextRenderer{
		indent: "    ",
	}
	r.initTemplates()

	return r
}

// initTemplates compiles all templates with their helper functions
func (r *TextRenderer) initTemplates() {
	funcMap := template.FuncMap{
		"renderField":         r.renderFieldHelper,
		"renderCall":          r.renderCallHelper,
		"getChainName":        GetChainNameBySelector,
		"getChainNameOrEmpty": r.getChainNameOrEmpty,
		"hexEncode":           hexutil.Encode,
	}

	// Load templates from filesystem
	r.proposalTmpl = template.Must(template.New("proposal.tmpl").Funcs(funcMap).ParseFS(textTemplateFS, "templates/text/proposal.tmpl"))
	r.timelockTmpl = template.Must(template.New("timelock_proposal.tmpl").Funcs(funcMap).ParseFS(textTemplateFS, "templates/text/timelock_proposal.tmpl"))
	r.decodedCallTmpl = template.Must(template.New("decoded_call.tmpl").Funcs(funcMap).ParseFS(textTemplateFS, "templates/text/decoded_call.tmpl"))
	r.simpleFieldTmpl = template.Must(template.New("simple_field.tmpl").Funcs(funcMap).ParseFS(textTemplateFS, "templates/text/simple_field.tmpl"))
	r.addressFieldTmpl = template.Must(template.New("address_field.tmpl").Funcs(funcMap).ParseFS(textTemplateFS, "templates/text/address_field.tmpl"))
	r.bytesFieldTmpl = template.Must(template.New("bytes_field.tmpl").Funcs(funcMap).ParseFS(textTemplateFS, "templates/text/bytes_field.tmpl"))
	r.chainSelectorFieldTmpl = template.Must(template.New("chain_selector_field.tmpl").Funcs(funcMap).ParseFS(textTemplateFS, "templates/text/chain_selector_field.tmpl"))
	r.yamlFieldTmpl = template.Must(template.New("yaml_field.tmpl").Funcs(funcMap).ParseFS(textTemplateFS, "templates/text/yaml_field.tmpl"))
	r.arrayFieldTmpl = template.Must(template.New("array_field.tmpl").Funcs(funcMap).ParseFS(textTemplateFS, "templates/text/array_field.tmpl"))
	r.structFieldTmpl = template.Must(template.New("struct_field.tmpl").Funcs(funcMap).ParseFS(textTemplateFS, "templates/text/struct_field.tmpl"))
	r.namedFieldTmpl = template.Must(template.New("named_field.tmpl").Funcs(funcMap).ParseFS(textTemplateFS, "templates/text/named_field.tmpl"))
}

// Template data structures
type TextProposalTemplateData struct {
	Operations []TextOperationTemplateData
	Context    *FieldContext
}

type TextOperationTemplateData struct {
	ChainSelector uint64
	ChainName     string
	Calls         []*DecodedCall
}

type TextTimelockTemplateData struct {
	Batches []TextBatchTemplateData
	Context *FieldContext
}

type TextBatchTemplateData struct {
	ChainSelector uint64
	ChainName     string
	Operations    []TextOperationTemplateData
}

type TextDecodedCallTemplateData struct {
	Address string
	Method  string
	Inputs  []TextArgumentTemplateData
	Outputs []TextArgumentTemplateData
}

type TextArgumentTemplateData struct {
	Name    string
	Summary string
	Details string
}

// RenderDecodedCall renders a DecodedCall as plain text using templates
func (r *TextRenderer) RenderDecodedCall(d *DecodedCall, ctx *FieldContext) string {
	data := TextDecodedCallTemplateData{
		Address: d.Address,
		Method:  d.Method,
		Inputs:  make([]TextArgumentTemplateData, len(d.Inputs)),
		Outputs: make([]TextArgumentTemplateData, len(d.Outputs)),
	}

	// Process inputs
	for i, input := range d.Inputs {
		summary := r.renderFieldValue(input.Value)
		data.Inputs[i] = TextArgumentTemplateData{
			Name:    input.Name,
			Summary: summary,
		}
	}

	// Process outputs
	for i, output := range d.Outputs {
		summary := r.renderFieldValue(output.Value)
		data.Outputs[i] = TextArgumentTemplateData{
			Name:    output.Name,
			Summary: summary,
		}
	}

	var buf bytes.Buffer
	if err := r.decodedCallTmpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("Error rendering decoded call: %v", err)
	}

	return buf.String()
}

// RenderProposal renders a ProposalReport as plain text using templates
func (r *TextRenderer) RenderProposal(rep *ProposalReport, ctx *FieldContext) string {
	data := TextProposalTemplateData{
		Operations: make([]TextOperationTemplateData, len(rep.Operations)),
		Context:    ctx,
	}

	for i, op := range rep.Operations {
		data.Operations[i] = TextOperationTemplateData{
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

// RenderTimelockProposal renders a Timelock ProposalReport as plain text using templates
func (r *TextRenderer) RenderTimelockProposal(rep *ProposalReport, ctx *FieldContext) string {
	data := TextTimelockTemplateData{
		Batches: make([]TextBatchTemplateData, len(rep.Batches)),
		Context: ctx,
	}

	for i, batch := range rep.Batches {
		operations := make([]TextOperationTemplateData, len(batch.Operations))
		for j, op := range batch.Operations {
			operations[j] = TextOperationTemplateData{
				ChainSelector: op.ChainSelector,
				ChainName:     op.ChainName,
				Calls:         op.Calls,
			}
		}

		data.Batches[i] = TextBatchTemplateData{
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

// RenderField renders a NamedField as plain text
func (r *TextRenderer) RenderField(field NamedField, ctx *FieldContext) string {
	return fmt.Sprintf("%s: %s", field.Name, r.renderFieldValue(field.Value))
}

// renderFieldHelper is a template helper function for rendering fields
func (r *TextRenderer) renderFieldHelper(field FieldValue) string {
	return r.renderFieldValue(field)
}

// renderCallHelper is a template helper function for rendering DecodedCall
func (r *TextRenderer) renderCallHelper(call *DecodedCall, ctx *FieldContext) string {
	return r.RenderDecodedCall(call, ctx)
}

// getChainNameOrEmpty is a template helper function that safely gets chain name, returning empty string on error
func (r *TextRenderer) getChainNameOrEmpty(selector uint64) string {
	chainName, err := GetChainNameBySelector(selector)
	if err != nil || chainName == "" {
		return ""
	}

	return chainName
}

// renderFieldValue renders any field value as plain text using templates
func (r *TextRenderer) renderFieldValue(field FieldValue) string {
	var buf bytes.Buffer

	switch f := field.(type) {
	case AddressField:
		data := AddressFieldData{Value: f.GetValue()}
		if err := r.addressFieldTmpl.Execute(&buf, data); err != nil {
			return fmt.Sprintf("Error rendering address field: %v", err)
		}

		return buf.String()

	case ChainSelectorField:
		if err := r.chainSelectorFieldTmpl.Execute(&buf, f); err != nil {
			return fmt.Sprintf("Error rendering chain selector field: %v", err)
		}

		return buf.String()

	case BytesField:
		if err := r.bytesFieldTmpl.Execute(&buf, f); err != nil {
			return fmt.Sprintf("Error rendering bytes field: %v", err)
		}

		return buf.String()

	case ArrayField:
		if err := r.arrayFieldTmpl.Execute(&buf, f); err != nil {
			return fmt.Sprintf("Error rendering array field: %v", err)
		}

		return buf.String()

	case StructField:
		if err := r.structFieldTmpl.Execute(&buf, f); err != nil {
			return fmt.Sprintf("Error rendering struct field: %v", err)
		}

		return buf.String()

	case SimpleField:
		if err := r.simpleFieldTmpl.Execute(&buf, f); err != nil {
			return fmt.Sprintf("Error rendering simple field: %v", err)
		}

		return buf.String()

	case YamlField:
		if err := r.yamlFieldTmpl.Execute(&buf, f); err != nil {
			return fmt.Sprintf("Error rendering yaml field: %v", err)
		}

		return buf.String()

	case NamedField:
		if err := r.namedFieldTmpl.Execute(&buf, f); err != nil {
			return fmt.Sprintf("Error rendering named field: %v", err)
		}

		return buf.String()

	default:
		return fmt.Sprintf("<unknown field type: %s>", field.GetType())
	}
}

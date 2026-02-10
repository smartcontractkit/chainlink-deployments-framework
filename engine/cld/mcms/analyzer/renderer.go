package analyzer

import (
	"fmt"
	"strings"
)

func renderText(proposal *AnalyzedProposal, description string) string {
	var sb strings.Builder

	renderProposalHeader(&sb, description)

	renderAnnotations(&sb, proposal.Annotations(), "  ")

	for _, batchOp := range proposal.BatchOperations {
		renderBatch(&sb, batchOp)
	}

	return sb.String()
}

func renderProposalHeader(sb *strings.Builder, description string) {
	sb.WriteString(strings.Repeat("═", 64) + "\n")
	sb.WriteString(fmt.Sprintf("  Proposal: %s\n", description))
	sb.WriteString(strings.Repeat("═", 64) + "\n\n")
}

func renderBatch(sb *strings.Builder, batchOp *AnalyzedBatchOperation) {
	sb.WriteString(fmt.Sprintf("── %s (%d) ──\n\n", batchOp.ChainName, batchOp.ChainSelector))
	renderAnnotations(sb, batchOp.Annotations(), "  ")

	for _, call := range batchOp.Calls {
		renderCall(sb, call)
	}
}

func renderCall(sb *strings.Builder, call *AnalyzedCall) {
	sb.WriteString("  " + renderCallTarget(call) + "\n")
	sb.WriteString(fmt.Sprintf("  └─ %s\n", call.Name()))
	renderAnnotations(sb, call.Annotations(), "     ")

	for _, param := range call.AnalyzedInputs {
		renderAnnotations(sb, param.Annotations(), "       ")
	}

	sb.WriteString("\n")
}

func renderCallTarget(call *AnalyzedCall) string {
	if ct := call.ContractType(); ct != "" {
		return fmt.Sprintf("%s %s", ct, call.To())
	}

	return call.To()
}

func renderAnnotations(sb *strings.Builder, annotations Annotations, indent string) {
	if len(annotations) == 0 {
		return
	}

	type group struct {
		typeName string
		items    []Annotation
	}

	var groups []group
	seen := map[string]int{}

	for _, ann := range annotations {
		t := ann.Type()
		if idx, ok := seen[t]; ok {
			groups[idx].items = append(groups[idx].items, ann)
		} else {
			seen[t] = len(groups)
			groups = append(groups, group{typeName: t, items: []Annotation{ann}})
		}
	}

	for _, g := range groups {
		bulletIndent := indent
		if g.typeName != "" {
			sb.WriteString(fmt.Sprintf("%s%s:\n", indent, g.typeName))
			bulletIndent = indent + "  "
		}

		for _, ann := range g.items {
			if ann.Name() != "" {
				sb.WriteString(fmt.Sprintf("%s• %s: %s\n", bulletIndent, ann.Name(), ann.Value()))
			} else {
				sb.WriteString(fmt.Sprintf("%s• %s\n", bulletIndent, ann.Value()))
			}
		}
	}
}

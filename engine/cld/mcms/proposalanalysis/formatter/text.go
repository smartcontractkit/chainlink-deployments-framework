package formatter

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/types"
)

const (
	FormatterTextID = "text"
)

type TextFormatter struct{}

var _ types.Formatter = (*TextFormatter)(nil)

func (f *TextFormatter) ID() string { return FormatterTextID }

func (f *TextFormatter) Format(_ context.Context, w io.Writer, _ types.FormatterRequest, proposal types.AnalyzedProposal) error {
	out := renderText(proposal)
	_, err := io.WriteString(w, out)
	return err
}

func renderText(proposal types.AnalyzedProposal) string {
	var sb strings.Builder
	for _, batchOp := range proposal.BatchOperations() {
		fmt.Fprintf(&sb, "── %s (%d) ──\n\n", batchOp.ChainName(), batchOp.ChainSelector())
		renderAnnotations(&sb, batchOp.Annotations(), "  ")
		for _, call := range batchOp.Calls() {
			renderCall(&sb, call)
		}
	}
	return sb.String()
}

func renderCall(sb *strings.Builder, call types.AnalyzedCall) {
	sb.WriteString("  " + renderCallTarget(call) + "\n")
	fmt.Fprintf(sb, "  └─ %s\n", call.Name())
	renderAnnotations(sb, call.Annotations(), "     ")
	for _, param := range call.Inputs() {
		renderAnnotations(sb, param.Annotations(), "       ")
	}
	sb.WriteString("\n")
}

func renderCallTarget(call types.AnalyzedCall) string {
	if ct := call.ContractType(); ct != "" {
		return ct
	}
	return "<unknown-contract>"
}

func renderAnnotations(sb *strings.Builder, annotations types.Annotations, indent string) {
	if len(annotations) == 0 {
		return
	}

	for _, ann := range annotations {
		if ann.Type() != "" {
			if ann.Name() != "" {
				fmt.Fprintf(sb, "%s• %s %s: %v\n", indent, ann.Type(), ann.Name(), ann.Value())
			} else {
				fmt.Fprintf(sb, "%s• %s: %v\n", indent, ann.Type(), ann.Value())
			}
			continue
		}
		if ann.Name() != "" {
			fmt.Fprintf(sb, "%s• %s: %v\n", indent, ann.Name(), ann.Value())
		} else {
			fmt.Fprintf(sb, "%s• %v\n", indent, ann.Value())
		}
	}
}

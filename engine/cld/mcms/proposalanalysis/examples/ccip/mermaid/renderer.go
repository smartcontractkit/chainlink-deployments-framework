package mermaid

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	chainutils "github.com/smartcontractkit/chainlink-deployments-framework/chain/utils"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotation"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/renderer"
)

const IDMermaid = "mermaid"

type MermaidRenderer struct{}

var _ renderer.Renderer = (*MermaidRenderer)(nil)

func NewMermaidRenderer() *MermaidRenderer { return &MermaidRenderer{} }

func (r *MermaidRenderer) ID() string { return IDMermaid }

func (r *MermaidRenderer) RenderTo(w io.Writer, _ renderer.RenderRequest, proposal analyzer.AnalyzedProposal) error {
	var b strings.Builder

	b.WriteString("graph TD\n")
	b.WriteString("    classDef registry fill:#e1f5fe,stroke:#01579b,stroke-width:2px\n")
	b.WriteString("    classDef pool fill:#d4edda,stroke:#28a745,stroke-width:2px\n")
	b.WriteString("    classDef remote fill:#fff3cd,stroke:#ffc107,stroke-width:2px\n")
	b.WriteString("    classDef default fill:#f5f5f5,stroke:#666,stroke-width:1px\n")

	batches := proposal.BatchOperations()
	nodes, nodeID := collectNodes(batches)

	rendered := make(map[uint64]bool)
	for _, batch := range batches {
		sel := batch.ChainSelector()
		if rendered[sel] {
			continue
		}

		rendered[sel] = true

		name := resolveChainSelector(sel)
		id := sanitizeID(name)
		b.WriteString(fmt.Sprintf("    subgraph %s [\"%s\"]\n", id, escapeQuotes(name)))

		for _, n := range nodes {
			if n.chainSelector != sel {
				continue
			}

			b.WriteString(fmt.Sprintf("        %s[\"%s\"]:::%s\n", n.id, escapeQuotes(n.label), contractStyle(n.contractType)))
		}

		b.WriteString("    end\n")
	}

	step := 0
	for _, batch := range batches {
		sel := batch.ChainSelector()
		var prevID string
		for _, call := range batch.Calls() {
			step++
			curID := nodeID[contractKey{sel, call.To()}]
			from := curID
			if prevID != "" {
				from = prevID
			}

			b.WriteString(fmt.Sprintf("    %s -->|\"%d. %s\"| %s\n", from, step, escapeQuotes(call.Name()), curID))
			prevID = curID
		}
	}

	for _, src := range nodes {
		for _, ann := range src.annotations {
			if ann.Name() != "ccip.chain_update" {
				continue
			}

			remoteSel := extractChainSelector(ann)
			if remoteSel == 0 {
				continue
			}

			for j := range nodes {
				if nodes[j].chainSelector == remoteSel && nodes[j].id != src.id {
					b.WriteString(fmt.Sprintf("    %s -->|\"%s\"| %s\n", src.id, "chain update", nodes[j].id))

					break
				}
			}
		}
	}

	_, err := io.WriteString(w, b.String())

	return err
}

type contractKey struct {
	chain   uint64
	address string
}

type contractNode struct {
	id            string
	label         string
	contractType  string
	chainSelector uint64
	annotations   annotation.Annotations
}

func collectNodes(batches analyzer.AnalyzedBatchOperations) ([]contractNode, map[contractKey]string) {
	seen := make(map[contractKey]int)
	idMap := make(map[contractKey]string)
	var nodes []contractNode

	for _, batch := range batches {
		sel := batch.ChainSelector()

		for _, call := range batch.Calls() {
			k := contractKey{sel, call.To()}

			if idx, ok := seen[k]; ok {
				nodes[idx].annotations = append(nodes[idx].annotations, call.Annotations()...)

				continue
			}

			id := fmt.Sprintf("n%d", len(nodes)+1)
			label := buildLabel(call)

			seen[k] = len(nodes)
			idMap[k] = id
			nodes = append(nodes, contractNode{
				id:            id,
				label:         label,
				contractType:  call.ContractType(),
				chainSelector: sel,
				annotations:   call.Annotations(),
			})
		}
	}

	return nodes, idMap
}

func buildLabel(call analyzer.AnalyzedCall) string {
	var parts []string

	ct := call.ContractType()
	if ct != "" {
		if cv := call.ContractVersion(); cv != "" {
			ct += " " + cv
		}

		parts = append(parts, ct)
	}

	parts = append(parts, truncateAddress(call.To()))

	for _, ann := range call.Annotations() {
		if ann.Name() == "ccip.token.symbol" {
			parts = append(parts, fmt.Sprintf("%v", ann.Value()))

			break
		}
	}

	return strings.Join(parts, "<br/>")
}

func contractStyle(ct string) string {
	lower := strings.ToLower(ct)
	if strings.Contains(lower, "registry") {
		return "registry"
	}

	if strings.Contains(lower, "pool") {
		return "pool"
	}

	return "default"
}

var chainSelectorInParens = regexp.MustCompile(`\((\d+)\)`)

func extractChainSelector(ann annotation.Annotation) uint64 {
	matches := chainSelectorInParens.FindStringSubmatch(fmt.Sprintf("%v", ann.Value()))
	if len(matches) < 2 {
		return 0
	}

	sel, err := strconv.ParseUint(matches[1], 10, 64)
	if err != nil {
		return 0
	}

	return sel
}

func resolveChainSelector(sel uint64) string {
	info, err := chainutils.ChainInfo(sel)
	if err != nil {
		return "chain-" + strconv.FormatUint(sel, 10)
	}

	return info.ChainName
}

func truncateAddress(addr string) string {
	if strings.HasPrefix(addr, "0x") && len(addr) > 12 {
		return addr[:6] + ".." + addr[len(addr)-4:]
	}
	if len(addr) > 12 {
		return addr[:4] + ".." + addr[len(addr)-3:]
	}

	return addr
}

func sanitizeID(s string) string {
	return strings.NewReplacer("-", "_", " ", "_").Replace(s)
}

func escapeQuotes(s string) string {
	return strings.ReplaceAll(s, "\"", "#quot;")
}
